package httpserver

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
)

// RunHealthcheck performs a healthcheck against the HTTP server.
// Returns 0 on success, 1 on failure.
func RunHealthcheck(cfg *config.Config) int {
	url := os.Getenv("HEALTHCHECK_URL")
	if url == "" {
		host := cfg.Server.BindAddress
		if host == "" {
			host = "127.0.0.1"
		}
		port := cfg.Server.HTTPPort
		if port == 0 {
			port = 8080
		}
		url = fmt.Sprintf("http://%s:%d/healthz", host, port)
	}

	timeout := 2 * time.Second
	if v := os.Getenv("HEALTHCHECK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			timeout = d
		}
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("error closing healthcheck response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
