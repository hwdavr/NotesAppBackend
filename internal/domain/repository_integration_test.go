//go:build integration

package domain

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/hwdavr/notes-app-backend/internal/db"
	"github.com/hwdavr/notes-app-backend/internal/pkg/email"
)

const (
	integrationUserID   = "integration-user"
	integrationFolderID = "integration-folder"
	integrationNoteID   = "integration-note"
)

func TestPostgresIntegrationItemLifecycle(t *testing.T) {
	database := openIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clearIntegrationData(t, ctx, database)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		clearIntegrationData(t, cleanupContext, database)
	})

	service := NewService(NewRepository(database), email.NewMockService())
	folder, err := service.CreateFolder(ctx, integrationUserID, "", CreateItemInput{
		ID:       integrationFolderID,
		Name:     "Integration Folder",
		SortKey:  "a0",
		DeviceID: "integration-device",
	})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if folder.Version != 1 || folder.Type != ItemTypeFolder {
		t.Fatalf("created folder = %#v, want version 1 and type %q", folder, ItemTypeFolder)
	}

	note, err := service.CreateNote(ctx, integrationUserID, "", CreateItemInput{
		ID:       integrationNoteID,
		ParentID: &folder.ID,
		Name:     "Integration Note",
		Content:  "initial content",
		SortKey:  "a0",
		DeviceID: "integration-device",
	})
	if err != nil {
		t.Fatalf("create note: %v", err)
	}
	if note.ParentID == nil || *note.ParentID != folder.ID {
		t.Fatalf("created note parent = %v, want %q", note.ParentID, folder.ID)
	}

	rootItems, err := service.ListItems(ctx, integrationUserID, "", ListItemsFilter{RootOnly: true})
	if err != nil {
		t.Fatalf("list root items: %v", err)
	}
	if len(rootItems) != 1 || rootItems[0].ID != folder.ID {
		t.Fatalf("root items = %#v, want only folder %q", rootItems, folder.ID)
	}

	childItems, err := service.ListItems(ctx, integrationUserID, "", ListItemsFilter{ParentID: &folder.ID})
	if err != nil {
		t.Fatalf("list child items: %v", err)
	}
	if len(childItems) != 1 || childItems[0].ID != note.ID || childItems[0].Content != "initial content" {
		t.Fatalf("child items did not contain note %q with its initial content", note.ID)
	}

	mutation, err := service.UpdateNoteContent(ctx, integrationUserID, "", note.ID, "updated content", "integration-device", note.Version)
	if err != nil {
		t.Fatalf("update note content: %v", err)
	}
	if mutation.Status != "merged" || mutation.Item.Content != "updated content" || mutation.Item.Version != note.Version+1 {
		t.Fatalf("content mutation was not a merged update at version %d", note.Version+1)
	}

	deleted, err := service.DeleteItem(ctx, integrationUserID, "", folder.ID, "integration-device", folder.Version)
	if err != nil {
		t.Fatalf("delete folder tree: %v", err)
	}
	if deleted.Item.DeletedAt == nil {
		t.Fatalf("deleted folder = %#v, want tombstone", deleted.Item)
	}

	visibleItems, err := service.ListItems(ctx, integrationUserID, "", ListItemsFilter{RootOnly: true})
	if err != nil {
		t.Fatalf("list visible items after delete: %v", err)
	}
	if len(visibleItems) != 0 {
		t.Fatalf("visible items after delete = %#v, want empty list", visibleItems)
	}

	tombstones, err := service.ListItems(ctx, integrationUserID, "", ListItemsFilter{
		RootOnly:       true,
		IncludeDeleted: true,
	})
	if err != nil {
		t.Fatalf("list deleted items: %v", err)
	}
	if len(tombstones) != 1 || tombstones[0].ID != folder.ID || tombstones[0].DeletedAt == nil {
		t.Fatalf("tombstones = %#v, want deleted folder %q", tombstones, folder.ID)
	}
}

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL must point to the disposable integration database")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("connect to integration database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func clearIntegrationData(t *testing.T, ctx context.Context, database *sql.DB) {
	t.Helper()

	queries := []string{
		`DELETE FROM note_block_comments WHERE note_id IN (SELECT id FROM items WHERE user_id = $1)`,
		`DELETE FROM note_shares WHERE note_id IN (SELECT id FROM items WHERE user_id = $1)`,
		`DELETE FROM items WHERE user_id = $1`,
	}
	for _, query := range queries {
		if _, err := database.ExecContext(ctx, query, integrationUserID); err != nil {
			t.Fatalf("clear integration data: %v", err)
		}
	}
}
