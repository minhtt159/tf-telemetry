package main

import (
	"context"
	"testing"

	"github.com/elastic/go-elasticsearch/v9/esutil"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/minhtt159/tf-telemetry/internal/config"
	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
	"github.com/minhtt159/tf-telemetry/internal/httpserver"
	"github.com/minhtt159/tf-telemetry/internal/ingest"
)

type recordingBulkIndexer struct {
	recorded int
}

func (m *recordingBulkIndexer) Add(_ context.Context, item esutil.BulkIndexerItem) error {
	m.recorded++
	return nil
}

func (m *recordingBulkIndexer) Close(context.Context) error { return nil }

func (m *recordingBulkIndexer) Stats() esutil.BulkIndexerStats { return esutil.BulkIndexerStats{} }

func makeUUIDv7() []byte {
	u := uuid.Must(uuid.NewV7())
	b, _ := u.MarshalBinary()
	return b
}

func TestMainCanSendTelemetry(t *testing.T) {
	indexer := &recordingBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs-index"
	cfg.Elastic.IndexMetrics = "metrics-index"
	cfg.Server.MaxContextAttrs = 6

	sender := ingest.NewSender(zap.NewNop(), indexer, cfg)
	svc := httpserver.NewService(sender)
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:       pb.Platform_ANDROID,
			InstallationId: makeUUIDv7(),
			JourneyId:      makeUUIDv7(),
		},
		Metrics: &pb.MetricBatch{Points: []*pb.MetricPoint{{}}},
		Logs:    &pb.LogBatch{Entries: []*pb.LogEntry{{Message: "ok"}}},
	}
	if _, err := svc.SendTelemetry(context.Background(), packet); err != nil {
		t.Fatalf("SendTelemetry returned error: %v", err)
	}
	if indexer.recorded != 2 {
		t.Fatalf("expected 2 items indexed, got %d", indexer.recorded)
	}
}

func BenchmarkMarshalTelemetryPacket(b *testing.B) {
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID},
		Metrics:  &pb.MetricBatch{Points: []*pb.MetricPoint{{}}},
	}
	for b.Loop() {
		if _, err := protojson.Marshal(packet); err != nil {
			b.Fatalf("marshal failed: %v", err)
		}
	}
}
