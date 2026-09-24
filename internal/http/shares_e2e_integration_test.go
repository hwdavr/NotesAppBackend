//go:build integration

package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/domain"
)

func TestAPINoteShareEndpoints(t *testing.T) {
	const (
		userID       = "api-e2e-shares-user"
		noteID       = "api-e2e-shares-note"
		collaborator = "collaborator@example.invalid"
	)
	env := newAPIE2EEnv(t, userID)
	note := createAPINote(t, env, noteID, "", "Shared Note")

	status, header, body := env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/notes/"+note.ID+"/shares", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var shares []domain.NoteShare
	decodeAPIJSON(t, body, &shares)
	if len(shares) != 0 {
		t.Fatal("initial share list was not empty")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodPost, "/v1/notes/"+note.ID+"/shares", map[string]any{
		"email":      collaborator,
		"accessRole": domain.AccessRoleReadOnly,
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)
	var share domain.NoteShare
	decodeAPIJSON(t, body, &share)
	if share.ID == "" || share.NoteID != note.ID || share.Email != collaborator || share.AccessRole != domain.AccessRoleReadOnly {
		t.Fatal("create share response did not contain the expected active share")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/notes/"+note.ID+"/shares", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &shares)
	if len(shares) != 1 || shares[0].ID != share.ID {
		t.Fatal("share list did not contain the created share")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodPatch, "/v1/notes/"+note.ID+"/shares/"+share.ID, map[string]any{
		"accessRole": domain.AccessRoleFullAccess,
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &share)
	if share.AccessRole != domain.AccessRoleFullAccess {
		t.Fatal("update share response did not contain the requested access role")
	}

	status, _, _ = env.authenticatedRequest(t, stdhttp.MethodDelete, "/v1/notes/"+note.ID+"/shares/"+share.ID, nil)
	if status != stdhttp.StatusNoContent {
		t.Fatalf("delete share status = %d, want %d", status, stdhttp.StatusNoContent)
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/notes/"+note.ID+"/shares", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &shares)
	if len(shares) != 0 {
		t.Fatal("share list was not empty after deletion")
	}
}
