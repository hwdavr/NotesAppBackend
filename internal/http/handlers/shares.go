package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"go.uber.org/zap"
)

type SharesHandler struct {
	Svc *domain.Service
	Log *zap.Logger
}

func (h *SharesHandler) List(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	shares, err := h.Svc.ListNoteShares(r.Context(), userIDFromContext(r), noteID)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(shares)
}

func (h *SharesHandler) Create(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	var req domain.CreateNoteShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	share, err := h.Svc.CreateNoteShare(r.Context(), userIDFromContext(r), noteID, req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(share)
}

func (h *SharesHandler) Update(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	shareID := chi.URLParam(r, "shareID")
	var req domain.UpdateNoteShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	share, err := h.Svc.UpdateNoteShare(r.Context(), userIDFromContext(r), noteID, shareID, req)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(share)
}

func (h *SharesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	noteID := chi.URLParam(r, "itemID")
	shareID := chi.URLParam(r, "shareID")

	err := h.Svc.DeleteNoteShare(r.Context(), userIDFromContext(r), noteID, shareID)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SharesHandler) writeDomainError(w http.ResponseWriter, err error) {
	// Reusing the logic from ItemsHandler if possible, but for simplicity I'll duplicate or call a helper if I had one.
	// Since I cannot easily create a shared helper across packages without more restructuring, I'll just implement it here.
	status := http.StatusInternalServerError
	if err == domain.ErrInvalidItem {
		status = http.StatusBadRequest
	} else if err == domain.ErrItemNotFound {
		status = http.StatusNotFound
	} else if err == domain.ErrConflict {
		status = http.StatusConflict
	} else if err == domain.ErrUnauthorized {
		status = http.StatusForbidden
	}

	if status == http.StatusInternalServerError {
		h.Log.Error("domain error", zap.Error(err))
	}

	http.Error(w, err.Error(), status)
}
