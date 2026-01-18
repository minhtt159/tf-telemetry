// Package service implements the telemetry collector gRPC service.
package service

import (
	"context"

	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
)

// TelemetrySender processes telemetry packets.
type TelemetrySender interface {
	SendTelemetry(context.Context, *pb.TelemetryPacket) (*pb.Ack, error)
}

// Service implements the CollectorServer and delegates to the telemetry sender.
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

// SendTelemetry delegates incoming telemetry data to the sender.
func (s *Service) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	return s.sender.SendTelemetry(ctx, packet)
}
