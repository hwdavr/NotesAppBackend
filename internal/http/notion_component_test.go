package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"go.uber.org/zap"
)

type notionComponentStore struct {
	states      map[[32]byte]string
	connections map[string]domain.NotionConnection
}

func (s *notionComponentStore) CreateState(_ context.Context, u string, h []byte, _ time.Time) error {
	var k [32]byte
	copy(k[:], h)
	s.states[k] = u
	return nil
}
func (s *notionComponentStore) ConsumeState(_ context.Context, h []byte, _ time.Time) (string, int64, error) {
	var k [32]byte
	copy(k[:], h)
	u := s.states[k]
	delete(s.states, k)
	if u == "" {
		return "", 0, errors.New("invalid state")
	}
	return u, 0, nil
}
func (s *notionComponentStore) SaveConnection(_ context.Context, u string, _ int64, n string, _ []byte) error {
	s.connections[u] = domain.NotionConnection{Linked: true, AccountName: &n}
	return nil
}
func (s *notionComponentStore) Connection(_ context.Context, u string) (domain.NotionConnection, error) {
	return s.connections[u], nil
}
func (s *notionComponentStore) Disconnect(_ context.Context, u string) error {
	delete(s.connections, u)
	return nil
}

type notionTransport func(*http.Request) (*http.Response, error)

func (f notionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestNotionAuthenticatedComponent(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	auth0JWKSCache.set(map[string]*rsa.PublicKey{"notion-test": &key.PublicKey}, time.Now().Add(time.Minute))
	t.Cleanup(func() { auth0JWKSCache = jwksCache{} })
	token := func(user string) string {
		c := jwt.MapClaims{"iss": "https://test.invalid/", "aud": "test-api", "sub": user, "exp": time.Now().Add(time.Minute).Unix()}
		v := jwt.NewWithClaims(jwt.SigningMethodRS256, c)
		v.Header["kid"] = "notion-test"
		s, e := v.SignedString(key)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	store := &notionComponentStore{states: map[[32]byte]string{}, connections: map[string]domain.NotionConnection{}}
	client := &http.Client{Transport: notionTransport(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"synthetic-secret","workspace_name":"Test workspace"}`))}, nil
	})}
	svc := &domain.NotionService{Store: store, ClientID: "client", ClientSecret: "secret", RedirectURL: "https://test.invalid/v1/notion/callback", EncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), TokenURL: "https://test.invalid/token", Client: client}
	router := NewRouter(nil, nil, nil, config.Config{Auth0Issuer: "https://test.invalid/", Auth0Audience: "test-api"}, zap.NewNop(), &handlers.NotionHandler{Svc: svc})
	call := func(method, path, bearer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if validator := os.Getenv("NOTION_CONFORMANCE_VALIDATOR"); validator != "" {
			receipt := map[string]any{"method": method, "path": r.URL.Path, "status": w.Code, "contentType": w.Header().Get("Content-Type"), "location": w.Header().Get("Location"), "body": w.Body.String()}
			payload, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("node", validator)
			cmd.Stdin = bytes.NewReader(payload)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("contract validation failed: %v %s", err, output)
			}
		}
		return w
	}
	a, b := token("user-a"), token("user-b")
	w := call("POST", "/v1/notion/link", a)
	if w.Code != 200 {
		t.Fatalf("start: %d", w.Code)
	}
	var link struct {
		URL string `json:"authorizeUrl"`
	}
	if json.Unmarshal(w.Body.Bytes(), &link) != nil {
		t.Fatal("invalid link JSON")
	}
	parsed, err := url.Parse(link.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatal("missing state")
	}
	w = call("GET", "/v1/notion/callback?state="+url.QueryEscape(state)+"&code=synthetic-code", "")
	if w.Code != 302 || w.Header().Get("Location") != "notesapp://notion/linked?status=linked" {
		t.Fatalf("callback: %d %s", w.Code, w.Header().Get("Location"))
	}
	w = call("GET", "/v1/notion/connection", a)
	if w.Code != 200 || w.Body.String() != `{"linked":true,"accountName":"Test workspace"}`+"\n" {
		t.Fatalf("linked response: %d %s", w.Code, w.Body.String())
	}
	w = call("GET", "/v1/notion/connection", b)
	if w.Code != 200 || w.Body.String() != `{"linked":false}`+"\n" {
		t.Fatalf("cross-user response: %d %s", w.Code, w.Body.String())
	}
	w = call("GET", "/v1/notion/callback?state="+url.QueryEscape(state)+"&code=synthetic-code", "")
	if w.Header().Get("Location") != "notesapp://notion/linked?status=failed" {
		t.Fatal("replay accepted")
	}
	w = call("DELETE", "/v1/notion/connection", a)
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatalf("delete: %d", w.Code)
	}
	w = call("GET", "/v1/notion/connection", a)
	if w.Body.String() != `{"linked":false}`+"\n" {
		t.Fatal("connection survived delete")
	}
	w = call("POST", "/v1/notion/link", "")
	if w.Code != 401 {
		t.Fatal("missing authentication accepted")
	}
	svc.ClientID = ""
	w = call("POST", "/v1/notion/link", a)
	if w.Code != 503 {
		t.Fatalf("missing configuration: %d", w.Code)
	}
}
