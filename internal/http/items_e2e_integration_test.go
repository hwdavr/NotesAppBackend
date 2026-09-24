//go:build integration

package http

import (
	stdhttp "net/http"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/domain"
)

func TestAPIItemEndpoints(t *testing.T) {
	const (
		userID      = "api-e2e-items-user"
		folderOneID = "api-e2e-items-folder-one"
		folderTwoID = "api-e2e-items-folder-two"
		noteID      = "api-e2e-items-note"
	)
	env := newAPIE2EEnv(t, userID)

	folderOne := createAPIFolder(t, env, folderOneID, "First Folder")
	folderTwo := createAPIFolder(t, env, folderTwoID, "Second Folder")
	note := createAPINote(t, env, noteID, folderOne.ID, "Endpoint Note")

	status, header, body := env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/items?rootOnly=true", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var rootItems []domain.Item
	decodeAPIJSON(t, body, &rootItems)
	if len(rootItems) != 2 {
		t.Fatal("root item list did not contain both created folders")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/items?parentId="+folderOne.ID, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var children []domain.Item
	decodeAPIJSON(t, body, &children)
	if len(children) != 1 || children[0].ID != note.ID {
		t.Fatal("parent item list did not contain the created note")
	}

	fetched := getAPIItem(t, env, note.ID)
	if fetched.ParentID == nil || *fetched.ParentID != folderOne.ID || fetched.UserID != userID {
		t.Fatal("get item response did not contain the created note ownership and parent")
	}

	rename := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/items/"+note.ID+"/rename", map[string]any{
		"name":              "Renamed Endpoint Note",
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": note.Version,
	})
	if rename.Item.Name != "Renamed Endpoint Note" || rename.Item.Version != 2 {
		t.Fatal("rename response did not persist the new name at version 2")
	}

	reorder := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/items/"+note.ID+"/reorder", map[string]any{
		"sortKey":           "z9",
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": rename.Item.Version,
	})
	if reorder.Item.SortKey != "z9" || reorder.Item.Version != 3 {
		t.Fatal("reorder response did not persist the new sort key at version 3")
	}

	favorite := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/items/"+note.ID+"/favorite", map[string]any{
		"isFavorite":        true,
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": reorder.Item.Version,
	})
	if !favorite.Item.IsFavorite || favorite.Item.Version != 4 {
		t.Fatal("favorite response did not persist the favorite flag at version 4")
	}

	folderContent := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/items/"+folderTwo.ID+"/content", map[string]any{
		"content":           "Folder description",
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": folderTwo.Version,
	})
	if folderContent.Item.Content != "Folder description" || folderContent.Item.Version != 2 {
		t.Fatal("item content response did not persist the folder content at version 2")
	}

	noteContent := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/notes/"+note.ID+"/content", map[string]any{
		"content":           "Updated endpoint note content",
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": favorite.Item.Version,
	})
	if noteContent.Item.Content != "Updated endpoint note content" || noteContent.Item.Version != 5 {
		t.Fatal("note content response did not persist the note content at version 5")
	}

	moved := mutateAPIItem(t, env, stdhttp.MethodPatch, "/v1/items/"+note.ID+"/move", map[string]any{
		"parentId":          folderTwo.ID,
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": noteContent.Item.Version,
	})
	if moved.Item.ParentID == nil || *moved.Item.ParentID != folderTwo.ID || moved.Item.Version != 6 {
		t.Fatal("move response did not place the note under the second folder at version 6")
	}

	fetched = getAPIItem(t, env, note.ID)
	if fetched.Name != "Renamed Endpoint Note" || fetched.SortKey != "z9" || !fetched.IsFavorite || fetched.Content != "Updated endpoint note content" || fetched.Version != 6 {
		t.Fatal("get item response did not persist the item mutations")
	}

	deleted := mutateAPIItem(t, env, stdhttp.MethodDelete, "/v1/items/"+folderTwo.ID, map[string]any{
		"deviceId":          apiE2EDeviceID,
		"lastSyncedVersion": folderContent.Item.Version,
	})
	if deleted.Item.DeletedAt == nil || deleted.Item.ID != folderTwo.ID {
		t.Fatal("delete response did not return the folder tombstone")
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/items?rootOnly=true", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	decodeAPIJSON(t, body, &rootItems)
	if len(rootItems) != 1 || rootItems[0].ID != folderOne.ID {
		t.Fatal("post-delete root list did not retain only the first folder")
	}
}
