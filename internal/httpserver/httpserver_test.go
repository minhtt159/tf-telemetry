package httpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/minhtt159/tf-telemetry/internal/config"
	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
	"github.com/minhtt159/tf-telemetry/internal/middleware"
)

type stubTelemetryService struct {
	calls int
}

func (s *stubTelemetryService) SendTelemetry(_ context.Context, _ *pb.TelemetryPacket) (*pb.Ack, error) {
	s.calls++
	return &pb.Ack{Success: true}, nil
}

func baseConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.HTTPPort = 8080
	cfg.Server.BindAddress = "127.0.0.1"
	return cfg
}

func TestHTTPServerBasicAuth(t *testing.T) {
	cfg := baseConfig()
	cfg.Server.BasicAuth.Enabled = true
	cfg.Server.BasicAuth.Username = "user"
	cfg.Server.BasicAuth.Password = "pass"

	server := New(cfg, &stubTelemetryService{}, middleware.NewRateLimiter(config.RateLimitConfig{}))

	body, err := protojson.Marshal(&pb.TelemetryPacket{Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID}})
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	req.SetBasicAuth("user", "pass")
	rr = httptest.NewRecorder()
	server.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with auth, got %d", rr.Code)
	}
}

func TestHTTPServerRateLimitByInstallationID(t *testing.T) {
	cfg := baseConfig()
	rl := middleware.NewRateLimiter(config.RateLimitConfig{Enabled: true, RequestsPerSecond: 1, Burst: 1})
	server := New(cfg, &stubTelemetryService{}, rl)

	body, err := protojson.Marshal(&pb.TelemetryPacket{Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID, InstallationId: []byte{0x01}}})
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	handler := server.Handler
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected first request accepted, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limited response, got %d", rr2.Code)
	}
}
