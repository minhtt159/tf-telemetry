package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
)

// Tests for RateLimiter

func TestNewRateLimiter_Disabled(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           false,
		RequestsPerSecond: 10,
		Burst:             20,
	}
	limiter := NewRateLimiter(cfg)
	if limiter != nil {
		t.Fatal("expected nil limiter when disabled")
	}
}

func TestNewRateLimiter_InvalidRate(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 0,
		Burst:             20,
	}
	limiter := NewRateLimiter(cfg)
	if limiter != nil {
		t.Fatal("expected nil limiter when rate is invalid")
	}
}

func TestNewRateLimiter_Valid(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		Burst:             20,
	}
	limiter := NewRateLimiter(cfg)
	if limiter == nil {
		t.Fatal("expected non-nil limiter")
	}
}

func TestRateLimiter_Allow_NilLimiter(t *testing.T) {
	var limiter *RateLimiter
	if !limiter.Allow("test-key") {
		t.Fatal("expected nil limiter to allow all requests")
	}
}

func TestRateLimiter_Allow_EmptyKey(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 10,
		Burst:             1,
	}
	limiter := NewRateLimiter(cfg)

	// Should use "missing-installation-id" as key
	if !limiter.Allow("") {
		t.Fatal("expected first request to be allowed")
	}
}

func TestRateLimiter_Allow_RateLimiting(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	}
	limiter := NewRateLimiter(cfg)

	key := "test-key"

	// First request should be allowed
	if !limiter.Allow(key) {
		t.Fatal("expected first request to be allowed")
	}

	// Second request should be rate limited
	if limiter.Allow(key) {
		t.Fatal("expected second request to be rate limited")
	}

	// Wait for rate limiter to refill
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	if !limiter.Allow(key) {
		t.Fatal("expected request to be allowed after waiting")
	}
}

func TestRateLimiter_Allow_DifferentKeys(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	}
	limiter := NewRateLimiter(cfg)

	// Different keys should have independent rate limits
	if !limiter.Allow("key1") {
		t.Fatal("expected first request for key1 to be allowed")
	}
	if !limiter.Allow("key2") {
		t.Fatal("expected first request for key2 to be allowed")
	}

	// Second request for key1 should be limited
	if limiter.Allow("key1") {
		t.Fatal("expected second request for key1 to be rate limited")
	}
}

func TestRateLimitHTTP_NoLimiter(t *testing.T) {
	handler := RateLimitHTTP(nil, func(r *http.Request) string { return "key" })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRateLimitHTTP_WithLimiter(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	}
	limiter := NewRateLimiter(cfg)

	handler := RateLimitHTTP(limiter, func(r *http.Request) string { return "test-key" })(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	// First request
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request, got %d", w1.Code)
	}

	// Second request (should be rate limited)
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for second request, got %d", w2.Code)
	}
}

func TestRateLimitUnary_NoLimiter(t *testing.T) {
	interceptor := RateLimitUnary(nil)
	if interceptor != nil {
		t.Fatal("expected nil interceptor for nil limiter")
	}
}

func TestRateLimitUnary_WithLimiter(t *testing.T) {
	cfg := config.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 1,
		Burst:             1,
	}
	limiter := NewRateLimiter(cfg)
	interceptor := RateLimitUnary(limiter)

	if interceptor == nil {
		t.Fatal("expected non-nil interceptor")
	}

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			InstallationId: []byte{0x01, 0x02, 0x03},
		},
	}

	// First request should succeed
	_, err := interceptor(context.Background(), packet, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected first request to succeed, got error: %v", err)
	}

	// Second request should be rate limited
	_, err = interceptor(context.Background(), packet, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		t.Fatal("expected second request to be rate limited")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Fatalf("expected ResourceExhausted error, got: %v", err)
	}
}

func TestInstallationIDFromRequest_ValidPacket(t *testing.T) {
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			InstallationId: []byte{0x01, 0x02, 0x03},
		},
	}

	id := installationIDFromRequest(packet)
	if id == "" {
		t.Fatal("expected non-empty installation ID")
	}
	if id != "010203" {
		t.Fatalf("expected '010203', got '%s'", id)
	}
}

func TestInstallationIDFromRequest_NoMetadata(t *testing.T) {
	packet := &pb.TelemetryPacket{}
	id := installationIDFromRequest(packet)
	if id != "" {
		t.Fatalf("expected empty installation ID, got '%s'", id)
	}
}

func TestInstallationIDFromRequest_WrongType(t *testing.T) {
	id := installationIDFromRequest("not a packet")
	if id != "" {
		t.Fatalf("expected empty installation ID for wrong type, got '%s'", id)
	}
}

// Tests for BasicAuth

func TestBasicAuthHTTP_Valid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	handler := BasicAuthHTTP(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.SetBasicAuth("user", "pass")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestBasicAuthHTTP_Invalid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	handler := BasicAuthHTTP(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"wrong username", "wrong", "pass"},
		{"wrong password", "user", "wrong"},
		{"both wrong", "wrong", "wrong"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.SetBasicAuth(tt.username, tt.password)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", w.Code)
			}
		})
	}
}

func TestBasicAuthHTTP_NoAuth(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	handler := BasicAuthHTTP(cfg)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestBasicAuthUnary_Valid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	interceptor := BasicAuthUnary(cfg)
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	md := metadata.Pairs("authorization", "Basic "+auth)
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestBasicAuthUnary_Invalid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	tests := []struct {
		name string
		auth string
	}{
		{"wrong credentials", base64.StdEncoding.EncodeToString([]byte("wrong:wrong"))},
		{"invalid base64", "not-base64"},
		{"no colon", base64.StdEncoding.EncodeToString([]byte("nocolon"))},
		{"wrong prefix", "Bearer token"},
	}

	interceptor := BasicAuthUnary(cfg)
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.Pairs("authorization", "Basic "+tt.auth)
			ctx := metadata.NewIncomingContext(context.Background(), md)

			_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
			if err == nil {
				t.Fatal("expected error")
			}

			st, ok := status.FromError(err)
			if !ok || st.Code() != codes.Unauthenticated {
				t.Fatalf("expected Unauthenticated error, got: %v", err)
			}
		})
	}
}

func TestBasicAuthUnary_NoMetadata(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	interceptor := BasicAuthUnary(cfg)
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		t.Fatal("expected error for missing metadata")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated error, got: %v", err)
	}
}

func TestBasicAuthUnary_NoAuthHeader(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Enabled:  true,
		Username: "user",
		Password: "pass",
	}

	interceptor := BasicAuthUnary(cfg)
	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	md := metadata.Pairs("other-header", "value")
	ctx := metadata.NewIncomingContext(context.Background(), md)

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, handler)
	if err == nil {
		t.Fatal("expected error for missing auth header")
	}
}

func TestCredentialsMatch_Valid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	if !credentialsMatch("user", "pass", cfg) {
		t.Fatal("expected credentials to match")
	}
}

func TestCredentialsMatch_Invalid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	tests := []struct {
		name     string
		username string
		password string
	}{
		{"wrong username", "wrong", "pass"},
		{"wrong password", "user", "wrong"},
		{"both wrong", "wrong", "wrong"},
		{"case sensitive", "User", "Pass"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if credentialsMatch(tt.username, tt.password, cfg) {
				t.Fatal("expected credentials not to match")
			}
		})
	}
}

func TestValidateBasicAuth_Valid(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	md := metadata.Pairs("authorization", "Basic "+auth)

	if err := validateBasicAuth(md, cfg); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateBasicAuth_MissingHeader(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	md := metadata.Pairs("other", "value")

	err := validateBasicAuth(md, cfg)
	if err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestValidateBasicAuth_InvalidPrefix(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	md := metadata.Pairs("authorization", "Bearer token")

	err := validateBasicAuth(md, cfg)
	if err == nil {
		t.Fatal("expected error for invalid prefix")
	}
}

func TestValidateBasicAuth_InvalidBase64(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	md := metadata.Pairs("authorization", "Basic not-valid-base64!!!")

	err := validateBasicAuth(md, cfg)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestValidateBasicAuth_NoColon(t *testing.T) {
	cfg := config.BasicAuthConfig{
		Username: "user",
		Password: "pass",
	}

	auth := base64.StdEncoding.EncodeToString([]byte("nocolon"))
	md := metadata.Pairs("authorization", "Basic "+auth)

	err := validateBasicAuth(md, cfg)
	if err == nil {
		t.Fatal("expected error for missing colon")
	}
}

// Tests for CORS Middleware

func TestCorsMiddleware_Disabled(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: false,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Should not have CORS headers when disabled
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected no CORS headers when disabled")
	}
}

func TestCorsMiddleware_WildcardOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard origin, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCorsMiddleware_SpecificOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"http://localhost:3000", "http://example.com"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	tests := []struct {
		name           string
		origin         string
		expectOrigin   string
		expectCORS     bool
	}{
		{"allowed origin 1", "http://localhost:3000", "http://localhost:3000", true},
		{"allowed origin 2", "http://example.com", "http://example.com", true},
		{"disallowed origin", "http://evil.com", "", false},
		{"no origin header", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()

			corsHandler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", w.Code)
			}

			gotOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectCORS {
				if gotOrigin != tt.expectOrigin {
					t.Fatalf("expected origin %s, got %s", tt.expectOrigin, gotOrigin)
				}
			} else {
				if gotOrigin != "" {
					t.Fatalf("expected no CORS headers for disallowed origin, got %s", gotOrigin)
				}
			}
		})
	}
}

func TestCorsMiddleware_DefaultMethodsAndHeaders(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
		// Don't set methods to test defaults
		// Headers are not set - should not be present in response
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Fatal("expected default methods to be set")
	}

	// Headers should NOT be set when not explicitly configured
	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "" {
		t.Fatalf("expected no headers when not explicitly configured, got: %s", headers)
	}

	// Check for expected default methods
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	for _, method := range expectedMethods {
		if !strings.Contains(methods, method) {
			t.Errorf("expected method %s to be in default methods: %s", method, methods)
		}
	}
}

func TestCorsMiddleware_CustomMethodsAndHeaders(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "X-Custom-Header"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods != "GET, POST" {
		t.Fatalf("expected 'GET, POST', got %s", methods)
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers != "Content-Type, X-Custom-Header" {
		t.Fatalf("expected 'Content-Type, X-Custom-Header', got %s", headers)
	}
}

func TestCorsMiddleware_OptionsRequest(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be called for OPTIONS requests
		t.Fatal("handler should not be called for OPTIONS requests")
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal("expected CORS headers for OPTIONS request")
	}
}

func TestCorsMiddleware_PrivateNetworkAccess_Preflight(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This should not be called for OPTIONS requests
		t.Fatal("handler should not be called for OPTIONS requests")
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for OPTIONS, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Fatalf("expected Access-Control-Allow-Private-Network: true, got %s",
			w.Header().Get("Access-Control-Allow-Private-Network"))
	}
}

func TestCorsMiddleware_PrivateNetworkAccess_ActualRequest(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	if w.Header().Get("Access-Control-Allow-Private-Network") != "true" {
		t.Fatalf("expected Access-Control-Allow-Private-Network: true, got %s",
			w.Header().Get("Access-Control-Allow-Private-Network"))
	}
}

func TestCorsMiddleware_NoPrivateNetworkHeader(t *testing.T) {
	cfg := config.CORSConfig{
		Enabled: true,
		AllowedOrigins: []string{"*"},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	corsHandler := CorsMiddleware(handler, cfg)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	corsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Should not have Private Network header when not requested
	if w.Header().Get("Access-Control-Allow-Private-Network") != "" {
		t.Fatal("expected no Access-Control-Allow-Private-Network header when not requested")
	}
}
