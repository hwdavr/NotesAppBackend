package domain

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/hwdavr/notes-app-backend/internal/db/models"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
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

func (r *Repository) ListItems(ctx context.Context, userID string, filter ListItemsFilter) ([]Item, error) {
	var mods []bob.Mod[*dialect.SelectQuery]
	mods = append(mods, sm.Where(models.ItemColumns.UserID.EQ(psql.Arg(userID))))

	if filter.Type != "" {
		mods = append(mods, sm.Where(models.ItemColumns.Type.EQ(psql.Arg(filter.Type))))
	}

	if filter.ParentID != nil {
		mods = append(mods, sm.Where(models.ItemColumns.ParentID.EQ(psql.Arg(*filter.ParentID))))
	} else if filter.RootOnly {
		mods = append(mods, sm.Where(models.ItemColumns.ParentID.IsNull()))
	}

	if !filter.IncludeDeleted {
		mods = append(mods, sm.Where(models.ItemColumns.DeletedAt.IsNull()))
	}

	if filter.SinceVersion != nil {
		mods = append(mods, sm.Where(models.ItemColumns.Version.GT(psql.Arg(*filter.SinceVersion))))
	}

	if q := strings.TrimSpace(filter.Query); q != "" {
		pattern := "%" + q + "%"
		mods = append(mods, sm.Where(psql.Or(
			models.ItemColumns.Name.ILike(psql.Arg(pattern)),
			models.ItemColumns.Content.ILike(psql.Arg(pattern)),
		)))
	}

	mods = append(mods,
		sm.OrderBy(models.ItemColumns.ParentID).Asc().NullsFirst(),
		sm.OrderBy(models.ItemColumns.SortKey).Asc(),
		sm.OrderBy(models.ItemColumns.UpdatedAt).Desc(),
	)

	items, err := models.Items.Query(ctx, r.DB, mods...).All()
	if err != nil {
		return nil, err
	}

	result := make([]Item, len(items))
	for i, it := range items {
		result[i] = mapModelToItem(it)
	}
	return result, nil
}

func (r *Repository) GetItem(ctx context.Context, userID, itemID string) (Item, error) {
	item, err := models.Items.Query(ctx, r.DB,
		models.SelectWhere.Items.ID.EQ(itemID),
		models.SelectWhere.Items.UserID.EQ(userID),
	).One()

	if err != nil {
		if err == sql.ErrNoRows {
			return Item{}, ErrItemNotFound
		}
		return Item{}, err
	}

	return mapModelToItem(item), nil
}

func (r *Repository) UpdateItem(ctx context.Context, userID, itemID string, input UpdateItemInput) (Item, error) {
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

	_, err := psql.Update(
		um.Table(models.TableNames.Items),
		setter,
		um.Where(models.ItemColumns.ID.EQ(psql.Arg(itemID))),
		um.Where(models.ItemColumns.UserID.EQ(psql.Arg(userID))),
		um.Set(psql.Raw("version = version + 1")),
	).Exec(ctx, r.DB)

	if err != nil {
		return Item{}, err
	}

	return r.GetItem(ctx, userID, itemID)
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
