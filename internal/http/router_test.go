package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/config"
	apihttp "github.com/hwdavr/notes-app-backend/internal/http"
	"go.uber.org/zap"
)

func TestRouterHealthz(t *testing.T) {
	router := apihttp.NewRouter(nil, nil, nil, config.Config{}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", res.Code, http.StatusOK)
	}
	if body := res.Body.String(); body != "ok" {
		t.Fatalf("health body = %q, want %q", body, "ok")
	}
}

func TestRouterRejectsUnauthenticatedAPIRequest(t *testing.T) {
	router := apihttp.NewRouter(nil, nil, nil, config.Config{}, zap.NewNop())
	req := httptest.NewRequest(http.MethodGet, "/v1/items", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
