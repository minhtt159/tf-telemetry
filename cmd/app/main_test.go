package main

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.uber.org/zap"

	pb "github.com/threatfabric-devops/tf-telemetry/internal/observability"
)

type recordedItem struct {
	index string
	doc   map[string]interface{}
}

type recordingBulkIndexer struct {
	mu      sync.Mutex
	records []recordedItem
}

func (m *recordingBulkIndexer) Add(_ context.Context, item esutil.BulkIndexerItem) error {
	body, err := io.ReadAll(item.Body)
	if err != nil {
		return err
	}
	var doc map[string]interface{}
	if err := json.Unmarshal(body, &doc); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, recordedItem{index: item.Index, doc: doc})
	return nil
}

func (m *recordingBulkIndexer) Close(context.Context) error { return nil }

func (m *recordingBulkIndexer) Stats() esutil.BulkIndexerStats { return esutil.BulkIndexerStats{} }

func TestSubmitMetricsIndexesAllPoints(t *testing.T) {
	indexer := &recordingBulkIndexer{}
	cfg := &Config{}
	cfg.Elastic.IndexMetrics = "metrics-index"

	srv := &Server{
		logger:      zap.NewNop(),
		bulkIndexer: indexer,
		cfg:         cfg,
	}

	batch := &pb.MetricBatch{
		SchemaVersion:    1,
		Platform:         pb.Platform_IOS,
		CustomerId:       42,
		SdkVersionPacked: 1001,
		HostAppVersion:   "2.0.0",
		InstallationId:   []byte{0x01, 0x02},
		JourneyId:        []byte{0x0A},
		Metrics: []*pb.MetricPoint{
			{ClientTimestampMs: 1111, NetworkType: pb.NetworkType_NET_WIFI, BatteryLevelPercent: 80.5, CpuUsagePercent: 12.3, MemoryUsageMb: 256},
			{ClientTimestampMs: 2222, NetworkType: pb.NetworkType_NET_CELLULAR_5G, BatteryLevelPercent: 70.0, CpuUsagePercent: 20.0, MemoryUsageMb: 512},
		},
	}

	ack, err := srv.SubmitMetrics(context.Background(), batch)
	if err != nil {
		t.Fatalf("SubmitMetrics returned error: %v", err)
	}
	if !ack.Success {
		t.Fatalf("SubmitMetrics ack should be successful")
	}

	if len(indexer.records) != len(batch.Metrics) {
		t.Fatalf("expected %d indexed metrics, got %d", len(batch.Metrics), len(indexer.records))
	}
	first := indexer.records[0]
	if first.index != cfg.Elastic.IndexMetrics {
		t.Fatalf("expected metric index %q, got %q", cfg.Elastic.IndexMetrics, first.index)
	}
	if first.doc["platform"] != batch.Platform.String() {
		t.Fatalf("expected platform %q, got %v", batch.Platform.String(), first.doc["platform"])
	}
	if first.doc["installation_id"] != "0102" {
		t.Fatalf("expected installation_id hex encoded, got %v", first.doc["installation_id"])
	}
}

func TestSubmitLogsIndexesAllEntries(t *testing.T) {
	indexer := &recordingBulkIndexer{}
	cfg := &Config{}
	cfg.Elastic.IndexLogs = "logs-index"

	srv := &Server{
		logger:      zap.NewNop(),
		bulkIndexer: indexer,
		cfg:         cfg,
	}

	batch := &pb.LogBatch{
		SchemaVersion:    1,
		Platform:         pb.Platform_ANDROID,
		CustomerId:       7,
		SdkVersionPacked: 1002,
		HostAppVersion:   "3.1.4",
		InstallationId:   []byte{0x0B, 0x0C},
		JourneyId:        []byte{0x0D},
		Logs: []*pb.LogEntry{
			{ClientTimestampMs: 3333, Level: pb.LogLevel_INFO, Tag: "tag", Message: "hello", ThreadName: "main"},
			func() *pb.LogEntry {
				trace := "trace"
				return &pb.LogEntry{ClientTimestampMs: 4444, Level: pb.LogLevel_ERROR, Tag: "tag2", Message: "oops", ThreadName: "worker", StackTrace: &trace, ErrorCode: -1}
			}(),
		},
	}

	ack, err := srv.SubmitLogs(context.Background(), batch)
	if err != nil {
		t.Fatalf("SubmitLogs returned error: %v", err)
	}
	if !ack.Success {
		t.Fatalf("SubmitLogs ack should be successful")
	}

	if len(indexer.records) != len(batch.Logs) {
		t.Fatalf("expected %d indexed logs, got %d", len(batch.Logs), len(indexer.records))
	}
	first := indexer.records[0]
	if first.index != cfg.Elastic.IndexLogs {
		t.Fatalf("expected log index %q, got %q", cfg.Elastic.IndexLogs, first.index)
	}
	if first.doc["level"] != batch.Logs[0].Level.String() {
		t.Fatalf("expected level %q, got %v", batch.Logs[0].Level.String(), first.doc["level"])
	}
	if first.doc["journey_id"] != "0d" {
		t.Fatalf("expected journey_id hex encoded, got %v", first.doc["journey_id"])
	}
}
