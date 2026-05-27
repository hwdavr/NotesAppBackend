package domain

import (
	"context"
	"database/sql"
)

// ListNoteBlockComments returns all comments for a given note block, ordered by creation time.
func (r *Repository) ListNoteBlockComments(ctx context.Context, noteID, blockID string) ([]NoteBlockComment, error) {
	query := `
		SELECT id, note_id, block_id, author_user_id, author_display_name, author_email, body, created_at, updated_at
		FROM note_block_comments
		WHERE note_id = $1 AND block_id = $2
		ORDER BY created_at ASC
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, blockID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []NoteBlockComment
	for rows.Next() {
		var c NoteBlockComment
		if err := rows.Scan(
			&c.ID, &c.NoteID, &c.BlockID, &c.AuthorUserID,
			&c.AuthorDisplayName, &c.AuthorEmail, &c.Body,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

// CreateNoteBlockComment inserts a new comment on a note block, returning the persisted record.
// authorDisplayName and authorEmail may be nil if not available from the identity provider.
func (r *Repository) CreateNoteBlockComment(
	ctx context.Context,
	noteID, blockID, authorUserID string,
	authorDisplayName, authorEmail *string,
	body string,
) (NoteBlockComment, error) {
	query := `
		INSERT INTO note_block_comments (note_id, block_id, author_user_id, author_display_name, author_email, body)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, note_id, block_id, author_user_id, author_display_name, author_email, body, created_at, updated_at
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, blockID, authorUserID, authorDisplayName, authorEmail, body)
	if err != nil {
		return NoteBlockComment{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteBlockComment{}, sql.ErrNoRows
	}

	var c NoteBlockComment
	if err := rows.Scan(
		&c.ID, &c.NoteID, &c.BlockID, &c.AuthorUserID,
		&c.AuthorDisplayName, &c.AuthorEmail, &c.Body,
		&c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return NoteBlockComment{}, err
	}
	return c, nil
}
