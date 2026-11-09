package domain

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

var ErrNotionUnavailable = errors.New("notion service unavailable")

type NotionConnection struct {
	Linked      bool    `json:"linked"`
	AccountName *string `json:"accountName,omitempty"`
}
type NotionStore interface {
	CreateState(context.Context, string, []byte, time.Time) error
	ConsumeState(context.Context, []byte, time.Time) (string, int64, error)
	SaveConnection(context.Context, string, int64, string, []byte) error
	Connection(context.Context, string) (NotionConnection, error)
	Disconnect(context.Context, string) error
}
type NotionService struct {
	Store                                              NotionStore
	ClientID, ClientSecret, RedirectURL, EncryptionKey string
	Client                                             *http.Client
	AuthorizeURL, TokenURL                             string
	Now                                                func() time.Time
}

func (s *NotionService) configured() bool {
	key, err := base64.StdEncoding.DecodeString(s.EncryptionKey)
	u, e := url.Parse(s.RedirectURL)
	return s.Store != nil && s.ClientID != "" && s.ClientSecret != "" && err == nil && len(key) == 32 && e == nil && u.IsAbs() && u.Host != "" && (u.Scheme == "https" || u.Scheme == "http")
}
func (s *NotionService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
func (s *NotionService) Start(ctx context.Context, user string) (string, error) {
	if !s.configured() || user == "" {
		return "", ErrNotionUnavailable
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", ErrNotionUnavailable
	}
	state := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(state))
	if err := s.Store.CreateState(ctx, user, sum[:], s.now().Add(10*time.Minute)); err != nil {
		return "", ErrNotionUnavailable
	}
	target := s.AuthorizeURL
	if target == "" {
		target = "https://api.notion.com/v1/oauth/authorize"
	}
	u, err := url.Parse(target)
	if err != nil {
		return "", ErrNotionUnavailable
	}
	q := u.Query()
	q.Set("client_id", s.ClientID)
	q.Set("response_type", "code")
	q.Set("owner", "user")
	q.Set("redirect_uri", s.RedirectURL)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
func (s *NotionService) Callback(ctx context.Context, state, code, oauthError string) string {
	if !s.configured() || state == "" {
		return "failed"
	}
	sum := sha256.Sum256([]byte(state))
	user, generation, err := s.Store.ConsumeState(ctx, sum[:], s.now())
	if err != nil {
		return "failed"
	}
	if oauthError == "access_denied" {
		return "cancelled"
	}
	if oauthError != "" || code == "" {
		return "failed"
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	body, _ := json.Marshal(map[string]string{"grant_type": "authorization_code", "code": code, "redirect_uri": s.RedirectURL})
	target := s.TokenURL
	if target == "" {
		target = "https://api.notion.com/v1/oauth/token"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return "failed"
	}
	req.SetBasicAuth(s.ClientID, s.ClientSecret)
	req.Header.Set("Content-Type", "application/json")
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return "failed"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "failed"
	}
	var payload struct {
		AccessToken   string `json:"access_token"`
		WorkspaceName string `json:"workspace_name"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload) != nil || payload.AccessToken == "" {
		return "failed"
	}
	key, _ := base64.StdEncoding.DecodeString(s.EncryptionKey)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "failed"
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "failed"
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "failed"
	}
	encrypted := gcm.Seal(nonce, nonce, []byte(payload.AccessToken), []byte(user))
	if s.Store.SaveConnection(ctx, user, generation, payload.WorkspaceName, encrypted) != nil {
		return "failed"
	}
	return "linked"
}
func (s *NotionService) Status(ctx context.Context, user string) (NotionConnection, error) {
	if s.Store == nil || user == "" {
		return NotionConnection{}, ErrNotionUnavailable
	}
	c, err := s.Store.Connection(ctx, user)
	if err != nil {
		return NotionConnection{}, ErrNotionUnavailable
	}
	return c, nil
}
func (s *NotionService) Disconnect(ctx context.Context, user string) error {
	if s.Store == nil || user == "" {
		return ErrNotionUnavailable
	}
	if s.Store.Disconnect(ctx, user) != nil {
		return ErrNotionUnavailable
	}
	return nil
}
