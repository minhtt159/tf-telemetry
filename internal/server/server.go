// Package server implements the telemetry collector gRPC service.
package server

import (
	"context"

	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
)

// TelemetrySender indexes telemetry packets.
type TelemetrySender interface {
	SendTelemetry(context.Context, *pb.TelemetryPacket) (*pb.Ack, error)
}

// Service implements the CollectorServer and delegates indexing to the ingest sender.
type Service struct {
	pb.UnimplementedCollectorServer
	sender TelemetrySender
}

// New creates a telemetry Service.
func New(sender TelemetrySender) *Service {
	return &Service{
		sender: sender,
	}
}

// SendTelemetry indexes incoming telemetry data.
func (s *Service) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	return s.sender.SendTelemetry(ctx, packet)
}
