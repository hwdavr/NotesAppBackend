package main

import (
	stdhttp "net/http"

	"go.uber.org/zap"

	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/db"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	apihttp "github.com/hwdavr/notes-app-backend/internal/http"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"github.com/hwdavr/notes-app-backend/internal/pkg/email"
)

func main() {
	cfg := config.FromEnv()
	log, _ := zap.NewProduction()
	defer log.Sync()

	pg, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db connect", zap.Error(err))
	}

	repo := domain.NewRepository(pg)
	emailSvc := email.NewMockService()
	svc := domain.NewService(repo, emailSvc)
	ih := &handlers.ItemsHandler{Svc: svc, Log: log}
	sh := &handlers.SharesHandler{Svc: svc, Log: log}

	router := apihttp.NewRouter(ih, sh, cfg, log)

	log.Info("server starting", zap.String("addr", cfg.Addr))
	if err := stdhttp.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal("server error", zap.Error(err))
	}
}
