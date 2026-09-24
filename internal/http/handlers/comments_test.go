package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentsHandlerUpdateRejectsMalformedJSON(t *testing.T) {
	handler := &CommentsHandler{}
	request := httptest.NewRequest(http.MethodPatch, "/v1/notes/note/blocks/block/comments/comment", strings.NewReader("{"))
	response := httptest.NewRecorder()

	handler.Update(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
