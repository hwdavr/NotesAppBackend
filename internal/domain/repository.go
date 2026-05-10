package domain

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/hwdavr/notes-app-backend/internal/db/models"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type Repository struct {
	DB bob.Executor
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: bob.NewDB(db)}
}

func (r *Repository) CreateItem(ctx context.Context, userID string, input CreateItemInput) (Item, error) {
	setter := &models.ItemSetter{
		ID:                omit.From(input.ID),
		UserID:            omit.From(userID),
		Type:              omit.From(input.Type),
		ParentID:          omitnull.FromPtr(input.ParentID),
		Name:              omit.From(input.Name),
		Content:           omit.From(input.Content),
		SortKey:           omit.From(input.SortKey),
		DeviceID:          omit.From(input.DeviceID),
		Version:           omit.From(int64(1)),
		LastSyncedVersion: omit.From(int64(0)),
	}

	if input.ID == "" {
		setter.ID = omit.Val[string]{}
	}

	item, err := models.Items.Insert(ctx, r.DB, setter)
	if err != nil {
		return Item{}, err
	}

	return mapModelToItem(item), nil
}

func (r *Repository) ListItems(ctx context.Context, userID, userEmail string, filter ListItemsFilter) ([]Item, error) {
	query := `
		WITH RECURSIVE accessible AS (
			-- Direct access (owned or shared)
			SELECT i.id, i.user_id, i.parent_id, 
			       COALESCE(ns.access_role, 'full_access') as access_role,
			       (i.user_id != $1) as is_shared,
			       (i.parent_id IS NULL) as is_effective_root
			FROM items i
			LEFT JOIN note_shares ns ON i.id = ns.note_id AND ns.email = $2 AND ns.status IN ('active', 'pending')
			WHERE (i.user_id = $1 OR ns.id IS NOT NULL)
			
			UNION ALL
			
			-- Inherited access
			SELECT i.id, i.user_id, i.parent_id, 
			       a.access_role,
			       true as is_shared,
			       false as is_effective_root
			FROM items i
			JOIN accessible a ON i.parent_id = a.id
			WHERE i.user_id != $1
		)
		SELECT * FROM (
			SELECT DISTINCT ON (i.id)
				i.*, 
				a.access_role,
				a.is_shared,
				a.is_effective_root
			FROM items i
			JOIN accessible a ON i.id = a.id
			ORDER BY i.id, a.access_role DESC -- full_access before read_only
		) i
		WHERE 1=1
	`
	args := []any{userID, userEmail}

	// Add filters
	if filter.Type != "" {
		query += " AND i.type = $" + strconv.Itoa(len(args)+1)
		args = append(args, filter.Type)
	}

	if filter.ParentID != nil {
		query += " AND i.parent_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, *filter.ParentID)
	} else if filter.RootOnly {
		query += " AND i.is_effective_root = TRUE"
	}

	if !filter.IncludeDeleted {
		query += " AND i.deleted_at IS NULL"
	}

	if filter.SinceVersion != nil {
		query += " AND i.version > $" + strconv.Itoa(len(args)+1)
		args = append(args, *filter.SinceVersion)
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		pattern := "%" + q + "%"
		query += " AND (i.name ILIKE $" + strconv.Itoa(len(args)+1) + " OR i.content ILIKE $" + strconv.Itoa(len(args)+1) + ")"
		args = append(args, pattern)
	}

	query += " ORDER BY i.parent_id ASC NULLS FIRST, i.sort_key ASC, i.updated_at DESC"

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Item
	for rows.Next() {
		var m models.Item
		var accessRole string
		var isShared bool
		
		// Scan all columns from models.Item plus our extra columns
		// Since we use SELECT i.*, we need to be careful.
		// It's safer to use Bob's scanning if possible, but for raw queries:
		err := rows.Scan(
			&m.ID, &m.UserID, &m.Type, &m.ParentID, &m.Name, &m.Content, &m.SortKey,
			&m.Version, &m.DeviceID, &m.LastSyncedVersion, &m.DeletedAt, &m.CreatedAt, &m.UpdatedAt, &m.IsFavorite,
			&accessRole, &isShared,
		)
		if err != nil {
			return nil, err
		}

		domainItem := mapModelToItem(&m)
		domainItem.IsShared = isShared
		domainItem.AccessRole = accessRole
		result = append(result, domainItem)
	}
	return result, nil
}

func (r *Repository) GetItem(ctx context.Context, userID, userEmail, itemID string) (Item, error) {
	query := `
		WITH RECURSIVE accessible AS (
			SELECT i.id, i.user_id, i.parent_id, 
			       COALESCE(ns.access_role, 'full_access') as access_role,
			       (i.user_id != $1) as is_shared
			FROM items i
			LEFT JOIN note_shares ns ON i.id = ns.note_id AND ns.email = $2 AND ns.status IN ('active', 'pending')
			WHERE (i.user_id = $1 OR ns.id IS NOT NULL)
			
			UNION ALL
			
			SELECT i.id, i.user_id, i.parent_id, 
			       a.access_role,
			       true as is_shared
			FROM items i
			JOIN accessible a ON i.parent_id = a.id
			WHERE i.user_id != $1
		)
		SELECT 
			i.*, 
			a.access_role,
			a.is_shared
		FROM items i
		JOIN accessible a ON i.id = a.id
		WHERE i.id = $3
		ORDER BY a.access_role DESC -- full_access before read_only
		LIMIT 1
	`
	rows, err := r.DB.QueryContext(ctx, query, userID, userEmail, itemID)
	if err != nil {
		return Item{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return Item{}, ErrItemNotFound
	}

	var m models.Item
	var accessRole string
	var isShared bool
	err = rows.Scan(
		&m.ID, &m.UserID, &m.Type, &m.ParentID, &m.Name, &m.Content, &m.SortKey,
		&m.Version, &m.DeviceID, &m.LastSyncedVersion, &m.DeletedAt, &m.CreatedAt, &m.UpdatedAt, &m.IsFavorite,
		&accessRole, &isShared,
	)
	if err != nil {
		return Item{}, err
	}

	domainItem := mapModelToItem(&m)
	domainItem.IsShared = isShared
	domainItem.AccessRole = accessRole
	return domainItem, nil
}

func (r *Repository) UpdateItem(ctx context.Context, userID, userEmail, itemID string, input UpdateItemInput) (Item, error) {
	setter := &models.ItemSetter{
		DeviceID:          omit.From(input.DeviceID),
		LastSyncedVersion: omit.From(input.LastSyncedVersion),
		UpdatedAt:         omit.From(time.Now()),
	}

	if input.Name != nil {
		setter.Name = omit.From(*input.Name)
	}
	if input.Content != nil {
		setter.Content = omit.From(*input.Content)
	}
	if input.SortKey != nil {
		setter.SortKey = omit.From(*input.SortKey)
	}

	if input.ClearParentID {
		setter.ParentID = omitnull.FromPtr((*string)(nil))
	} else if input.ParentID != nil {
		setter.ParentID = omitnull.From(*input.ParentID)
	}

	if input.ClearDeletedAt {
		setter.DeletedAt = omitnull.FromPtr((*time.Time)(nil))
	} else if input.DeletedAt != nil {
		setter.DeletedAt = omitnull.From(*input.DeletedAt)
	}
	if input.IsFavorite != nil {
		setter.IsFavorite = omit.From(*input.IsFavorite)
	}

	// Update only if owner OR has active full_access share
	where := psql.And(
		models.ItemColumns.ID.EQ(psql.Arg(itemID)),
		psql.Or(
			models.ItemColumns.UserID.EQ(psql.Arg(userID)),
			psql.Raw(`id IN (
				WITH RECURSIVE full_access_items AS (
					SELECT note_id as id FROM note_shares 
					WHERE email = ? AND access_role = 'full_access' AND status IN ('active', 'pending')
					UNION ALL
					SELECT i.id FROM items i
					JOIN full_access_items fai ON i.parent_id = fai.id
				)
				SELECT id FROM full_access_items
			)`, userEmail),
		),
	)

	_, err := psql.Update(
		um.Table(models.TableNames.Items),
		setter,
		um.Where(where),
		um.Set(psql.Raw("version = version + 1")),
	).Exec(ctx, r.DB)

	if err != nil {
		return Item{}, err
	}

	return r.GetItem(ctx, userID, userEmail, itemID)
}

func (r *Repository) FolderHasDescendant(ctx context.Context, userID, folderID, candidateID string) (bool, error) {
	query := `
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
	`
	// Correct way to use RawQuery with Scan in Bob
	var exists bool
	rows, err := r.DB.QueryContext(ctx, query, folderID, userID, candidateID)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if rows.Next() {
		if err := rows.Scan(&exists); err != nil {
			return false, err
		}
	}
	return exists, nil
}

func (r *Repository) SoftDeleteItemTree(ctx context.Context, userID, itemID, deviceID string, lastSyncedVersion int64, deletedAt time.Time) error {
	query := `
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
	`
	_, err := r.DB.ExecContext(ctx, query, itemID, userID, deviceID, lastSyncedVersion, deletedAt)
	return err
}

func mapModelToItem(m *models.Item) Item {
	return Item{
		ID:                m.ID,
		UserID:            m.UserID,
		Type:              m.Type,
		ParentID:          m.ParentID.Ptr(),
		Name:              m.Name,
		Content:           m.Content,
		SortKey:           m.SortKey,
		Version:           m.Version,
		DeviceID:          m.DeviceID,
		LastSyncedVersion: m.LastSyncedVersion,
		DeletedAt:         m.DeletedAt.Ptr(),
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
		IsFavorite:        m.IsFavorite,
	}
}

func (r *Repository) CreateNoteShare(ctx context.Context, noteID, email, accessRole, status, invitedBy string) (NoteShare, error) {
	query := `
		INSERT INTO note_shares (note_id, email, access_role, status, invited_by_user_id, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, note_id, email, access_role, status, invited_by_user_id, created_at, updated_at
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, email, accessRole, status, invitedBy)
	if err != nil {
		return NoteShare{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteShare{}, sql.ErrNoRows
	}

	var s NoteShare
	err = rows.Scan(
		&s.ID, &s.NoteID, &s.Email, &s.AccessRole, &s.Status, &s.InvitedByUserID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return NoteShare{}, err
	}
	return s, nil
}

func (r *Repository) ListNoteShares(ctx context.Context, noteID string) ([]NoteShare, error) {
	query := `
		SELECT id, note_id, email, access_role, status, invited_by_user_id, created_at, updated_at
		FROM note_shares
		WHERE note_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []NoteShare
	for rows.Next() {
		var s NoteShare
		if err := rows.Scan(&s.ID, &s.NoteID, &s.Email, &s.AccessRole, &s.Status, &s.InvitedByUserID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, nil
}

func (r *Repository) GetNoteShare(ctx context.Context, noteID, shareID string) (NoteShare, error) {
	query := `
		SELECT id, note_id, email, access_role, status, invited_by_user_id, created_at, updated_at
		FROM note_shares
		WHERE note_id = $1 AND id = $2
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, shareID)
	if err != nil {
		return NoteShare{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteShare{}, ErrItemNotFound
	}

	var s NoteShare
	err = rows.Scan(
		&s.ID, &s.NoteID, &s.Email, &s.AccessRole, &s.Status, &s.InvitedByUserID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return NoteShare{}, err
	}
	return s, nil
}

func (r *Repository) GetNoteShareByNoteAndEmail(ctx context.Context, noteID, email string) (NoteShare, error) {
	query := `
		SELECT id, note_id, email, access_role, status, invited_by_user_id, created_at, updated_at
		FROM note_shares
		WHERE note_id = $1 AND email = $2
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, email)
	if err != nil {
		return NoteShare{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteShare{}, ErrItemNotFound
	}

	var s NoteShare
	err = rows.Scan(
		&s.ID, &s.NoteID, &s.Email, &s.AccessRole, &s.Status, &s.InvitedByUserID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return NoteShare{}, err
	}
	return s, nil
}

func (r *Repository) UpdateNoteShare(ctx context.Context, noteID, shareID, accessRole string) (NoteShare, error) {
	query := `
		UPDATE note_shares
		SET access_role = $3, updated_at = NOW()
		WHERE note_id = $1 AND id = $2
		RETURNING id, note_id, email, access_role, status, invited_by_user_id, created_at, updated_at
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, shareID, accessRole)
	if err != nil {
		return NoteShare{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteShare{}, ErrItemNotFound
	}

	var s NoteShare
	err = rows.Scan(
		&s.ID, &s.NoteID, &s.Email, &s.AccessRole, &s.Status, &s.InvitedByUserID, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return NoteShare{}, err
	}
	return s, nil
}

func (r *Repository) DeleteNoteShare(ctx context.Context, noteID, shareID string) error {
	query := `DELETE FROM note_shares WHERE note_id = $1 AND id = $2`
	res, err := r.DB.ExecContext(ctx, query, noteID, shareID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrItemNotFound
	}
	return nil
}
