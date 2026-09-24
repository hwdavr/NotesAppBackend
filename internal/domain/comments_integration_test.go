//go:build integration

package domain

import (
	"context"
	"testing"
	"time"

	"github.com/hwdavr/notes-app-backend/internal/pkg/email"
)

func TestPostgresIntegrationNoteBlockCommentMutations(t *testing.T) {
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
	note, err := service.CreateNote(ctx, integrationUserID, "", CreateItemInput{
		ID:       "integration-comment-note",
		Name:     "Comment Integration Note",
		Content:  "comment integration content",
		SortKey:  "a0",
		DeviceID: "integration-device",
	})
	if err != nil {
		t.Fatalf("create comment integration note: %v", err)
	}

	mention := MentionReference{
		Kind:        "person",
		TargetID:    "integration-mentioned-user",
		DisplayText: "Mentioned User",
		RangeStart:  0,
		RangeLength: 14,
	}
	parent, err := service.CreateNoteBlockComment(ctx, integrationUserID, "", note.ID, "integration-block", CreateNoteBlockCommentRequest{
		Body:     "parent comment",
		Mentions: []MentionReference{mention},
	})
	if err != nil {
		t.Fatalf("create parent comment: %v", err)
	}

	child, err := service.CreateNoteBlockComment(ctx, integrationUserID, "", note.ID, "integration-block", CreateNoteBlockCommentRequest{
		Body:            "child comment",
		ParentCommentID: &parent.ID,
	})
	if err != nil {
		t.Fatalf("create child comment: %v", err)
	}
	if child.ParentCommentID == nil || *child.ParentCommentID != parent.ID || child.Mentions == nil {
		t.Fatalf("child comment = %#v, want parent and non-nil mentions", child)
	}

	comments, err := service.ListNoteBlockComments(ctx, integrationUserID, "", note.ID, "integration-block")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 2 || len(comments[0].Mentions) != 1 || comments[1].ParentCommentID == nil || *comments[1].ParentCommentID != parent.ID {
		t.Fatalf("listed comments = %#v, want persisted metadata", comments)
	}

	updated, err := service.UpdateNoteBlockComment(ctx, integrationUserID, note.ID, "integration-block", child.ID, UpdateNoteBlockCommentRequest{
		Body: "updated child comment",
		Mentions: []MentionReference{{
			Kind:        "note",
			TargetID:    note.ID,
			DisplayText: "Comment Integration Note",
			RangeStart:  0,
			RangeLength: 24,
		}},
	})
	if err != nil {
		t.Fatalf("update child comment: %v", err)
	}
	if updated.Body != "updated child comment" || len(updated.Mentions) != 1 || updated.Mentions[0].Kind != "note" {
		t.Fatalf("updated comment = %#v, want updated body and mentions", updated)
	}

	if _, err := service.UpdateNoteBlockComment(ctx, "other-integration-user", note.ID, "integration-block", child.ID, UpdateNoteBlockCommentRequest{Body: "forbidden"}); err != ErrUnauthorized {
		t.Fatalf("non-author update error = %v, want %v", err, ErrUnauthorized)
	}

	if err := service.DeleteNoteBlockComment(ctx, integrationUserID, note.ID, "integration-block", child.ID); err != nil {
		t.Fatalf("delete child comment: %v", err)
	}
	comments, err = service.ListNoteBlockComments(ctx, integrationUserID, "", note.ID, "integration-block")
	if err != nil {
		t.Fatalf("list comments after delete: %v", err)
	}
	if len(comments) != 1 || comments[0].ID != parent.ID {
		t.Fatalf("comments after delete = %#v, want only parent comment", comments)
	}
}
