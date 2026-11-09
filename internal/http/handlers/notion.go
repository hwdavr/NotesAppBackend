package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/hwdavr/notes-app-backend/internal/domain"
)

type NotionHandler struct{ Svc *domain.NotionService }

func notionUnavailable(w http.ResponseWriter) {
	http.Error(w, "notion service unavailable", http.StatusServiceUnavailable)
}
func (h *NotionHandler) Start(w http.ResponseWriter, r *http.Request) {
	u, err := h.Svc.Start(r.Context(), userIDFromContext(r))
	if err != nil {
		notionUnavailable(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		AuthorizeURL string `json:"authorizeUrl"`
	}{u})
}
func (h *NotionHandler) Status(w http.ResponseWriter, r *http.Request) {
	c, err := h.Svc.Status(r.Context(), userIDFromContext(r))
	if err != nil {
		notionUnavailable(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}
func (h *NotionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if h.Svc.Disconnect(r.Context(), userIDFromContext(r)) != nil {
		notionUnavailable(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *NotionHandler) Callback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := h.Svc.Callback(r.Context(), q.Get("state"), q.Get("code"), q.Get("error"))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, "notesapp://notion/linked?status="+status, http.StatusFound)
}
