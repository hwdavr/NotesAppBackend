package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/domain"
	"go.uber.org/zap"
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

func TestCommentsHandlerDoesNotExposeInternalErrors(t *testing.T) {
	handler := &CommentsHandler{Log: zap.NewNop()}
	response := httptest.NewRecorder()

	handler.writeDomainError(response, errors.New("database password=secret"))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if response.Body.String() != "internal server error\n" {
		t.Fatalf("body = %q, want sanitized internal error", response.Body.String())
	}
}

func TestCommentsHandlerMapsDomainErrorsToStableMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code int
		body string
	}{
		{name: "bad request", err: domain.ErrInvalidItem, code: http.StatusBadRequest, body: "bad request\n"},
		{name: "not found", err: domain.ErrItemNotFound, code: http.StatusNotFound, body: "not found\n"},
		{name: "forbidden", err: domain.ErrUnauthorized, code: http.StatusForbidden, body: "forbidden\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &CommentsHandler{Log: zap.NewNop()}
			response := httptest.NewRecorder()

			handler.writeDomainError(response, tt.err)

			if response.Code != tt.code || response.Body.String() != tt.body {
				t.Fatalf("response = (%d, %q), want (%d, %q)", response.Code, response.Body.String(), tt.code, tt.body)
			}
		})
	}
}

func TestCommentsHandlerUpdateRejectsNonStrictJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"body":"comment","unexpected":true}`},
		{name: "trailing value", body: `{"body":"comment"}{}`},
		{name: "null mentions", body: `{"body":"comment","mentions":null}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &CommentsHandler{}
			request := httptest.NewRequest(http.MethodPatch, "/v1/notes/note/blocks/block/comments/comment", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			handler.Update(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
		})
	}
}
