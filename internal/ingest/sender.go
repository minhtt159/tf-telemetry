// Package ingest handles preparing and sending telemetry documents to Elasticsearch.
package ingest

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
)

// Sender converts telemetry packets into documents and forwards them to a bulk indexer.
type Sender struct {
	logger      *zap.Logger
	bulkIndexer esutil.BulkIndexer
	cfg         *config.Config
}

// NewSender returns a Sender instance.
func NewSender(logger *zap.Logger, bi esutil.BulkIndexer, cfg *config.Config) *Sender {
	return &Sender{
		logger:      logger,
		bulkIndexer: bi,
		cfg:         cfg,
	}
}

// SendTelemetry indexes incoming telemetry data and returns an acknowledgement.
func (s *Sender) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	if packet == nil || packet.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "missing metadata")
	}

	// Validate packet size - use configured value or default to 1500
	maxPacketSize := s.cfg.Server.MaxPacketSizeBytes
	if maxPacketSize == 0 {
		maxPacketSize = 1500 // default: 1 MTU
	}
	if err := validatePacketSize(packet, maxPacketSize); err != nil {
		s.logger.Warn("packet size exceeded", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate UUIDs
	if err := validateUUIDv7(packet.Metadata.GetInstallationId(), "installation_id"); err != nil {
		s.logger.Warn("invalid installation_id", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validateUUIDv7(packet.Metadata.GetJourneyId(), "journey_id"); err != nil {
		s.logger.Warn("invalid journey_id", zap.Error(err))
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if packet.Metrics != nil {
		for _, point := range packet.Metrics.Points {
			doc := metricDocument(packet.Metadata, point)
			s.indexAsync(ctx, s.cfg.Elastic.IndexMetrics, doc)
		}
	}

	if packet.Logs != nil {
		maxContextAttrs := s.cfg.Server.MaxContextAttrs
		if maxContextAttrs == 0 {
			maxContextAttrs = 6 // default
		}
		
		for _, entry := range packet.Logs.Entries {
			// Validate context attributes before processing
			if len(entry.GetContext()) > maxContextAttrs {
				s.logger.Warn("log entry context exceeds maximum attributes",
					zap.Int("count", len(entry.GetContext())),
					zap.Int("max", maxContextAttrs))
				return nil, status.Errorf(codes.InvalidArgument,
					"log entry context has %d attributes, maximum allowed is %d",
					len(entry.GetContext()), maxContextAttrs)
			}
			
			doc := logDocument(packet.Metadata, entry)
			s.indexAsync(ctx, s.cfg.Elastic.IndexLogs, doc)
		}
	}

	return &pb.Ack{Success: true, Message: "Accepted"}, nil
}

func metricDocument(metadata *pb.ClientMetadata, point *pb.MetricPoint) map[string]any {
	doc := map[string]any{
		"timestamp":        point.GetClientTimestampMs(),
		"platform":         metadata.GetPlatform().String(),
		"installation_id":  hex.EncodeToString(metadata.GetInstallationId()),
		"journey_id":       hex.EncodeToString(metadata.GetJourneyId()),
		"sdk_version":      metadata.GetSdkVersionPacked(),
		"host_app_version": metadata.GetHostAppVersion(),
		"host_app_name":    metadata.GetHostAppName(),
		"network":          point.GetNetworkType().String(),
		"battery_level":    point.GetBatteryLevelPercent(),
	}

	if hardware := metadata.GetDeviceHardware(); hardware != nil {
		doc["device_hardware"] = map[string]any{
			"physical_cores":       hardware.GetPhysicalCores(),
			"logical_cpus":         hardware.GetLogicalCpus(),
			"l1_cache_kb":          hardware.GetL1CacheKb(),
			"l2_cache_kb":          hardware.GetL2CacheKb(),
			"l3_cache_kb":          hardware.GetL3CacheKb(),
			"total_physical_bytes": hardware.GetTotalPhysicalMemory(),
		}
	}

	if cpu := point.GetCpu(); cpu != nil {
		doc["cpu"] = map[string]any{
			"total_usage_percent": cpu.GetTotalUsagePercent(),
			"core_usage_percent":  cpu.GetCoreUsagePercent(),
		}
	}

	if memory := point.GetMemory(); memory != nil {
		doc["memory"] = map[string]any{
			"app_resident_bytes":    memory.GetAppResidentBytes(),
			"app_virtual_bytes":     memory.GetAppVirtualBytes(),
			"system_free_bytes":     memory.GetSystemFreeBytes(),
			"system_active_bytes":   memory.GetSystemActiveBytes(),
			"system_inactive_bytes": memory.GetSystemInactiveBytes(),
			"system_wired_bytes":    memory.GetSystemWiredBytes(),
		}
	}

	return doc
}

func logDocument(metadata *pb.ClientMetadata, entry *pb.LogEntry) map[string]any {
	return map[string]any{
		"timestamp":        entry.GetClientTimestampMs(),
		"platform":         metadata.GetPlatform().String(),
		"installation_id":  hex.EncodeToString(metadata.GetInstallationId()),
		"journey_id":       hex.EncodeToString(metadata.GetJourneyId()),
		"sdk_version":      metadata.GetSdkVersionPacked(),
		"host_app_version": metadata.GetHostAppVersion(),
		"host_app_name":    metadata.GetHostAppName(),
		"network":          entry.GetNetworkType().String(),
		"level":            entry.GetLevel().String(),
		"tag":              entry.GetTag(),
		"message":          entry.GetMessage(),
		"context":          entry.GetContext(),
		"stack_trace":      entry.GetStackTrace(),
	}
}

func (s *Sender) indexAsync(ctx context.Context, index string, doc map[string]any) {
	data, err := json.Marshal(doc)
	if err != nil {
		s.logger.Error("Failed to marshal document", zap.Error(err))
		return
	}

	err = s.bulkIndexer.Add(ctx, esutil.BulkIndexerItem{
		Action: "index",
		Index:  index,
		Body:   bytes.NewReader(data),
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			fields := []zap.Field{
				zap.String("index", item.Index),
				zap.Int("status", res.Status),
				zap.String("result", res.Result),
			}
			if err != nil {
				s.logger.Error("Failed to index", append(fields, zap.Error(err))...)
				return
			}
			if res.Error.Type != "" || res.Error.Reason != "" {
				fields = append(fields, zap.String("err", res.Error.Reason), zap.String("type", res.Error.Type))
				s.logger.Error("Failed to index", fields...)
			}
		},
	})
	if err != nil {
		s.logger.Error("Failed to add to indexer", zap.Error(err))
	}
}

// validateUUIDv7 checks if the given byte slice is a valid UUID v7.
func validateUUIDv7(data []byte, fieldName string) error {
	if len(data) == 0 {
		return fmt.Errorf("%s is required", fieldName)
	}
	
	if len(data) != 16 {
		return fmt.Errorf("%s must be 16 bytes, got %d", fieldName, len(data))
	}
	
	// Parse as UUID
	u, err := uuid.FromBytes(data)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", fieldName, err)
	}
	
	// Check if it's UUID v7 by examining the version bits
	// UUID v7 has version 7 in the most significant 4 bits of the 7th byte (index 6)
	version := (data[6] & 0xf0) >> 4
	if version != 7 {
		return fmt.Errorf("%s must be UUID v7, got version %d (UUID: %s)", fieldName, version, u.String())
	}
	
	// Check variant bits (should be 10xx in the most significant bits of the 9th byte)
	variant := (data[8] & 0xc0) >> 6
	if variant != 2 { // Variant 10xx in binary = 2 in decimal
		return fmt.Errorf("%s has invalid UUID variant", fieldName)
	}
	
	return nil
}

// validatePacketSize checks if the packet size exceeds the maximum allowed size.
func validatePacketSize(packet *pb.TelemetryPacket, maxSize int) error {
	size := proto.Size(packet)
	if size > maxSize {
		return fmt.Errorf("packet size %d bytes exceeds maximum %d bytes", size, maxSize)
	}
	return nil
}
