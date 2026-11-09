package domain

import (
	"context"
	"database/sql"
	"time"
)

type NotionRepository struct{ DB *sql.DB }

func (r *NotionRepository) CreateState(ctx context.Context, user string, hash []byte, expires time.Time) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO notion_link_users (user_id) VALUES ($1) ON CONFLICT DO NOTHING`, user); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `SELECT user_id FROM notion_link_users WHERE user_id=$1 FOR UPDATE`, user); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO notion_oauth_states (state_hash,user_id,generation,expires_at) SELECT $1,user_id,generation,$3 FROM notion_link_users WHERE user_id=$2`, hash, user, expires); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *NotionRepository) ConsumeState(ctx context.Context, hash []byte, now time.Time) (string, int64, error) {
	var user string
	var generation int64
	err := r.DB.QueryRowContext(ctx, `DELETE FROM notion_oauth_states WHERE state_hash=$1 AND expires_at>$2 RETURNING user_id,generation`, hash, now).Scan(&user, &generation)
	return user, generation, err
}
func (r *NotionRepository) SaveConnection(ctx context.Context, user string, generation int64, name string, token []byte) error {
	result, err := r.DB.ExecContext(ctx, `UPDATE notion_link_users SET account_name=$3,encrypted_token=$4 WHERE user_id=$1 AND generation=$2`, user, generation, name, token)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func (r *NotionRepository) Connection(ctx context.Context, user string) (NotionConnection, error) {
	var name sql.NullString
	err := r.DB.QueryRowContext(ctx, `SELECT account_name FROM notion_link_users WHERE user_id=$1 AND encrypted_token IS NOT NULL`, user).Scan(&name)
	if err == sql.ErrNoRows {
		return NotionConnection{}, nil
	}
	if err != nil {
		return NotionConnection{}, err
	}
	c := NotionConnection{Linked: true}
	if name.Valid && name.String != "" {
		c.AccountName = &name.String
	}
	return c, nil
}
func (r *NotionRepository) Disconnect(ctx context.Context, user string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO notion_link_users(user_id,generation) VALUES($1,1) ON CONFLICT(user_id) DO UPDATE SET generation=notion_link_users.generation+1, account_name=NULL, encrypted_token=NULL`, user); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM notion_oauth_states WHERE user_id=$1`, user); err != nil {
		return err
	}
	return tx.Commit()
}
