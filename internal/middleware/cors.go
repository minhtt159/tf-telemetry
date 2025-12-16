package middleware

import (
	"net/http"
	"strings"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
)

func CorsMiddleware(next http.Handler, cfg config.CORSConfig) http.Handler {
	if !cfg.Enabled {
		return next
	}

	// Set sensible defaults if not configured
	allowedMethods := cfg.AllowedMethods
	if len(allowedMethods) == 0 {
		allowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	allowedHeaders := cfg.AllowedHeaders
	if len(allowedHeaders) == 0 {
		allowedHeaders = []string{"Content-Type", "Authorization", "X-Requested-With"}
	}

	// Join methods and headers for header values
	methodsStr := strings.Join(allowedMethods, ", ")
	headersStr := strings.Join(allowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// If no origin header, skip CORS (not a cross-origin request)
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Check if origin is allowed
		allowedOrigin := ""
		if len(cfg.AllowedOrigins) == 0 {
			// No allowed origins configured, default to wildcard
			allowedOrigin = "*"
		} else {
			for _, allowedOrig := range cfg.AllowedOrigins {
				if allowedOrig == "*" || allowedOrig == origin {
					allowedOrigin = allowedOrig
					break
				}
			}
		}

		// If origin not allowed, don't set CORS headers
		if allowedOrigin == "" {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", methodsStr)
		w.Header().Set("Access-Control-Allow-Headers", headersStr)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}


