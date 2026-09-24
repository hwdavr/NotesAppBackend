package domain

import (
	"context"
	"strings"
)

var validMentionKinds = map[string]struct{}{
	"person": {},
	"date":   {},
	"note":   {},
	"folder": {},
}

func normalizeNoteBlockCommentInput(body string, mentions []MentionReference) (string, []MentionReference, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", nil, ErrInvalidItem
	}

	normalizedMentions := make([]MentionReference, len(mentions))
	for i, mention := range mentions {
		mention.Kind = strings.TrimSpace(mention.Kind)
		mention.TargetID = strings.TrimSpace(mention.TargetID)
		mention.DisplayText = strings.TrimSpace(mention.DisplayText)
		if _, ok := validMentionKinds[mention.Kind]; !ok || mention.TargetID == "" || mention.DisplayText == "" || mention.RangeStart < 0 || mention.RangeLength < 0 {
			return "", nil, ErrInvalidItem
		}
		normalizedMentions[i] = mention
	}

	return body, normalizedMentions, nil
}

func normalizeParentCommentID(parentCommentID *string) *string {
	if parentCommentID == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*parentCommentID)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

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
	body, mentions, err := normalizeNoteBlockCommentInput(input.Body, input.Mentions)
	parentCommentID := normalizeParentCommentID(input.ParentCommentID)
	if noteID == "" || blockID == "" || userID == "" || body == "" || err != nil {
		return NoteBlockComment{}, ErrInvalidItem
	}

	// Verify the requesting user has access to the note (any role suffices for commenting).
	if _, err := s.Repo.GetItem(ctx, userID, userEmail, noteID); err != nil {
		return NoteBlockComment{}, err
	}
	if parentCommentID != nil {
		if _, err := s.Repo.GetNoteBlockComment(ctx, noteID, blockID, *parentCommentID); err != nil {
			return NoteBlockComment{}, err
		}
	}

	// Populate author metadata from the context email if available.
	var authorEmail *string
	if userEmail != "" {
		e := userEmail
		authorEmail = &e
	}

	return s.Repo.CreateNoteBlockComment(ctx, noteID, blockID, userID, nil, authorEmail, body, parentCommentID, mentions)
}

// UpdateNoteBlockComment updates a comment when the authenticated user is its author.
func (s *Service) UpdateNoteBlockComment(
	ctx context.Context,
	userID, noteID, blockID, commentID string,
	input UpdateNoteBlockCommentRequest,
) (NoteBlockComment, error) {
	body, mentions, err := normalizeNoteBlockCommentInput(input.Body, input.Mentions)
	if userID == "" || noteID == "" || blockID == "" || commentID == "" || err != nil {
		return NoteBlockComment{}, ErrInvalidItem
	}

	comment, err := s.Repo.GetNoteBlockComment(ctx, noteID, blockID, commentID)
	if err != nil {
		return NoteBlockComment{}, err
	}
	if comment.AuthorUserID != userID {
		return NoteBlockComment{}, ErrUnauthorized
	}

	return s.Repo.UpdateNoteBlockComment(ctx, noteID, blockID, commentID, body, mentions)
}

// DeleteNoteBlockComment deletes a comment when the authenticated user is its author.
func (s *Service) DeleteNoteBlockComment(ctx context.Context, userID, noteID, blockID, commentID string) error {
	if userID == "" || noteID == "" || blockID == "" || commentID == "" {
		return ErrInvalidItem
	}

	comment, err := s.Repo.GetNoteBlockComment(ctx, noteID, blockID, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorUserID != userID {
		return ErrUnauthorized
	}

	return s.Repo.DeleteNoteBlockComment(ctx, noteID, blockID, commentID)
}
