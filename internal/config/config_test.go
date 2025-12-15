package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadWithAuthAndRateLimit(t *testing.T) {
	path := writeConfig(t, `
server:
  grpc_port: 6000
  http_port: 7000
  basic_auth:
    enabled: true
    username: "user"
    password: "pass"
  rate_limit:
    enabled: true
    requests_per_second: 5
    burst: 8
elasticsearch:
  addresses: ["http://localhost:9200"]
  username: "elastic"
  password: "changeme"
  index_metrics: "metrics"
  index_logs: "logs"
  batch_size: 1000
  flush_interval_seconds: 3
logging:
  level: "debug"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if !cfg.Server.BasicAuth.Enabled || cfg.Server.BasicAuth.Username != "user" || cfg.Server.BasicAuth.Password != "pass" {
		t.Fatalf("basic auth not parsed: %+v", cfg.Server.BasicAuth)
	}

	if !cfg.Server.RateLimit.Enabled || cfg.Server.RateLimit.RequestsPerSecond != 5 || cfg.Server.RateLimit.Burst != 8 {
		t.Fatalf("rate limit not parsed: %+v", cfg.Server.RateLimit)
	}
}

func TestLoadRateLimitDefaultBurst(t *testing.T) {
	path := writeConfig(t, `
server:
  grpc_port: 1
  http_port: 2
  rate_limit:
    enabled: true
    requests_per_second: 2
elasticsearch:
  addresses: ["http://localhost:9200"]
  index_metrics: "metrics"
  index_logs: "logs"
  flush_interval_seconds: 3
logging:
  level: "info"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.RateLimit.Burst == 0 {
		t.Fatalf("expected burst to be defaulted, got 0")
	}
}
