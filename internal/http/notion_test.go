package http_test

import (
	"net/http/httptest"
	"testing"

	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	apihttp "github.com/hwdavr/notes-app-backend/internal/http"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"go.uber.org/zap"
)

func TestNotionRouterAuthenticationAndCallback(t *testing.T) {
	r := apihttp.NewRouter(nil, nil, nil, config.Config{}, zap.NewNop(), &handlers.NotionHandler{Svc: &domain.NotionService{}})
	for _, tc := range []struct {
		method, path string
		want         int
	}{{"POST", "/v1/notion/link", 401}, {"GET", "/v1/notion/connection", 401}, {"DELETE", "/v1/notion/connection", 401}, {"GET", "/v1/notion/callback", 302}} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.want {
				t.Fatalf("status = %d, want %d", w.Code, tc.want)
			}
			if tc.want == 302 && w.Header().Get("Location") != "notesapp://notion/linked?status=failed" {
				t.Fatalf("unexpected redirect %q", w.Header().Get("Location"))
			}
		})
	}
}
