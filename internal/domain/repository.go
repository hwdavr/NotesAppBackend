package domain

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) CreateItem(ctx context.Context, userID string, input CreateItemInput) (Item, error) {
	var item Item
	err := r.DB.GetContext(ctx, &item, `
		INSERT INTO items (id, user_id, type, parent_id, name, content, sort_key, device_id, version, last_synced_version)
		VALUES (COALESCE(NULLIF($1, ''), uuid_generate_v4()::text), $2, $3, $4, $5, $6, $7, $8, 1, 0)
		RETURNING id, user_id, type, parent_id, name, content, sort_key, version, device_id, last_synced_version, deleted_at, created_at, updated_at
	`, input.ID, userID, input.Type, input.ParentID, input.Name, input.Content, input.SortKey, input.DeviceID)
	return item, err
}

func (r *Repository) ListItems(ctx context.Context, userID string, filter ListItemsFilter) ([]Item, error) {
	query := `
		SELECT id, user_id, type, parent_id, name, content, sort_key, version, device_id, last_synced_version, deleted_at, created_at, updated_at
		FROM items
		WHERE user_id = $1
	`
	args := []any{userID}
	next := 2

	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", next)
		args = append(args, filter.Type)
		next++
	}
	if filter.ParentID != nil {
		query += fmt.Sprintf(" AND parent_id = $%d", next)
		args = append(args, *filter.ParentID)
		next++
	} else if filter.RootOnly {
		query += " AND parent_id IS NULL"
	}
	if !filter.IncludeDeleted {
		query += " AND deleted_at IS NULL"
	}
	if filter.SinceVersion != nil {
		query += fmt.Sprintf(" AND version > $%d", next)
		args = append(args, *filter.SinceVersion)
		next++
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR content ILIKE $%d)", next, next)
		args = append(args, "%"+q+"%")
		next++
	}

	query += " ORDER BY parent_id NULLS FIRST, sort_key ASC, updated_at DESC"

	var items []Item
	err := r.DB.SelectContext(ctx, &items, query, args...)
	return items, err
}

func (r *Repository) GetItem(ctx context.Context, userID, itemID string) (Item, error) {
	var item Item
	err := r.DB.GetContext(ctx, &item, `
		SELECT id, user_id, type, parent_id, name, content, sort_key, version, device_id, last_synced_version, deleted_at, created_at, updated_at
		FROM items
		WHERE id = $1 AND user_id = $2
	`, itemID, userID)
	if err == sql.ErrNoRows {
		return Item{}, ErrItemNotFound
	}
	return item, err
}

func (r *Repository) UpdateItem(ctx context.Context, userID, itemID string, input UpdateItemInput) (Item, error) {
	var item Item
	err := r.DB.GetContext(ctx, &item, `
		UPDATE items
		SET
			name = COALESCE($3, name),
			content = COALESCE($4, content),
			parent_id = CASE
				WHEN $5 THEN NULL
				WHEN $6::text IS NOT NULL THEN $6::text
				ELSE parent_id
			END,
			sort_key = COALESCE($7, sort_key),
			deleted_at = CASE
				WHEN $8 THEN NULL
				WHEN $9::timestamptz IS NOT NULL THEN $9::timestamptz
				ELSE deleted_at
			END,
			device_id = $10,
			last_synced_version = $11,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, user_id, type, parent_id, name, content, sort_key, version, device_id, last_synced_version, deleted_at, created_at, updated_at
	`, itemID, userID, input.Name, input.Content, input.ClearParentID, input.ParentID, input.SortKey, input.ClearDeletedAt, input.DeletedAt, input.DeviceID, input.LastSyncedVersion)
	if err == sql.ErrNoRows {
		return Item{}, ErrItemNotFound
	}
	return item, err
}

func (r *Repository) FolderHasDescendant(ctx context.Context, userID, folderID, candidateID string) (bool, error) {
	var exists bool
	err := r.DB.GetContext(ctx, &exists, `
		WITH RECURSIVE descendants AS (
			SELECT id, parent_id
			FROM items
			WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL
			UNION ALL
			SELECT i.id, i.parent_id
			FROM items i
			INNER JOIN descendants d ON i.parent_id = d.id
			WHERE i.user_id = $2 AND i.deleted_at IS NULL
		)
		SELECT EXISTS(SELECT 1 FROM descendants WHERE id = $3)
	`, folderID, userID, candidateID)
	return exists, err
}

func (r *Repository) SoftDeleteItemTree(ctx context.Context, userID, itemID, deviceID string, lastSyncedVersion int64, deletedAt time.Time) error {
	result, err := r.DB.ExecContext(ctx, `
		WITH RECURSIVE subtree AS (
			SELECT id
			FROM items
			WHERE id = $1 AND user_id = $2
			UNION ALL
			SELECT i.id
			FROM items i
			INNER JOIN subtree s ON i.parent_id = s.id
			WHERE i.user_id = $2 AND i.deleted_at IS NULL
		)
		UPDATE items
		SET
			deleted_at = $5,
			device_id = $3,
			last_synced_version = $4,
			version = version + 1,
			updated_at = NOW()
		WHERE id IN (SELECT id FROM subtree) AND user_id = $2
	`, itemID, userID, deviceID, lastSyncedVersion, deletedAt)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}
