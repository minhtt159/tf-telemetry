package ingest

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
)

type capturedItem struct {
	index string
	body  []byte
}

type stubBulkIndexer struct {
	items []capturedItem
}

func (s *stubBulkIndexer) Add(_ context.Context, item esutil.BulkIndexerItem) error {
	data, _ := io.ReadAll(item.Body)
	s.items = append(s.items, capturedItem{index: item.Index, body: data})
	return nil
}

func (s *stubBulkIndexer) Close(context.Context) error { return nil }

func (s *stubBulkIndexer) Stats() esutil.BulkIndexerStats { return esutil.BulkIndexerStats{} }

func makeUUIDv7() []byte {
	// Create a valid UUID v7
	u := uuid.Must(uuid.NewV7())
	b, _ := u.MarshalBinary()
	return b
}

func TestSendTelemetryIndexesMetricsAndLogs(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	cfg.Server.MaxContextAttrs = 6
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   makeUUIDv7(),
			JourneyId:        makeUUIDv7(),
			SdkVersionPacked: 3,
			HostAppVersion:   "1.0.0",
			HostAppName:      "app",
		},
		Metrics: &pb.MetricBatch{
			Points: []*pb.MetricPoint{
				{ClientTimestampMs: 10, NetworkType: pb.NetworkType_NET_WIFI, BatteryLevelPercent: 0.5},
			},
		},
		Logs: &pb.LogBatch{
			Entries: []*pb.LogEntry{
				{ClientTimestampMs: 20, NetworkType: pb.NetworkType_NET_WIFI, Level: pb.LogLevel_INFO, Tag: "t", Message: "m"},
			},
		},
	}

	ack, err := sender.SendTelemetry(context.Background(), packet)
	if err != nil {
		t.Fatalf("SendTelemetry() error = %v", err)
	}
	if ack == nil || !ack.Success {
		t.Fatalf("unexpected ack: %+v", ack)
	}

	if len(bi.items) != 2 {
		t.Fatalf("expected 2 indexed documents, got %d", len(bi.items))
	}

	var doc map[string]any
	if err := json.Unmarshal(bi.items[0].body, &doc); err != nil {
		t.Fatalf("failed to unmarshal metric doc: %v", err)
	}
	if doc["installation_id"] == "" {
		t.Fatalf("expected installation_id to be set")
	}
}

func TestSendTelemetry_InvalidInstallationID(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   []byte{0x01, 0x02}, // Invalid: too short
			JourneyId:        makeUUIDv7(),
			SdkVersionPacked: 3,
		},
	}

	_, err := sender.SendTelemetry(context.Background(), packet)
	if err == nil {
		t.Fatal("expected error for invalid installation_id")
	}
	if !strings.Contains(err.Error(), "installation_id") {
		t.Fatalf("expected error about installation_id, got: %v", err)
	}
}

func TestSendTelemetry_InvalidJourneyID(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   makeUUIDv7(),
			JourneyId:        []byte{0x01, 0x02}, // Invalid: too short
			SdkVersionPacked: 3,
		},
	}

	_, err := sender.SendTelemetry(context.Background(), packet)
	if err == nil {
		t.Fatal("expected error for invalid journey_id")
	}
	if !strings.Contains(err.Error(), "journey_id") {
		t.Fatalf("expected error about journey_id, got: %v", err)
	}
}

func TestSendTelemetry_PacketTooLarge(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	sender := NewSender(zap.NewNop(), bi, cfg)

	// Create a packet with a very large message
	largeMessage := strings.Repeat("a", 2000)
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   makeUUIDv7(),
			JourneyId:        makeUUIDv7(),
			SdkVersionPacked: 3,
		},
		Logs: &pb.LogBatch{
			Entries: []*pb.LogEntry{
				{Message: largeMessage},
			},
		},
	}

	_, err := sender.SendTelemetry(context.Background(), packet)
	if err == nil {
		t.Fatal("expected error for packet too large")
	}
	if !strings.Contains(err.Error(), "packet size") {
		t.Fatalf("expected error about packet size, got: %v", err)
	}
}

func TestSendTelemetry_ContextMapLimit(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	cfg.Server.MaxContextAttrs = 3 // Limit to 3
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   makeUUIDv7(),
			JourneyId:        makeUUIDv7(),
			SdkVersionPacked: 3,
		},
		Logs: &pb.LogBatch{
			Entries: []*pb.LogEntry{
				{
					ClientTimestampMs: 20,
					Level:             pb.LogLevel_INFO,
					Tag:               "test",
					Message:           "test message",
					Context: map[string]string{
						"key1": "val1",
						"key2": "val2",
						"key3": "val3",
						"key4": "val4", // Exceeds limit of 3
						"key5": "val5",
					},
				},
			},
		},
	}

	// Should now reject the packet with too many context attributes
	_, err := sender.SendTelemetry(context.Background(), packet)
	if err == nil {
		t.Fatal("expected error for too many context attributes")
	}
	if !strings.Contains(err.Error(), "context") {
		t.Fatalf("expected error about context, got: %v", err)
	}
}

func TestSendTelemetry_ContextMapWithinLimit(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	cfg.Server.MaxContextAttrs = 3 // Limit to 3
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   makeUUIDv7(),
			JourneyId:        makeUUIDv7(),
			SdkVersionPacked: 3,
		},
		Logs: &pb.LogBatch{
			Entries: []*pb.LogEntry{
				{
					ClientTimestampMs: 20,
					Level:             pb.LogLevel_INFO,
					Tag:               "test",
					Message:           "test message",
					Context: map[string]string{
						"key1": "val1",
						"key2": "val2",
						"key3": "val3",
					},
				},
			},
		},
	}

	ack, err := sender.SendTelemetry(context.Background(), packet)
	if err != nil {
		t.Fatalf("SendTelemetry() error = %v", err)
	}
	if !ack.Success {
		t.Fatal("expected success")
	}

	if len(bi.items) != 1 {
		t.Fatalf("expected 1 indexed document, got %d", len(bi.items))
	}

	var doc map[string]any
	if err := json.Unmarshal(bi.items[0].body, &doc); err != nil {
		t.Fatalf("failed to unmarshal log doc: %v", err)
	}

	contextMap, ok := doc["context"].(map[string]any)
	if !ok {
		t.Fatal("expected context to be a map")
	}

	if len(contextMap) != 3 {
		t.Fatalf("expected context map to have 3 attributes, got %d", len(contextMap))
	}
}

func TestValidateUUIDv7_Valid(t *testing.T) {
	validUUID := makeUUIDv7()
	if err := validateUUIDv7(validUUID, "test_id"); err != nil {
		t.Fatalf("expected valid UUID v7, got error: %v", err)
	}
}

func TestValidateUUIDv7_Empty(t *testing.T) {
	if err := validateUUIDv7([]byte{}, "test_id"); err == nil {
		t.Fatal("expected error for empty UUID")
	}
}

func TestValidateUUIDv7_WrongLength(t *testing.T) {
	if err := validateUUIDv7([]byte{0x01, 0x02}, "test_id"); err == nil {
		t.Fatal("expected error for wrong length")
	}
}

func TestValidateUUIDv7_WrongVersion(t *testing.T) {
	// Create a UUID v4 (wrong version)
	u := uuid.New()
	b, _ := u.MarshalBinary()
	
	err := validateUUIDv7(b, "test_id")
	if err == nil {
		t.Fatal("expected error for wrong UUID version")
	}
	if !strings.Contains(err.Error(), "version") {
		t.Fatalf("expected error about version, got: %v", err)
	}
}

func TestValidatePacketSize_Valid(t *testing.T) {
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform: pb.Platform_ANDROID,
		},
	}
	
	if err := validatePacketSize(packet, 1500); err != nil {
		t.Fatalf("expected valid packet size, got error: %v", err)
	}
}

func TestValidatePacketSize_TooLarge(t *testing.T) {
	largeMessage := strings.Repeat("a", 2000)
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform: pb.Platform_ANDROID,
		},
		Logs: &pb.LogBatch{
			Entries: []*pb.LogEntry{
				{Message: largeMessage},
			},
		},
	}
	
	err := validatePacketSize(packet, 1500)
	if err == nil {
		t.Fatal("expected error for packet too large")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("expected error about exceeds maximum, got: %v", err)
	}
}
