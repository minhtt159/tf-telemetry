// Package middleware provides shared HTTP and gRPC middleware such as basic auth and rate limiting.
package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
)

// RateLimiter maintains per-key limiters backed by installation IDs.
type RateLimiter struct {
	limit    rate.Limit
	burst    int
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

// NewRateLimiter returns a limiter configured from rate limit configuration.
func NewRateLimiter(cfg config.RateLimitConfig) *RateLimiter {
	if !cfg.Enabled || cfg.RequestsPerSecond <= 0 {
		return nil
	}
	return &RateLimiter{
		limit:    rate.Limit(cfg.RequestsPerSecond),
		burst:    cfg.Burst,
		limiters: make(map[string]*rate.Limiter),
	}
}

// Allow returns true if the request for the given key can proceed.
func (r *RateLimiter) Allow(key string) bool {
	if r == nil || r.limit <= 0 {
		return true
	}
	if key == "" {
		key = "unknown"
	}
	limiter := r.limiterForKey(key)
	return limiter.Allow()
}

func (r *RateLimiter) limiterForKey(key string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limiter, ok := r.limiters[key]; ok {
		return limiter
	}
	limiter := rate.NewLimiter(r.limit, r.burst)
	r.limiters[key] = limiter
	return limiter
}

// BasicAuthHTTP wraps an HTTP handler with basic auth validation.
func BasicAuthHTTP(cfg config.BasicAuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok || !credentialsMatch(username, password, cfg) {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitHTTP applies rate limiting using the provided key extractor.
func RateLimitHTTP(limiter *RateLimiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter != nil && !limiter.Allow(keyFn(r)) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// BasicAuthUnary provides gRPC basic auth enforcement.
func BasicAuthUnary(cfg config.BasicAuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		if err := validateBasicAuth(md, cfg); err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return handler(ctx, req)
	}
}

// RateLimitUnary limits gRPC requests using installation IDs from telemetry packets.
func RateLimitUnary(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	if limiter == nil || limiter.limit <= 0 {
		return nil
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		key := installationIDFromRequest(req)
		if !limiter.Allow(key) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

func installationIDFromRequest(req any) string {
	if packet, ok := req.(*pb.TelemetryPacket); ok && packet.GetMetadata() != nil {
		return hex.EncodeToString(packet.GetMetadata().GetInstallationId())
	}
	return ""
}

func validateBasicAuth(md metadata.MD, cfg config.BasicAuthConfig) error {
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return errors.New("authorization header missing")
	}
	const prefix = "Basic "
	header := authHeaders[0]
	if !strings.HasPrefix(header, prefix) {
		return errors.New("invalid authorization header")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return errors.New("invalid base64 in authorization header")
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return errors.New("invalid authorization value")
	}
	if !credentialsMatch(parts[0], parts[1], cfg) {
		return errors.New("invalid credentials")
	}
	return nil
}

func credentialsMatch(username, password string, cfg config.BasicAuthConfig) bool {
	return subtle.ConstantTimeCompare([]byte(username), []byte(cfg.Username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(cfg.Password)) == 1
}
