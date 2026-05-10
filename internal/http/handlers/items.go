package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"github.com/hwdavr/notes-app-backend/internal/pkg/userctx"
	"go.uber.org/zap"
)

type ItemsHandler struct {
	Svc *domain.Service
	Log *zap.Logger
}

func (h *ItemsHandler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseListFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	items, err := h.Svc.ListItems(r.Context(), userIDFromContext(r), userEmailFromContext(r), filter)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (h *ItemsHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.Svc.GetItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

func (h *ItemsHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string  `json:"id"`
		ParentID *string `json:"parentId"`
		Name     string  `json:"name"`
		SortKey  string  `json:"sortKey"`
		DeviceID string  `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	item, err := h.Svc.CreateFolder(r.Context(), userIDFromContext(r), domain.CreateItemInput{
		ID:       req.ID,
		ParentID: req.ParentID,
		Name:     req.Name,
		SortKey:  req.SortKey,
		DeviceID: req.DeviceID,
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *ItemsHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string  `json:"id"`
		ParentID *string `json:"parentId"`
		Name     string  `json:"name"`
		Content  string  `json:"content"`
		SortKey  string  `json:"sortKey"`
		DeviceID string  `json:"deviceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	item, err := h.Svc.CreateNote(r.Context(), userIDFromContext(r), domain.CreateItemInput{
		ID:       req.ID,
		ParentID: req.ParentID,
		Name:     req.Name,
		Content:  req.Content,
		SortKey:  req.SortKey,
		DeviceID: req.DeviceID,
	})
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(item)
}

func (h *ItemsHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string `json:"name"`
		DeviceID          string `json:"deviceId"`
		LastSyncedVersion int64  `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.RenameItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.Name, req.DeviceID, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func (h *ItemsHandler) Move(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentID          *string `json:"parentId"`
		DeviceID          string  `json:"deviceId"`
		LastSyncedVersion int64   `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.MoveItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.DeviceID, req.ParentID, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func (h *ItemsHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SortKey           string `json:"sortKey"`
		DeviceID          string `json:"deviceId"`
		LastSyncedVersion int64  `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.ReorderItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.SortKey, req.DeviceID, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func (h *ItemsHandler) UpdateNoteContent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content           string `json:"content"`
		DeviceID          string `json:"deviceId"`
		LastSyncedVersion int64  `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.UpdateNoteContent(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.Content, req.DeviceID, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func (h *ItemsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID          string `json:"deviceId"`
		LastSyncedVersion int64  `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.DeleteItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.DeviceID, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func (h *ItemsHandler) Favorite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IsFavorite        bool   `json:"isFavorite"`
		DeviceID          string `json:"deviceId"`
		LastSyncedVersion int64  `json:"lastSyncedVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	result, err := h.Svc.FavoriteItem(r.Context(), userIDFromContext(r), userEmailFromContext(r), chi.URLParam(r, "itemID"), req.DeviceID, req.IsFavorite, req.LastSyncedVersion)
	h.writeMutationResult(w, result, err)
}

func parseListFilter(r *http.Request) (domain.ListItemsFilter, error) {
	filter := domain.ListItemsFilter{
		Type:  r.URL.Query().Get("type"),
		Query: r.URL.Query().Get("q"),
	}

	if raw := firstNonEmpty(r.URL.Query().Get("parentId"), r.URL.Query().Get("parent_id")); raw != "" {
		filter.ParentID = &raw
	}
	if raw := firstNonEmpty(r.URL.Query().Get("rootOnly"), r.URL.Query().Get("root_only")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return domain.ListItemsFilter{}, errors.New("invalid rootOnly filter")
		}
		filter.RootOnly = value
	}
	if raw := firstNonEmpty(r.URL.Query().Get("sinceVersion"), r.URL.Query().Get("since_version")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return domain.ListItemsFilter{}, errors.New("invalid sinceVersion filter")
		}
		filter.SinceVersion = &value
	}
	if raw := firstNonEmpty(r.URL.Query().Get("includeDeleted"), r.URL.Query().Get("include_deleted")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return domain.ListItemsFilter{}, errors.New("invalid includeDeleted filter")
		}
		filter.IncludeDeleted = value
	}

	return filter, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func userIDFromContext(r *http.Request) string {
	val := r.Context().Value(userctx.UserIDKey)
	userID, _ := val.(string)
	return userID
}
func userEmailFromContext(r *http.Request) string {
	val := r.Context().Value(userctx.UserEmailKey)
	email, _ := val.(string)
	return email
}

func (h *ItemsHandler) writeMutationResult(w http.ResponseWriter, result domain.MutationResult, err error) {
	if err != nil && !errors.Is(err, domain.ErrSyncConflict) {
		h.writeDomainError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if errors.Is(err, domain.ErrSyncConflict) {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(result)
}

func (h *ItemsHandler) writeDomainError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrInvalidItem):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrItemNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrSyncConflict):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidMove):
		status = http.StatusConflict
	}

	if status == http.StatusInternalServerError {
		h.Log.Error("domain error", zap.Error(err))
	}

	http.Error(w, err.Error(), status)
}
