// Package httpserver exposes the HTTP endpoints for telemetry collection.
package httpserver

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/minhtt159/tf-telemetry/internal/config"
	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
	"github.com/minhtt159/tf-telemetry/internal/middleware"
)

// TelemetryService defines the subset of the collector used by HTTP endpoints.
type TelemetryService interface {
	SendTelemetry(context.Context, *pb.TelemetryPacket) (*pb.Ack, error)
}

// TelemetryResponse represents the response for telemetry endpoint
type TelemetryResponse struct {
	Status string `json:"status" example:"accepted"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid request format"`
}

// Handler wraps the telemetry service and rate limiter for HTTP handlers
type Handler struct {
	svc     TelemetryService
	limiter *middleware.RateLimiter
}

// New returns a configured HTTP server for telemetry ingestion.
func New(cfg *config.Config, svc TelemetryService, limiter *middleware.RateLimiter) *http.Server {
	h := &Handler{svc: svc, limiter: limiter}
	handler := h.setupRoutes()

	if cfg.Server.BasicAuth.Enabled {
		handler = middleware.BasicAuthHTTP(cfg.Server.BasicAuth)(handler)
	}

	handler = middleware.CorsMiddleware(handler, cfg.Server.CORS)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.HTTPPort),
		Handler: handler,
	}
}

func (h *Handler) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Swagger UI handler
	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	mux.HandleFunc("/v1/telemetry", h.SendTelemetry)
	mux.HandleFunc("/healthz", h.Healthz)

	return mux
}

// SendTelemetry handles POST /v1/telemetry
//
//	@Summary		Send telemetry data
//	@Description	Submit metrics and/or logs telemetry data
//	@Tags			telemetry
//	@Accept			json
//	@Produce		json
//	@Param			body	body		object	true	"Telemetry packet with metadata, metrics, and logs"
//	@Success		202		{object}	TelemetryResponse
//	@Failure		400		{object}	ErrorResponse
//	@Failure		401		{object}	ErrorResponse
//	@Failure		429		{object}	ErrorResponse
//	@Security		BasicAuth
//	@Router			/v1/telemetry [post]
func (h *Handler) SendTelemetry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var packet pb.TelemetryPacket
	if err := decodeRequestBody(r, &packet); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}
	if h.limiter != nil && !h.limiter.Allow(installationIDHex(&packet)) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}
	if _, err := h.svc.SendTelemetry(r.Context(), &packet); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"accepted"}`))
}

// Healthz handles GET /healthz
//
//	@Summary		Health check
//	@Description	Returns health status of the server
//	@Tags			health
//	@Produce		plain
//	@Success		200	{string}	string	"ok"
//	@Router			/healthz [get]
func (h *Handler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
