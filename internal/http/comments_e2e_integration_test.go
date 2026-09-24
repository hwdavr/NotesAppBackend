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
