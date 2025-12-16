package middleware

import (
	"net/http"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
)

func CorsMiddleware(next http.Handler, cfg config.CORSConfig) http.Handler {
	if !cfg.Enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := "*"
		if len(cfg.AllowedOrigins) > 0 {
			allowed := false
			for _, allowedOrig := range cfg.AllowedOrigins {
				if allowedOrig == "*" || allowedOrig == origin {
					allowedOrigin = allowedOrig
					allowed = true
					break
				}
			}
			if !allowed && origin != "" {
				// Origin not in allowed list, don't set CORS headers
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
