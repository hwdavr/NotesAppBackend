package domain

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type notionMemory struct {
	hashes     map[[32]byte]string
	generation int64
	token      []byte
	connection NotionConnection
}

func (m *notionMemory) CreateState(_ context.Context, u string, h []byte, _ time.Time) error {
	var k [32]byte
	copy(k[:], h)
	m.hashes[k] = u
	return nil
}
func (m *notionMemory) ConsumeState(_ context.Context, h []byte, _ time.Time) (string, int64, error) {
	var k [32]byte
	copy(k[:], h)
	u := m.hashes[k]
	delete(m.hashes, k)
	if u == "" {
		return "", 0, errors.New("missing")
	}
	return u, m.generation, nil
}
func (m *notionMemory) SaveConnection(_ context.Context, _ string, g int64, n string, t []byte) error {
	if g != m.generation {
		return errors.New("disconnected")
	}
	m.token = t
	m.connection = NotionConnection{Linked: true, AccountName: &n}
	return nil
}
func (m *notionMemory) Connection(context.Context, string) (NotionConnection, error) {
	return m.connection, nil
}
func (m *notionMemory) Disconnect(context.Context, string) error {
	m.generation++
	m.hashes = map[[32]byte]string{}
	m.connection = NotionConnection{}
	return nil
}

type notionRoundTrip func(*http.Request) (*http.Response, error)

func (f notionRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNotionLinkAndState(t *testing.T) {
	store := &notionMemory{hashes: map[[32]byte]string{}}
	client := &http.Client{Transport: notionRoundTrip(func(r *http.Request) (*http.Response, error) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "client" || p != "secret" {
			t.Error("incorrect client authentication")
		}
		var body map[string]string
		if json.NewDecoder(r.Body).Decode(&body) != nil || body["code"] != "code" {
			t.Error("incorrect exchange")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"access_token":"sensitive-token","workspace_name":"Workspace"}`))}, nil
	})}

	svc := &NotionService{Store: store, ClientID: "client", ClientSecret: "secret", RedirectURL: "https://example.test/v1/notion/callback", EncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), TokenURL: "https://notion.test/token", Client: client}
	auth, err := svc.Start(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(auth)
	if err != nil {
		t.Fatal(err)
	}
	state := u.Query().Get("state")
	if state == "" || u.Query().Get("client_id") != "client" {
		t.Fatal("invalid authorization URL")
	}
	hash := sha256.Sum256([]byte(state))
	if store.hashes[hash] != "user" {
		t.Fatal("state not bound to user")
	}
	if got := svc.Callback(context.Background(), state, "code", ""); got != "linked" {
		t.Fatalf("callback = %s", got)
	}
	if strings.Contains(string(store.token), "sensitive-token") || len(store.token) == 0 {
		t.Fatal("token not encrypted")
	}
	if !store.connection.Linked || *store.connection.AccountName != "Workspace" {
		t.Fatal("connection missing")
	}
	if got := svc.Callback(context.Background(), state, "code", ""); got != "failed" {
		t.Fatal("state replay accepted")
	}
	auth, err = svc.Start(context.Background(), "user")
	if err != nil {
		t.Fatal(err)
	}
	u, _ = url.Parse(auth)
	if got := svc.Callback(context.Background(), u.Query().Get("state"), "", "access_denied"); got != "cancelled" {
		t.Fatalf("cancellation = %s", got)
	}
	if err := svc.Disconnect(context.Background(), "user"); err != nil {
		t.Fatal(err)
	}
	c, err := svc.Status(context.Background(), "user")
	if err != nil || c.Linked {
		t.Fatal("disconnect failed")
	}
}
func TestNotionConfigurationAndFailure(t *testing.T) {
	s := &NotionService{}
	if _, err := s.Start(context.Background(), "user"); !errors.Is(err, ErrNotionUnavailable) {
		t.Fatal("missing config accepted")
	}
	if s.Callback(context.Background(), "bad", "code", "") != "failed" {
		t.Fatal("invalid state accepted")
	}
}

func TestNotionUpstreamFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{{"upstream error", 503, `{}`}, {"malformed response", 200, `{`}, {"missing token", 200, `{"workspace_name":"example"}`}} {
		t.Run(tc.name, func(t *testing.T) {
			store := &notionMemory{hashes: map[[32]byte]string{}}
			client := &http.Client{Transport: notionRoundTrip(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}
			svc := &NotionService{Store: store, ClientID: "client", ClientSecret: "secret", RedirectURL: "https://example.test/v1/notion/callback", EncryptionKey: base64.StdEncoding.EncodeToString(make([]byte, 32)), TokenURL: "https://notion.test/token", Client: client}
			auth, err := svc.Start(context.Background(), "user")
			if err != nil {
				t.Fatal(err)
			}
			u, err := url.Parse(auth)
			if err != nil {
				t.Fatal(err)
			}
			if got := svc.Callback(context.Background(), u.Query().Get("state"), "code", ""); got != "failed" {
				t.Fatalf("callback = %s", got)
			}
			if store.connection.Linked || len(store.token) != 0 {
				t.Fatal("failed exchange saved a connection")
			}
		})
	}
}
