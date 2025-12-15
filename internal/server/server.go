package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Service struct {
	pb.UnimplementedCollectorServer
	logger      *zap.Logger
	bulkIndexer esutil.BulkIndexer
	cfg         *config.Config
}

func New(logger *zap.Logger, bi esutil.BulkIndexer, cfg *config.Config) *Service {
	return &Service{
		logger:      logger,
		bulkIndexer: bi,
		cfg:         cfg,
	}
}

func (s *Service) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	if packet == nil || packet.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "missing metadata")
	}

	if packet.Metrics != nil {
		for _, point := range packet.Metrics.Points {
			doc := s.metricDocument(packet.Metadata, point)
			s.indexAsync(s.cfg.Elastic.IndexMetrics, doc)
		}
	}

	if packet.Logs != nil {
		for _, entry := range packet.Logs.Entries {
			doc := s.logDocument(packet.Metadata, entry)
			s.indexAsync(s.cfg.Elastic.IndexLogs, doc)
		}
	}

	return &pb.Ack{Success: true, Message: "Accepted"}, nil
}

func (s *Service) metricDocument(metadata *pb.ClientMetadata, point *pb.MetricPoint) map[string]interface{} {
	doc := map[string]interface{}{
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
		doc["device_hardware"] = map[string]interface{}{
			"physical_cores":       hardware.GetPhysicalCores(),
			"logical_cpus":         hardware.GetLogicalCpus(),
			"l1_cache_kb":          hardware.GetL1CacheKb(),
			"l2_cache_kb":          hardware.GetL2CacheKb(),
			"l3_cache_kb":          hardware.GetL3CacheKb(),
			"total_physical_bytes": hardware.GetTotalPhysicalMemory(),
		}
	}

	if cpu := point.GetCpu(); cpu != nil {
		doc["cpu"] = map[string]interface{}{
			"total_usage_percent": cpu.GetTotalUsagePercent(),
			"core_usage_percent":  cpu.GetCoreUsagePercent(),
		}
	}

	if memory := point.GetMemory(); memory != nil {
		doc["memory"] = map[string]interface{}{
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

func (s *Service) logDocument(metadata *pb.ClientMetadata, entry *pb.LogEntry) map[string]interface{} {
	return map[string]interface{}{
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

func (s *Service) indexAsync(index string, doc map[string]interface{}) {
	data, err := json.Marshal(doc)
	if err != nil {
		s.logger.Error("Failed to marshal document", zap.Error(err))
		return
	}

	err = s.bulkIndexer.Add(context.Background(), esutil.BulkIndexerItem{
		Action: "index",
		Index:  index,
		Body:   bytes.NewReader(data),
		OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
			if err != nil {
				s.logger.Error("Failed to index", zap.Error(err))
				return
			}
			if res.Error.Type != "" || res.Error.Reason != "" {
				s.logger.Error("Failed to index", zap.String("err", res.Error.Reason))
				return
			}
			s.logger.Error("Failed to index")
		},
	})
	if err != nil {
		s.logger.Error("Failed to add to indexer", zap.Error(err))
	}
}

func decodeRequestBody(r *http.Request, message proto.Message) error {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(body, message)
}

// HTTP
func (s *Service) httpHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/telemetry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var packet pb.TelemetryPacket
		if err := decodeRequestBody(r, &packet); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if _, err := s.SendTelemetry(r.Context(), &packet); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"accepted"}`))
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return mux
}

func basicAuthMiddleware(next http.Handler, cfg config.BasicAuthConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok || username != cfg.Username || password != cfg.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimitMiddleware(next http.Handler, limiter *rate.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) HTTPServer(cfg *config.Config) *http.Server {
	handler := s.httpHandler()

	if cfg.Server.RateLimit.Enabled {
		limiter := rate.NewLimiter(rate.Limit(cfg.Server.RateLimit.RequestsPerSecond), cfg.Server.RateLimit.Burst)
		handler = rateLimitMiddleware(handler, limiter)
	}

	if cfg.Server.BasicAuth.Enabled {
		handler = basicAuthMiddleware(handler, cfg.Server.BasicAuth)
	}

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HttpPort),
		Handler: handler,
	}
}

// gRPC
func (s *Service) grpcInterceptors(cfg *config.Config) []grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	if cfg.Server.BasicAuth.Enabled {
		interceptors = append(interceptors, s.basicAuthInterceptor(cfg.Server.BasicAuth))
	}

	if cfg.Server.RateLimit.Enabled {
		limiter := rate.NewLimiter(rate.Limit(cfg.Server.RateLimit.RequestsPerSecond), cfg.Server.RateLimit.Burst)
		interceptors = append(interceptors, rateLimitInterceptor(limiter))
	}

	return interceptors
}

func (s *Service) GRPCServer(cfg *config.Config) *grpc.Server {
	interceptors := s.grpcInterceptors(cfg)
	opts := []grpc.ServerOption{}
	if len(interceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(interceptors...))
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCollectorServer(grpcServer, s)
	return grpcServer
}

func (s *Service) StartGRPC(cfg *config.Config) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.GrpcPort))
	if err != nil {
		return nil, nil, err
	}
	grpcServer := s.GRPCServer(cfg)
	s.logger.Info("gRPC server listening", zap.Int("port", cfg.Server.GrpcPort))
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			s.logger.Error("Failed to serve gRPC", zap.Error(err))
		}
	}()
	return grpcServer, lis, nil
}

func (s *Service) basicAuthInterceptor(cfg config.BasicAuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}
		if err := validateBasicAuth(md, cfg); err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
		return handler(ctx, req)
	}
}

func validateBasicAuth(md metadata.MD, cfg config.BasicAuthConfig) error {
	authHeaders := md.Get("authorization")
	if len(authHeaders) == 0 {
		return errors.New("authorization header missing")
	}
	const prefix = "Basic "
	header := authHeaders[0]
	if !strings.HasPrefix(header, prefix) {
		return errors.New("invalid authorization header")
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, prefix))
	if err != nil {
		return errors.New("invalid base64 in authorization header")
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return errors.New("invalid authorization value")
	}
	if parts[0] != cfg.Username || parts[1] != cfg.Password {
		return errors.New("invalid credentials")
	}
	return nil
}

func rateLimitInterceptor(limiter *rate.Limiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
