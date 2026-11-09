package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/hwdavr/notes-app-backend/internal/domain"
	"github.com/hwdavr/notes-app-backend/internal/pkg/userctx"
)

type notionStoreFake struct {
	state     [32]byte
	owner     string
	connected bool
}

func (s *notionStoreFake) CreateState(_ context.Context, u string, h []byte, _ time.Time) error {
	copy(s.state[:], h)
	s.owner = u
	return nil
}
func (s *notionStoreFake) ConsumeState(_ context.Context, h []byte, _ time.Time) (string, int64, error) {
	if string(h) != string(s.state[:]) {
		return "", 0, domain.ErrNotionUnavailable
	}
	s.state = [32]byte{}
	return s.owner, 0, nil
}
func (s *notionStoreFake) SaveConnection(context.Context, string, int64, string, []byte) error {
	s.connected = true
	return nil
}
func (s *notionStoreFake) Connection(_ context.Context, u string) (domain.NotionConnection, error) {
	return domain.NotionConnection{Linked: s.connected && u == s.owner}, nil
}
func (s *notionStoreFake) Disconnect(_ context.Context, u string) error {
	if u == s.owner {
		s.connected = false
	}
	return nil
}
func TestNotionHandlers(t *testing.T) {
	store := &notionStoreFake{}
	h := &NotionHandler{Svc: &domain.NotionService{Store: store, ClientID: "client", ClientSecret: "secret", RedirectURL: "https://example.test/v1/notion/callback", EncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32))}}
	req := httptest.NewRequest(http.MethodPost, "/v1/notion/link", nil)
	req = req.WithContext(context.WithValue(req.Context(), userctx.UserIDKey, "user"))
	res := httptest.NewRecorder()
	h.Start(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), "authorizeUrl") {
		t.Fatal("link response", res.Code, res.Body.String())
	}
	state := "cancel-state"
	store.state = sha256.Sum256([]byte(state))
	store.owner = "user"
	req = httptest.NewRequest(http.MethodGet, "/v1/notion/callback?state="+url.QueryEscape(state)+"&error=access_denied", nil)
	res = httptest.NewRecorder()
	h.Callback(res, req)
	if res.Code != 302 || res.Header().Get("Location") != "notesapp://notion/linked?status=cancelled" {
		t.Fatal("callback redirect", res.Code, res.Header())
	}
	h.Callback(res, req)
	req = httptest.NewRequest(http.MethodGet, "/v1/notion/connection", nil)
	req = req.WithContext(context.WithValue(req.Context(), userctx.UserIDKey, "user"))
	res = httptest.NewRecorder()
	h.Status(res, req)
	if res.Code != 200 || !strings.Contains(res.Body.String(), `"linked":false`) {
		t.Fatal("status", res.Code, res.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/v1/notion/connection", nil)
	req = req.WithContext(context.WithValue(req.Context(), userctx.UserIDKey, "user"))
	res = httptest.NewRecorder()
	h.Disconnect(res, req)
	if res.Code != 204 || res.Body.Len() != 0 {
		t.Fatal("disconnect", res.Code)
	}
}
