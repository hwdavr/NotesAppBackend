//go:build integration

package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hwdavr/notes-app-backend/internal/config"
	"github.com/hwdavr/notes-app-backend/internal/db"
	"github.com/hwdavr/notes-app-backend/internal/domain"
	"github.com/hwdavr/notes-app-backend/internal/http/handlers"
	"go.uber.org/zap"
)

const (
	apiE2EDeviceID         = "api-e2e-device"
	apiE2EKID              = "api-e2e-jwks-key"
	apiE2EIssuer           = "https://notes-api-e2e.test/"
	apiE2EAudience         = "https://notes-api-e2e.test/audience"
	apiE2EInvalidAudience  = "https://notes-api-e2e.test/other-audience"
	apiE2ERequestTimeout   = 5 * time.Second
	apiE2EDatabaseDeadline = 15 * time.Second
)

type apiE2EEnv struct {
	baseURL     string
	client      *stdhttp.Client
	bearerToken string
	privateKey  *rsa.PrivateKey
}

type silentEmailService struct{}

func (silentEmailService) SendInvite(string, string, string) error {
	return nil
}

func newAPIE2EEnv(t *testing.T, userID string) *apiE2EEnv {
	t.Helper()

	database := openAPIIntegrationDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), apiE2EDatabaseDeadline)
	defer cancel()
	clearAPIIntegrationData(t, ctx, database, userID)
	t.Cleanup(func() {
		cleanupContext, cleanupCancel := context.WithTimeout(context.Background(), apiE2ERequestTimeout)
		defer cleanupCancel()
		clearAPIIntegrationData(t, cleanupContext, database, userID)
	})

	auth0JWKSCache = jwksCache{}
	t.Cleanup(func() { auth0JWKSCache = jwksCache{} })

	privateKey, jwksServer := newAPIIntegrationJWKS(t)
	t.Cleanup(jwksServer.Close)

	service := domain.NewService(domain.NewRepository(database), silentEmailService{})
	log := zap.NewNop()
	router := NewRouter(
		&handlers.ItemsHandler{Svc: service, Log: log},
		&handlers.SharesHandler{Svc: service, Log: log},
		&handlers.CommentsHandler{Svc: service, Log: log},
		config.Config{
			Auth0Issuer:   apiE2EIssuer,
			Auth0Audience: apiE2EAudience,
			Auth0JWKSURL:  jwksServer.URL,
		},
		log,
	)
	apiServer := httptest.NewServer(router)
	t.Cleanup(apiServer.Close)

	client := apiServer.Client()
	client.Timeout = apiE2ERequestTimeout
	return &apiE2EEnv{
		baseURL:     apiServer.URL,
		client:      client,
		bearerToken: signAPIIntegrationToken(t, privateKey, userID, apiE2EAudience),
		privateKey:  privateKey,
	}
}

func openAPIIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL must point to the disposable integration database")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		t.Fatalf("connect to integration database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func clearAPIIntegrationData(t *testing.T, ctx context.Context, database *sql.DB, userID string) {
	t.Helper()

	queries := []string{
		`DELETE FROM note_block_comments WHERE note_id IN (SELECT id FROM items WHERE user_id = $1)`,
		`DELETE FROM note_shares WHERE note_id IN (SELECT id FROM items WHERE user_id = $1)`,
		`DELETE FROM items WHERE user_id = $1`,
	}
	for _, query := range queries {
		if _, err := database.ExecContext(ctx, query, userID); err != nil {
			t.Fatalf("clear API integration data: %v", err)
		}
	}
}

func newAPIIntegrationJWKS(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test JWT key: %v", err)
	}
	payload, err := json.Marshal(jwksResponse{Keys: []jwk{{
		Kid: apiE2EKID,
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
	}}})
	if err != nil {
		t.Fatalf("encode test JWKS: %v", err)
	}

	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	}))
	return privateKey, server
}

func signAPIIntegrationToken(t *testing.T, privateKey *rsa.PrivateKey, userID, audience string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"aud": audience,
		"exp": time.Now().Add(5 * time.Minute).Unix(),
		"iss": apiE2EIssuer,
		"sub": userID,
	})
	token.Header["kid"] = apiE2EKID
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("sign test JWT: %v", err)
	}
	return signedToken
}

func (env *apiE2EEnv) request(t *testing.T, method, path, bearerToken string, payload any) (int, stdhttp.Header, []byte) {
	t.Helper()
	return apiRequest(t, env.client, method, env.baseURL+path, bearerToken, payload)
}

func (env *apiE2EEnv) authenticatedRequest(t *testing.T, method, path string, payload any) (int, stdhttp.Header, []byte) {
	t.Helper()
	return env.request(t, method, path, env.bearerToken, payload)
}

func apiRequest(t *testing.T, client *stdhttp.Client, method, url, bearerToken string, payload any) (int, stdhttp.Header, []byte) {
	t.Helper()

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode API request: %v", err)
		}
		body = bytes.NewReader(encoded)
	}

	request, err := stdhttp.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build API request: %v", err)
	}
	if bearerToken != "" {
		request.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send API request: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read API response: %v", err)
	}
	return response.StatusCode, response.Header, responseBody
}

func assertAPIJSONResponse(t *testing.T, status int, header stdhttp.Header, wantStatus int) {
	t.Helper()

	if status != wantStatus {
		t.Fatalf("API status = %d, want %d", status, wantStatus)
	}
	if !strings.HasPrefix(header.Get("Content-Type"), "application/json") {
		t.Fatalf("API content type = %q, want application/json", header.Get("Content-Type"))
	}
}

func decodeAPIJSON(t *testing.T, body []byte, destination any) {
	t.Helper()

	if err := json.Unmarshal(body, destination); err != nil {
		t.Fatalf("decode API JSON response: %v", err)
	}
}

func createAPIFolder(t *testing.T, env *apiE2EEnv, id, name string) domain.Item {
	t.Helper()

	status, header, body := env.authenticatedRequest(t, stdhttp.MethodPost, "/v1/folders", map[string]any{
		"id":       id,
		"name":     name,
		"sortKey":  "a0",
		"deviceId": apiE2EDeviceID,
	})
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)

	var item domain.Item
	decodeAPIJSON(t, body, &item)
	if item.ID != id || item.Type != domain.ItemTypeFolder || item.Version != 1 {
		t.Fatal("folder create response did not contain the expected version-1 folder")
	}
	return item
}

func createAPINote(t *testing.T, env *apiE2EEnv, id, parentID, name string) domain.Item {
	t.Helper()

	payload := map[string]any{
		"id":       id,
		"name":     name,
		"content":  "initial integration content",
		"sortKey":  "a0",
		"deviceId": apiE2EDeviceID,
	}
	if parentID != "" {
		payload["parentId"] = parentID
	}
	status, header, body := env.authenticatedRequest(t, stdhttp.MethodPost, "/v1/notes", payload)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusCreated)

	var item domain.Item
	decodeAPIJSON(t, body, &item)
	if item.ID != id || item.Type != domain.ItemTypeNote || item.Version != 1 {
		t.Fatal("note create response did not contain the expected version-1 note")
	}
	return item
}

func getAPIItem(t *testing.T, env *apiE2EEnv, itemID string) domain.Item {
	t.Helper()

	status, header, body := env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/items/"+itemID, nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)

	var item domain.Item
	decodeAPIJSON(t, body, &item)
	return item
}

func mutateAPIItem(t *testing.T, env *apiE2EEnv, method, path string, payload any) domain.MutationResult {
	t.Helper()

	status, header, body := env.authenticatedRequest(t, method, path, payload)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)

	var result domain.MutationResult
	decodeAPIJSON(t, body, &result)
	if result.Status != "merged" {
		t.Fatal("item mutation response was not merged")
	}
	return result
}
