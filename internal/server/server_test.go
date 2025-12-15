package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

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

func newTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.HttpPort = 8080
	cfg.Server.GrpcPort = 50051
	cfg.Elastic.IndexLogs = "logs"
	cfg.Elastic.IndexMetrics = "metrics"
	return cfg
}

func TestSendTelemetryIndexesMetricsAndLogs(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := newTestConfig()
	svc := New(zap.NewNop(), bi, cfg)

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

	ack, err := svc.SendTelemetry(context.Background(), packet)
	if err != nil {
		t.Fatalf("SendTelemetry() error = %v", err)
	}
	if ack == nil || !ack.Success {
		t.Fatalf("unexpected ack: %+v", ack)
	}

	if len(bi.items) != 2 {
		t.Fatalf("expected 2 indexed documents, got %d", len(bi.items))
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(bi.items[0].body, &doc); err != nil {
		t.Fatalf("failed to unmarshal metric doc: %v", err)
	}
	if doc["installation_id"] != "01" {
		t.Fatalf("unexpected installation_id: %v", doc["installation_id"])
	}
}

func TestHTTPServerBasicAuth(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := newTestConfig()
	cfg.Server.BasicAuth.Enabled = true
	cfg.Server.BasicAuth.Username = "user"
	cfg.Server.BasicAuth.Password = "pass"
	svc := New(zap.NewNop(), bi, cfg)

	body, _ := protojson.Marshal(&pb.TelemetryPacket{Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID}})

	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	svc.HTTPServer(cfg).Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	req.Header.Set("Authorization", "Basic "+auth)
	rr = httptest.NewRecorder()
	svc.HTTPServer(cfg).Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with auth, got %d", rr.Code)
	}
}

func TestHTTPServerRateLimit(t *testing.T) {
	bi := &stubBulkIndexer{}
	cfg := newTestConfig()
	cfg.Server.RateLimit.Enabled = true
	cfg.Server.RateLimit.RequestsPerSecond = 1
	cfg.Server.RateLimit.Burst = 1
	svc := New(zap.NewNop(), bi, cfg)

	body, _ := protojson.Marshal(&pb.TelemetryPacket{Metadata: &pb.ClientMetadata{Platform: pb.Platform_ANDROID}})

	handler := svc.HTTPServer(cfg).Handler
	req := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected first request accepted, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/telemetry", bytes.NewReader(body))
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected rate limited response, got %d", rr2.Code)
	}
}
