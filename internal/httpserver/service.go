package httpserver

import (
	"context"

	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
)

// Service implements the collector service and delegates to the telemetry sender.
type Service struct {
	pb.UnimplementedCollectorServer
	sender TelemetryService
}

// NewService creates a telemetry Service.
func NewService(sender TelemetryService) *Service {
	return &Service{
		sender: sender,
	}
}

// SendTelemetry indexes incoming telemetry data.
func (s *Service) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	return s.sender.SendTelemetry(ctx, packet)
}
