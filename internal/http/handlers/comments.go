package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"go.uber.org/zap"
)

// CommentsHandler handles HTTP requests for note block comments.
type CommentsHandler struct {
	Svc *domain.Service
	Log *zap.Logger
}

// List handles GET /v1/notes/{itemID}/blocks/{blockID}/comments
func (h *CommentsHandler) List(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	blockID := chi.URLParam(r, "blockID")

	comments, err := h.Svc.ListNoteBlockComments(r.Context(), userIDFromContext(r), userEmailFromContext(r), noteID, blockID)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(comments)
}

// Create handles POST /v1/notes/{itemID}/blocks/{blockID}/comments
func (h *CommentsHandler) Create(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	blockID := chi.URLParam(r, "blockID")

	var req domain.CreateNoteBlockCommentRequest
	if err := decodeCommentJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	comment, err := h.Svc.CreateNoteBlockComment(r.Context(), userIDFromContext(r), userEmailFromContext(r), noteID, blockID, req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(comment)
}

// Update handles PATCH /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}
func (h *CommentsHandler) Update(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	blockID := chi.URLParam(r, "blockID")
	commentID := chi.URLParam(r, "commentID")

	var req domain.UpdateNoteBlockCommentRequest
	if err := decodeCommentJSON(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	comment, err := h.Svc.UpdateNoteBlockComment(r.Context(), userIDFromContext(r), noteID, blockID, commentID, req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(comment)
}

// Delete handles DELETE /v1/notes/{itemID}/blocks/{blockID}/comments/{commentID}
func (h *CommentsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	blockID := chi.URLParam(r, "blockID")
	commentID := chi.URLParam(r, "commentID")

	if err := h.Svc.DeleteNoteBlockComment(r.Context(), userIDFromContext(r), noteID, blockID, commentID); err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CommentsHandler) writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidItem):
		status = http.StatusBadRequest
		message = "bad request"
	case errors.Is(err, domain.ErrItemNotFound):
		status = http.StatusNotFound
		message = "not found"
	case errors.Is(err, domain.ErrUnauthorized):
		status = http.StatusForbidden
		message = "forbidden"
	}

	if status == http.StatusInternalServerError && h.Log != nil {
		h.Log.Error("comments handler error", zap.Error(err))
	}

	http.Error(w, message, status)
}

func decodeCommentJSON(r *http.Request, destination any) error {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain one JSON value")
		}
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return err
	}
	if mentions, ok := fields["mentions"]; ok && bytes.Equal(bytes.TrimSpace(mentions), []byte("null")) {
		return errors.New("mentions must be an array")
	}
	return nil
}
