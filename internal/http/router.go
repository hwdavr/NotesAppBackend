package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"go.uber.org/zap"
)

func NewRouter(ih *handlers.ItemsHandler, cfg config.Config, log *zap.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(Logger(log))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	}))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Group(func(pr chi.Router) {
		pr.Use(AuthMiddleware(AuthConfig{
			Issuer:   cfg.Auth0Issuer,
			Audience: cfg.Auth0Audience,
			JWKSURL:  cfg.Auth0JWKSURL,
		}))
		pr.Route("/v1", func(api chi.Router) {
			api.Get("/items", ih.List)
			api.Get("/items/{itemID}", ih.Get)
			api.Post("/folders", ih.CreateFolder)
			api.Post("/notes", ih.CreateNote)
			api.Patch("/items/{itemID}/rename", ih.Rename)
			api.Patch("/items/{itemID}/move", ih.Move)
			api.Patch("/items/{itemID}/reorder", ih.Reorder)
			api.Patch("/notes/{itemID}/content", ih.UpdateNoteContent)
			api.Delete("/items/{itemID}", ih.Delete)
		})
	})

	return r
}
