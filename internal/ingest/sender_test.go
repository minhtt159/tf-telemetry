package ingest

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esutil"
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

func TestSendTelemetryIndexesMetricsAndLogs(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := &config.Config{}
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	sender := NewSender(zap.NewNop(), bi, cfg)

	packet := &pb.TelemetryPacket{
		Metadata: &pb.ClientMetadata{
			Platform:         pb.Platform_ANDROID,
			InstallationId:   []byte{0x01},
			JourneyId:        []byte{0x02},
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
	if doc["installation_id"] != "01" {
		t.Fatalf("unexpected installation_id: %v", doc["installation_id"])
	}
}
