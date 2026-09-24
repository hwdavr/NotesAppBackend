//go:build integration

package http

import (
	stdhttp "net/http"
	"strings"
	"testing"
)

func TestAPIHealthAndAuthenticationEndpoints(t *testing.T) {
	const userID = "api-e2e-health-user"
	env := newAPIE2EEnv(t, userID)

	status, header, body := env.request(t, stdhttp.MethodGet, "/healthz", "", nil)
	if status != stdhttp.StatusOK || !strings.HasPrefix(header.Get("Content-Type"), "text/plain") || string(body) != "ok" {
		t.Fatal("health endpoint did not return the expected plain-text response")
	}

	status, _, _ = env.request(t, stdhttp.MethodGet, "/v1/items", "", nil)
	if status != stdhttp.StatusUnauthorized {
		t.Fatalf("missing-token list status = %d, want %d", status, stdhttp.StatusUnauthorized)
	}

	invalidAudienceToken := signAPIIntegrationToken(t, env.privateKey, userID, apiE2EInvalidAudience)
	status, _, _ = env.request(t, stdhttp.MethodGet, "/v1/debug/me", invalidAudienceToken, nil)
	if status != stdhttp.StatusUnauthorized {
		t.Fatalf("invalid-audience debug status = %d, want %d", status, stdhttp.StatusUnauthorized)
	}

	status, header, body = env.authenticatedRequest(t, stdhttp.MethodGet, "/v1/debug/me", nil)
	assertAPIJSONResponse(t, status, header, stdhttp.StatusOK)
	var identity struct {
		UserID string `json:"userId"`
	}
	decodeAPIJSON(t, body, &identity)
	if identity.UserID != userID {
		t.Fatal("debug identity response did not contain the verified JWT subject")
	}
}
