package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/minhtt159/tf-telemetry/internal/config"
)

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
