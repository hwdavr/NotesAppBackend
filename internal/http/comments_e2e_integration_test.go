//go:build integration

package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/domain"
)

func TestAPINoteBlockCommentEndpoints(t *testing.T) {
	const (
		userID  = "api-e2e-comments-user"
		noteID  = "api-e2e-comments-note"
		blockID = "api-e2e-comments-block"
	)
	env := newAPIE2EEnv(t, userID)
	note := createAPINote(t, env, noteID, "", "Commented Note")
	path := "/v1/notes/" + note.ID + "/blocks/" + blockID + "/comments"

	status, header, body := env.authenticatedRequest(t, stdhttp.MethodGet, path, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var comments []domain.NoteBlockComment
	decodeAPIJSON(t, body, &comments)
	if len(comments) != 0 {
		t.Fatal("initial block comment list was not empty")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodPost, path, map[string]any{
		"body": "integration comment",
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)
	var comment domain.NoteBlockComment
	decodeAPIJSON(t, body, &comment)
	if comment.ID == "" || comment.NoteID != note.ID || comment.BlockID != blockID || comment.AuthorUserID != userID {
		t.Fatal("create comment response did not contain the authenticated author and block identity")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, path, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &comments)
	if len(comments) != 1 || comments[0].ID != comment.ID {
		t.Fatal("block comment list did not contain the created comment")
	}
}

func TestAPINoteBlockCommentMutationEndpoints(t *testing.T) {
	const (
		userID      = "api-e2e-comment-mutation-user"
		otherUserID = "api-e2e-comment-mutation-other-user"
		noteID      = "api-e2e-comment-mutation-note"
		blockID     = "api-e2e-comment-mutation-block"
	)
	env := newAPIE2EEnv(t, userID)
	note := createAPINote(t, env, noteID, "", "Comment Mutation Note")
	path := "/v1/notes/" + note.ID + "/blocks/" + blockID + "/comments"
	status, _, _ := env.request(t, stdhttp.MethodPatch, path+"/missing-comment", "", map[string]any{"body": "unauthorized"})
	if status != stdhttp.StatusUnauthorized {
		t.Fatalf("unauthenticated update status = %d, want %d", status, stdhttp.StatusUnauthorized)
	}

	parentMention := map[string]any{
		"kind":        "person",
		"targetId":    otherUserID,
		"displayText": "Other User",
		"rangeStart":  0,
		"rangeLength": 10,
	}
	status, header, body := env.authenticatedRequest(t, stdhttp.MethodPost, path, map[string]any{
		"body":     "parent comment",
		"mentions": []any{parentMention},
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)
	var parent domain.NoteBlockComment
	decodeAPIJSON(t, body, &parent)
	if len(parent.Mentions) != 1 || parent.Mentions[0].TargetID != otherUserID {
		t.Fatalf("parent mentions = %#v, want persisted mention", parent.Mentions)
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodPost, path, map[string]any{
		"body":            "reply comment",
		"parentCommentId": parent.ID,
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)
	var reply domain.NoteBlockComment
	decodeAPIJSON(t, body, &reply)
	if reply.ParentCommentID == nil || *reply.ParentCommentID != parent.ID || reply.Mentions == nil {
		t.Fatalf("reply = %#v, want parent and non-nil mentions", reply)
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, path, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var comments []domain.NoteBlockComment
	decodeAPIJSON(t, body, &comments)
	if len(comments) != 2 || comments[1].ParentCommentID == nil || *comments[1].ParentCommentID != parent.ID {
		t.Fatalf("listed comments = %#v, want persisted parent relationship", comments)
	}

	mutationPath := path + "/" + reply.ID
	updatedMention := map[string]any{
		"kind":        "note",
		"targetId":    note.ID,
		"displayText": "Comment Mutation Note",
		"rangeStart":  0,
		"rangeLength": 21,
	}
	status, header, body = env.authenticatedRequest(t, stdhttp.MethodPatch, mutationPath, map[string]any{
		"body":     "updated reply comment",
		"mentions": []any{updatedMention},
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var updated domain.NoteBlockComment
	decodeAPIJSON(t, body, &updated)
	if updated.ID != reply.ID || updated.Body != "updated reply comment" || len(updated.Mentions) != 1 || updated.Mentions[0].Kind != "note" {
		t.Fatalf("updated comment = %#v, want updated body and mention", updated)
	}

	otherToken := signAPIIntegrationToken(t, env.privateKey, otherUserID, apiE2EAudience)
	status, _, _ = env.request(t, stdhttp.MethodPatch, mutationPath, otherToken, map[string]any{"body": "forbidden"})
	if status != stdhttp.StatusForbidden {
		t.Fatalf("non-author update status = %d, want %d", status, stdhttp.StatusForbidden)
	}
	status, _, _ = env.request(t, stdhttp.MethodDelete, mutationPath, otherToken, nil)
	if status != stdhttp.StatusForbidden {
		t.Fatalf("non-author delete status = %d, want %d", status, stdhttp.StatusForbidden)
	}

	status, _, _ = env.authenticatedRequest(t, stdhttp.MethodPatch, path+"/missing-comment", map[string]any{"body": "missing"})
	if status != stdhttp.StatusNotFound {
		t.Fatalf("missing comment update status = %d, want %d", status, stdhttp.StatusNotFound)
	}

	status, _, body = env.authenticatedRequest(t, stdhttp.MethodDelete, mutationPath, nil)
	if status != stdhttp.StatusNoContent || len(body) != 0 {
		t.Fatalf("author delete = status %d body %q, want 204 and empty body", status, body)
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, path, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &comments)
	if len(comments) != 1 || comments[0].ID != parent.ID {
		t.Fatalf("comments after delete = %#v, want only parent comment", comments)
	}
}
