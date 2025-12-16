// Package httpserver exposes the HTTP endpoints for telemetry collection.
package httpserver

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
	"github.com/threatfabric-devops/tf-telemetry/internal/middleware"
)

// TelemetryService defines the subset of the collector used by HTTP endpoints.
type TelemetryService interface {
	SendTelemetry(context.Context, *pb.TelemetryPacket) (*pb.Ack, error)
}

// New returns a configured HTTP server for telemetry ingestion.
func New(cfg *config.Config, svc TelemetryService, limiter *middleware.RateLimiter) *http.Server {
	handler := telemetryMux(svc, limiter)
	// Add CORS middleware for web clients (configurable)
	handler = middleware.CorsMiddleware(handler, cfg.Server.CORS)

	if cfg.Server.BasicAuth.Enabled {
		handler = middleware.BasicAuthHTTP(cfg.Server.BasicAuth)(handler)
	}

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.HTTPPort),
		Handler: handler,
	}
}

func telemetryMux(svc TelemetryService, limiter *middleware.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/telemetry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var packet pb.TelemetryPacket
		if err := decodeRequestBody(r, &packet); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}
		if limiter != nil && !limiter.Allow(installationIDHex(&packet)) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		if _, err := svc.SendTelemetry(r.Context(), &packet); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return mux
}

func installationIDHex(packet *pb.TelemetryPacket) string {
	if packet == nil || packet.GetMetadata() == nil {
		return ""
	}
	return hex.EncodeToString(packet.GetMetadata().GetInstallationId())
}

func decodeRequestBody(r *http.Request, message proto.Message) error {
	body, readErr := io.ReadAll(r.Body)
	closeErr := r.Body.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, message)
}
