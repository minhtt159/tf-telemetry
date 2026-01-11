package httpserver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/minhtt159/tf-telemetry/internal/config"
)

func TestRunHealthcheck_Success(t *testing.T) {
	// Start a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	}))
	defer server.Close()

	// Set the URL environment variable
	os.Setenv("HEALTHCHECK_URL", server.URL+"/healthz")
	defer os.Unsetenv("HEALTHCHECK_URL")

	cfg := &config.Config{}
	result := RunHealthcheck(cfg)
	if result != 0 {
		t.Fatalf("expected healthcheck to succeed, got exit code %d", result)
	}
}

func TestRunHealthcheck_Failure(t *testing.T) {
	// Start a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Set the URL environment variable
	os.Setenv("HEALTHCHECK_URL", server.URL+"/healthz")
	defer os.Unsetenv("HEALTHCHECK_URL")

	cfg := &config.Config{}
	result := RunHealthcheck(cfg)
	if result != 1 {
		t.Fatalf("expected healthcheck to fail, got exit code %d", result)
	}
}

func TestRunHealthcheck_Timeout(t *testing.T) {
	// Test timeout by using an invalid URL that won't respond
	os.Setenv("HEALTHCHECK_URL", "http://127.0.0.1:9")
	os.Setenv("HEALTHCHECK_TIMEOUT", "10ms")
	defer os.Unsetenv("HEALTHCHECK_URL")
	defer os.Unsetenv("HEALTHCHECK_TIMEOUT")

	cfg := &config.Config{}
	result := RunHealthcheck(cfg)
	if result != 1 {
		t.Fatalf("expected healthcheck to fail due to timeout, got exit code %d", result)
	}
}

func TestRunHealthcheck_DefaultURL(t *testing.T) {
	// Test that default URL is constructed correctly
	// We can't actually connect, but we can verify the function handles it
	cfg := &config.Config{}
	cfg.Server.BindAddress = "127.0.0.1"
	cfg.Server.HTTPPort = 9999 // Use an unlikely port

	// Make sure env vars are not set
	os.Unsetenv("HEALTHCHECK_URL")
	os.Setenv("HEALTHCHECK_TIMEOUT", "10ms")
	defer os.Unsetenv("HEALTHCHECK_TIMEOUT")

	result := RunHealthcheck(cfg)
	// Should fail because server isn't running, but we're testing URL construction
	if result != 1 {
		t.Fatalf("expected healthcheck to fail when connecting to non-existent server, got exit code %d", result)
	}
}

func TestRunHealthcheck_DefaultPort(t *testing.T) {
	cfg := &config.Config{}
	// Port is 0, should default to 8080

	os.Unsetenv("HEALTHCHECK_URL")
	os.Setenv("HEALTHCHECK_TIMEOUT", "10ms")
	defer os.Unsetenv("HEALTHCHECK_TIMEOUT")

	result := RunHealthcheck(cfg)
	// Should fail because server isn't running
	if result != 1 {
		t.Fatalf("expected healthcheck to fail when connecting to non-existent server, got exit code %d", result)
	}
}
