package main

import (
	"context"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
	"github.com/threatfabric-devops/tf-telemetry/internal/server"
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

func TestMainCanSendTelemetry(t *testing.T) {
	indexer := &recordingBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs-index"
	cfg.Elastic.IndexMetrics = "metrics-index"

	svc := server.New(zap.NewNop(), indexer, cfg)
	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID},
		Metrics:  &pb.MetricBatch{Points: []*pb.MetricPoint{{}}},
		Logs:     &pb.LogBatch{Entries: []*pb.LogEntry{{Message: "ok"}}},
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
