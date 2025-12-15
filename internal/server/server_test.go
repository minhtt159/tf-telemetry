package server

import (
	"context"
	"testing"

	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
)

type stubSender struct {
	lastPacket *pb.TelemetryPacket
}

func (s *stubSender) SendTelemetry(_ context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	s.lastPacket = packet
	return &pb.Ack{Success: true}, nil
}

func TestServiceDelegatesToSender(t *testing.T) {
	sender := &stubSender{}
	svc := New(sender)

	packet := &pb.TelemetryPacket{Metadata: &pb.ClientMetadata{InstallationId: []byte{0x01}}}
	ack, err := svc.SendTelemetry(context.Background(), packet)
	if err != nil {
		t.Fatalf("SendTelemetry() error = %v", err)
	}
	if !ack.GetSuccess() {
		t.Fatalf("expected success ack, got %+v", ack)
	}
	if sender.lastPacket != packet {
		t.Fatalf("expected packet to be forwarded to sender")
	}
}
