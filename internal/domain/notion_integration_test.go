//go:build integration

package domain

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"
)

func TestPostgresIntegrationNotion(t *testing.T) {
	db := openIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	user := "notion-integration-user"
	other := "notion-integration-other"
	cleanup := func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if _, err := db.ExecContext(cleanupCtx, `DELETE FROM notion_link_users WHERE user_id IN ($1,$2)`, user, other); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	repo := &NotionRepository{DB: db}
	hash := sha256.Sum256([]byte("notion-test-state"))
	now := time.Now()
	if err := repo.CreateState(ctx, user, hash[:], now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	got, generation, err := repo.ConsumeState(ctx, hash[:], now)
	if err != nil || got != user {
		t.Fatalf("consume = %q, %v", got, err)
	}
	if _, _, err = repo.ConsumeState(ctx, hash[:], now); err != sql.ErrNoRows {
		t.Fatalf("state replay: %v", err)
	}
	if err = repo.SaveConnection(ctx, user, generation, "Workspace", []byte("encrypted")); err != nil {
		t.Fatal(err)
	}
	c, err := repo.Connection(ctx, user)
	if err != nil || !c.Linked || c.AccountName == nil || *c.AccountName != "Workspace" {
		t.Fatalf("connection = %#v, %v", c, err)
	}
	c, err = repo.Connection(ctx, other)
	if err != nil || c.Linked {
		t.Fatalf("other connection = %#v, %v", c, err)
	}
	next := sha256.Sum256([]byte("notion-test-second"))
	if err = repo.CreateState(ctx, user, next[:], now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	disconnected := make(chan struct{})
	staleResult := make(chan error, 1)
	go func() {
		close(ready)
		<-disconnected
		staleResult <- repo.SaveConnection(ctx, user, generation, "stale", []byte("encrypted"))
	}()
	<-ready
	if err = repo.Disconnect(ctx, user); err != nil {
		close(disconnected)
		t.Fatal(err)
	}
	close(disconnected)
	if err = <-staleResult; err != sql.ErrNoRows {
		t.Fatalf("in-flight callback restored connection: %v", err)
	}
	if _, _, err = repo.ConsumeState(ctx, next[:], now); err != sql.ErrNoRows {
		t.Fatalf("pending state survived disconnect: %v", err)
	}
	if err = repo.SaveConnection(ctx, user, generation, "stale", []byte("encrypted")); err != sql.ErrNoRows {
		t.Fatalf("stale callback restored connection: %v", err)
	}
	c, err = repo.Connection(ctx, user)
	if err != nil || c.Linked {
		t.Fatalf("disconnect = %#v, %v", c, err)
	}
	expired := sha256.Sum256([]byte("notion-test-expired"))
	if err = repo.CreateState(ctx, user, expired[:], now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err = repo.ConsumeState(ctx, expired[:], now); err != sql.ErrNoRows {
		t.Fatalf("expired state accepted: %v", err)
	}
}

func TestPostgresIntegrationNotionDisconnectRollback(t *testing.T) {
	db := openIntegrationDatabase(t)
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	user := "notion-rollback-test"
	hash := sha256.Sum256([]byte("rollback-state"))
	repo := &NotionRepository{DB: db}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_, err := db.ExecContext(cleanup, `DROP TRIGGER IF EXISTS notion_test_block_delete ON notion_oauth_states`)
		if err != nil {
			t.Errorf("drop trigger: %v", err)
		}
		_, err = db.ExecContext(cleanup, `DELETE FROM notion_link_users WHERE user_id = $1`, user)
		if err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	if err := repo.CreateState(ctx, user, hash[:], time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveConnection(ctx, user, 0, "Workspace", []byte("encrypted")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE FUNCTION pg_temp.notion_test_fail_delete() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic failure'; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TRIGGER notion_test_block_delete BEFORE DELETE ON notion_oauth_states FOR EACH ROW EXECUTE FUNCTION pg_temp.notion_test_fail_delete()`); err != nil {
		t.Fatal(err)
	}
	if err := repo.Disconnect(ctx, user); err == nil {
		t.Fatal("expected disconnect failure")
	}
	c, err := repo.Connection(ctx, user)
	if err != nil || !c.Linked {
		t.Fatalf("connection not rolled back: %#v %v", c, err)
	}
	var generation int
	var count int
	if err = db.QueryRowContext(ctx, `SELECT generation FROM notion_link_users WHERE user_id=$1`, user).Scan(&generation); err != nil || generation != 0 {
		t.Fatalf("generation not rolled back: %d %v", generation, err)
	}
	if err = db.QueryRowContext(ctx, `SELECT count(*) FROM notion_oauth_states WHERE user_id=$1`, user).Scan(&count); err != nil || count != 1 {
		t.Fatalf("state not preserved: %d %v", count, err)
	}
}
