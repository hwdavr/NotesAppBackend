package domain

import (
	"context"
	"strings"
)

// ListNoteBlockComments returns all comments on a specific block of a note.
// The caller must have at least read access (the note must be accessible to userID).
func (s *Service) ListNoteBlockComments(ctx context.Context, userID, userEmail, noteID, blockID string) ([]NoteBlockComment, error) {
	if noteID == "" || blockID == "" {
		return nil, ErrInvalidItem
	}

	// Verify the requesting user has access to the note (any role suffices for reads).
	if _, err := s.Repo.GetItem(ctx, userID, userEmail, noteID); err != nil {
		return nil, err
	}

	comments, err := s.Repo.ListNoteBlockComments(ctx, noteID, blockID)
	if err != nil {
		return nil, err
	}
	// Never return nil slice — callers expect an empty array in JSON.
	if comments == nil {
		comments = []NoteBlockComment{}
	}
	return comments, nil
}

// CreateNoteBlockComment creates a new comment on a note block.
// The author is derived from the authenticated userID/userEmail; only users with
// at least read access to the note may comment.
func (s *Service) CreateNoteBlockComment(
	ctx context.Context,
	userID, userEmail, noteID, blockID string,
	input CreateNoteBlockCommentRequest,
) (NoteBlockComment, error) {
	input.Body = strings.TrimSpace(input.Body)
	if noteID == "" || blockID == "" || userID == "" || input.Body == "" {
		return NoteBlockComment{}, ErrInvalidItem
	}

	// Verify the requesting user has access to the note (any role suffices for commenting).
	if _, err := s.Repo.GetItem(ctx, userID, userEmail, noteID); err != nil {
		return NoteBlockComment{}, err
	}

	// Populate author metadata from the context email if available.
	var authorEmail *string
	if userEmail != "" {
		e := userEmail
		authorEmail = &e
	}

	return s.Repo.CreateNoteBlockComment(ctx, noteID, blockID, userID, nil, authorEmail, input.Body)
}
