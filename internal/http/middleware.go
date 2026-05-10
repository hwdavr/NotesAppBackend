package http

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hwdavr/notes-app-backend/internal/pkg/userctx"
	"go.uber.org/zap"
)

type AuthConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

type jwksCache struct {
	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	N   string `json:"n"`
	E   string `json:"e"`
}

var auth0JWKSCache jwksCache

func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if t.Method.Alg() != jwt.SigningMethodRS256.Alg() {
					return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
				}

				kid, _ := t.Header["kid"].(string)
				if kid == "" {
					return nil, errors.New("missing kid header")
				}

				return getJWKSKey(r.Context(), cfg.JWKSURL, kid)
			},
				jwt.WithValidMethods([]string{"RS256"}),
				jwt.WithIssuer(cfg.Issuer),
				jwt.WithAudience(cfg.Audience),
				jwt.WithExpirationRequired(),
				jwt.WithLeeway(30*time.Second),
			)
			if err != nil || !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			uid, ok := claims["sub"].(string)
			if !ok || uid == "" {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			email, _ := claims["email"].(string)
			if email == "" {
				// Fallback to custom claim if standard email is missing
				email, _ = claims["https://notes-app.api/email"].(string)
			}
			email = strings.ToLower(email)
			ctx := context.WithValue(r.Context(), userctx.UserIDKey, uid)
			ctx = context.WithValue(ctx, userctx.UserEmailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func getJWKSKey(ctx context.Context, jwksURL, kid string) (*rsa.PublicKey, error) {
	if key, ok := auth0JWKSCache.get(kid); ok {
		return key, nil
	}

	keys, err := fetchJWKS(ctx, jwksURL)
	if err != nil {
		return nil, err
	}

	auth0JWKSCache.set(keys, time.Now().Add(5*time.Minute))

	key, ok := auth0JWKSCache.get(kid)
	if !ok {
		return nil, fmt.Errorf("kid %q not found in jwks", kid)
	}
	return key, nil
}

func fetchJWKS(ctx context.Context, jwksURL string) (map[string]*rsa.PublicKey, error) {
	parsedURL, err := url.Parse(jwksURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks fetch returned %s", resp.Status)
	}

	var payload jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, key := range payload.Keys {
		if key.Kty != "RSA" || key.Kid == "" {
			continue
		}

		publicKey, err := buildRSAPublicKey(key.N, key.E)
		if err != nil {
			return nil, err
		}
		keys[key.Kid] = publicKey
	}
	return keys, nil
}

func buildRSAPublicKey(modulusB64, exponentB64 string) (*rsa.PublicKey, error) {
	modulusBytes, err := base64.RawURLEncoding.DecodeString(modulusB64)
	if err != nil {
		return nil, err
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(exponentB64)
	if err != nil {
		return nil, err
	}

	exponent := 0
	for _, b := range exponentBytes {
		exponent = exponent<<8 + int(b)
	}
	if exponent == 0 {
		return nil, errors.New("invalid rsa exponent")
	}

	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulusBytes),
		E: exponent,
	}, nil
}

func (c *jwksCache) get(kid string) (*rsa.PublicKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if time.Now().After(c.expiresAt) {
		return nil, false
	}

	key, ok := c.keys[kid]
	return key, ok
}

func (c *jwksCache) set(keys map[string]*rsa.PublicKey, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.keys = keys
	c.expiresAt = expiresAt
}

func Logger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(ww, r)

			log.Info("request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.status),
				zap.Duration("duration", time.Since(start)),
				zap.String("ip", r.RemoteAddr),
			)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
