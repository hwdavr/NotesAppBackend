//go:build integration

package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"go.uber.org/zap"
)

func TestNotionPostgresAuthenticatedAPI(t *testing.T) {
	db := openAPIIntegrationDatabase(t)
	user := "notion-api-test-user"
	other := "notion-api-test-other"
	cleanup := func() {
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		if _, err := db.ExecContext(c, `DELETE FROM notion_link_users WHERE user_id IN ($1,$2)`, user, other); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	}
	cleanup()
	t.Cleanup(cleanup)
	auth0JWKSCache = jwksCache{}
	t.Cleanup(func() { auth0JWKSCache = jwksCache{} })
	privateKey, jwks := newAPIIntegrationJWKS(t)
	t.Cleanup(jwks.Close)
	client := &http.Client{Transport: notionTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"synthetic-token","workspace_name":"Synthetic workspace"}`))}, nil
	})}
	svc := &domain.NotionService{Store: &domain.NotionRepository{DB: db}, ClientID: "client", ClientSecret: "secret", RedirectURL: "https://example.test/v1/notion/callback", EncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), TokenURL: "https://notion.test/token", Client: client}
	router := NewRouter(nil, nil, nil, config.Config{Auth0Issuer: apiE2EIssuer, Auth0Audience: apiE2EAudience, Auth0JWKSURL: jwks.URL}, zap.NewNop(), &handlers.NotionHandler{Svc: svc})
	call := func(method, path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	token := signAPIIntegrationToken(t, privateKey, user, apiE2EAudience)
	otherToken := signAPIIntegrationToken(t, privateKey, other, apiE2EAudience)
	res := call("POST", "/v1/notion/link", token)
	if res.Code != 200 {
		t.Fatalf("start: %d", res.Code)
	}
	var link struct {
		URL string `json:"authorizeUrl"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &link); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(link.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("state missing")
	}
	res = call("GET", "/v1/notion/callback?state="+url.QueryEscape(state)+"&code=test", "")
	if res.Code != 302 || res.Header().Get("Location") != "notesapp://notion/linked?status=linked" {
		t.Fatalf("callback: %d %s", res.Code, res.Header().Get("Location"))
	}
	res = call("GET", "/v1/notion/connection", token)
	if res.Code != 200 || res.Body.String() != `{"linked":true,"accountName":"Synthetic workspace"}`+"\n" {
		t.Fatalf("status: %d %s", res.Code, res.Body.String())
	}
	res = call("GET", "/v1/notion/connection", otherToken)
	if res.Code != 200 || res.Body.String() != `{"linked":false}`+"\n" {
		t.Fatal("cross-user visibility")
	}
	res = call("DELETE", "/v1/notion/connection", token)
	if res.Code != 204 {
		t.Fatalf("disconnect: %d", res.Code)
	}
	res = call("GET", "/v1/notion/connection", token)
	if res.Code != 200 || res.Body.String() != `{"linked":false}`+"\n" {
		t.Fatal("connection survived disconnect")
	}
}
