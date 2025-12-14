package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"

	pb "observability"
)

// --- Config ---
type Config struct {
	Server struct {
		GrpcPort int `mapstructure:"grpc_port"`
		HttpPort int `mapstructure:"http_port"`
	} `mapstructure:"server"`
	Elastic struct {
		Addresses     []string `mapstructure:"addresses"`
		Username      string   `mapstructure:"username"`
		Password      string   `mapstructure:"password"`
		IndexMetrics  string   `mapstructure:"index_metrics"`
		IndexLogs     string   `mapstructure:"index_logs"`
		BatchSize     int      `mapstructure:"batch_size"`
		FlushInterval int      `mapstructure:"flush_interval_seconds"`
	} `mapstructure:"elasticsearch"`
	Logging struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"logging"`
}

// --- Service ---
type Server struct {
	pb.UnimplementedObservabilityServiceServer
	logger      *zap.Logger
	bulkIndexer esutil.BulkIndexer
	cfg         *Config
}

func main() {
	// 1. Config
	viper.SetConfigFile("config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		panic(err)
	}

	// 2. Logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.UnmarshalText([]byte(cfg.Logging.Level))
	logger, _ := loggerConfig.Build()
	defer logger.Sync()

	// 3. Elasticsearch Client
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Elastic.Addresses,
		Username:  cfg.Elastic.Username,
		Password:  cfg.Elastic.Password,
	})
	if err != nil {
		logger.Fatal("Error creating ES client", zap.Error(err))
	}

	// 4. Bulk Indexer (Async Buffer)
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         "", // defined per item
		Client:        es,
		NumWorkers:    2,
		FlushBytes:    1024 * 1024, // 1MB
		FlushInterval: time.Duration(cfg.Elastic.FlushInterval) * time.Second,
	})
	if err != nil {
		logger.Fatal("Error creating bulk indexer", zap.Error(err))
	}

	srv := &Server{
		logger:      logger,
		bulkIndexer: bi,
		cfg:         &cfg,
	}

	// 5. Run Servers
	go runGrpc(srv, cfg.Server.GrpcPort)
	go runHttp(srv, cfg.Server.HttpPort)

	// 6. Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down...")
	if err := bi.Close(context.Background()); err != nil {
		logger.Error("Error closing bulk indexer", zap.Error(err))
	}
	logger.Info("Shutdown complete")
}

// --- gRPC Implementation ---

func (s *Server) SubmitMetrics(ctx context.Context, batch *pb.MetricBatch) (*pb.Ack, error) {
	s.logger.Debug("Received metrics batch", zap.Int("count", len(batch.Metrics)))

	// Flatten batch to individual ES documents
	for _, point := range batch.Metrics {
		doc := map[string]interface{}{
			"timestamp":       point.ClientTimestampMs, // Map to @timestamp in ES
			"schema_version":  batch.SchemaVersion,
			"platform":        batch.Platform.String(),
			"customer_id":     batch.CustomerId,
			"sdk_version":     batch.SdkVersionPacked,
			"app_version":     batch.HostAppVersion,
			"installation_id": hex.EncodeToString(batch.InstallationId), // Bytes -> Hex
			"journey_id":      hex.EncodeToString(batch.JourneyId),      // Bytes -> Hex
			"network":         point.NetworkType.String(),
			"battery_level":   point.BatteryLevelPercent,
			"cpu_usage":       point.CpuUsagePercent,
			"memory_usage_mb": point.MemoryUsageMb,
		}
		s.indexAsync(s.cfg.Elastic.IndexMetrics, doc)
	}

	return &pb.Ack{Success: true, Message: "Accepted"}, nil
}

func (s *Server) SubmitLogs(ctx context.Context, batch *pb.LogBatch) (*pb.Ack, error) {
	s.logger.Debug("Received logs batch", zap.Int("count", len(batch.Logs)))

	for _, log := range batch.Logs {
		doc := map[string]interface{}{
			"timestamp":       log.ClientTimestampMs,
			"platform":        batch.Platform.String(),
			"customer_id":     batch.CustomerId,
			"app_version":     batch.HostAppVersion,
			"installation_id": hex.EncodeToString(batch.InstallationId),
			"journey_id":      hex.EncodeToString(batch.JourneyId),
			"level":           log.Level.String(),
			"tag":             log.Tag,
			"message":         log.Message,
			"thread":          log.ThreadName,
			"stack_trace":     log.StackTrace,
			"error_code":      log.ErrorCode,
			"attributes":      log.Attributes,
		}
		s.indexAsync(s.cfg.Elastic.IndexLogs, doc)
	}

	return &pb.Ack{Success: true, Message: "Accepted"}, nil
}

// --- Helper: Async ES Write ---
func (s *Server) indexAsync(index string, doc interface{}) {
	data, _ := json.Marshal(doc)

	err := s.bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
		Action: "index",
		Index:  index,
		Body:   bytes.NewReader(data),
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			s.logger.Error("Failed to index", zap.String("err", res.Error.Reason))
		},
	})
	if err != nil {
		s.logger.Error("Failed to add to indexer", zap.Error(err))
	}
}

// --- Server Launchers ---

func runGrpc(srv *Server, port int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		srv.logger.Fatal("Failed to listen gRPC", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	pb.RegisterObservabilityServiceServer(grpcServer, srv)
	srv.logger.Info("gRPC server listening", zap.Int("port", port))
	if err := grpcServer.Serve(lis); err != nil {
		srv.logger.Fatal("Failed to serve gRPC", zap.Error(err))
	}
}

func runHttp(srv *Server, port int) {
	mux := http.NewServeMux()

	// Helper to handle JSON -> Proto -> Service logic
	mux.HandleFunc("/v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var batch pb.MetricBatch
		// Use protojson to allow standard JSON mapping to proto fields
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)

		if err := protojson.Unmarshal(body, &batch); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		_, _ = srv.SubmitMetrics(r.Context(), &batch)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	mux.HandleFunc("/v1/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var batch pb.LogBatch
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		if err := protojson.Unmarshal(body, &batch); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		_, _ = srv.SubmitLogs(r.Context(), &batch)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	srv.logger.Info("HTTP server listening", zap.Int("port", port))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		srv.logger.Fatal("Failed to serve HTTP", zap.Error(err))
	}
}
