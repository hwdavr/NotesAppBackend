package domain

import (
	"context"
	"database/sql"
	"encoding/json"
)

type commentRowScanner interface {
	Scan(dest ...any) error
}

func scanNoteBlockComment(scanner commentRowScanner) (NoteBlockComment, error) {
	var comment NoteBlockComment
	var mentionsJSON []byte
	if err := scanner.Scan(
		&comment.ID, &comment.NoteID, &comment.BlockID, &comment.ParentCommentID,
		&comment.AuthorUserID, &comment.AuthorDisplayName, &comment.AuthorEmail,
		&comment.Body, &mentionsJSON, &comment.CreatedAt, &comment.UpdatedAt,
	); err != nil {
		return NoteBlockComment{}, err
	}

	if len(mentionsJSON) == 0 {
		comment.Mentions = []MentionReference{}
	} else if err := json.Unmarshal(mentionsJSON, &comment.Mentions); err != nil {
		return NoteBlockComment{}, err
	} else if comment.Mentions == nil {
		comment.Mentions = []MentionReference{}
	}

	return comment, nil
}

func marshalMentions(mentions []MentionReference) ([]byte, error) {
	if mentions == nil {
		mentions = []MentionReference{}
	}
	return json.Marshal(mentions)
}

// ListNoteBlockComments returns all comments for a given note block, ordered by creation time.
func (r *Repository) ListNoteBlockComments(ctx context.Context, noteID, blockID string) ([]NoteBlockComment, error) {
	query := `
		SELECT id, note_id, block_id, parent_comment_id, author_user_id, author_display_name, author_email, body, mentions, created_at, updated_at
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
		comment, err := scanNoteBlockComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
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
	parentCommentID *string,
	mentions []MentionReference,
) (NoteBlockComment, error) {
	mentionsJSON, err := marshalMentions(mentions)
	if err != nil {
		return NoteBlockComment{}, err
	}

	query := `
		INSERT INTO note_block_comments (note_id, block_id, parent_comment_id, author_user_id, author_display_name, author_email, body, mentions)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, note_id, block_id, parent_comment_id, author_user_id, author_display_name, author_email, body, mentions, created_at, updated_at
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, blockID, parentCommentID, authorUserID, authorDisplayName, authorEmail, body, mentionsJSON)
	if err != nil {
		return NoteBlockComment{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteBlockComment{}, sql.ErrNoRows
	}

	comment, err := scanNoteBlockComment(rows)
	if err != nil {
		return NoteBlockComment{}, err
	}
	return comment, nil
}

// GetNoteBlockComment returns one comment scoped to its note and block.
func (r *Repository) GetNoteBlockComment(ctx context.Context, noteID, blockID, commentID string) (NoteBlockComment, error) {
	query := `
		SELECT id, note_id, block_id, parent_comment_id, author_user_id, author_display_name, author_email, body, mentions, created_at, updated_at
		FROM note_block_comments
		WHERE note_id = $1 AND block_id = $2 AND id = $3
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, blockID, commentID)
	if err != nil {
		return NoteBlockComment{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteBlockComment{}, ErrItemNotFound
	}
	return scanNoteBlockComment(rows)
}

// UpdateNoteBlockComment updates mutable comment fields and returns the persisted record.
func (r *Repository) UpdateNoteBlockComment(
	ctx context.Context,
	noteID, blockID, commentID, body string,
	mentions []MentionReference,
) (NoteBlockComment, error) {
	mentionsJSON, err := marshalMentions(mentions)
	if err != nil {
		return NoteBlockComment{}, err
	}

	query := `
		UPDATE note_block_comments
		SET body = $4, mentions = $5, updated_at = NOW()
		WHERE note_id = $1 AND block_id = $2 AND id = $3
		RETURNING id, note_id, block_id, parent_comment_id, author_user_id, author_display_name, author_email, body, mentions, created_at, updated_at
	`
	rows, err := r.DB.QueryContext(ctx, query, noteID, blockID, commentID, body, mentionsJSON)
	if err != nil {
		return NoteBlockComment{}, err
	}
	defer rows.Close()

	if !rows.Next() {
		return NoteBlockComment{}, ErrItemNotFound
	}
	return scanNoteBlockComment(rows)
}

// DeleteNoteBlockComment deletes one comment scoped to its note and block.
func (r *Repository) DeleteNoteBlockComment(ctx context.Context, noteID, blockID, commentID string) error {
	result, err := r.DB.ExecContext(ctx, `
		DELETE FROM note_block_comments
		WHERE note_id = $1 AND block_id = $2 AND id = $3
	`, noteID, blockID, commentID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrItemNotFound
	}
	return nil
}
