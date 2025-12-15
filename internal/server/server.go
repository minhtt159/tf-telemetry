package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/elastic/go-elasticsearch/v8/esutil"
	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Service struct {
	pb.UnimplementedCollectorServer
	logger      *zap.Logger
	bulkIndexer esutil.BulkIndexer
	cfg         *config.Config
	rateLimit   rate.Limit
	rateBurst   int
	limiterMu   sync.Mutex
	limiters    map[string]*rate.Limiter
}

func New(logger *zap.Logger, bi esutil.BulkIndexer, cfg *config.Config) *Service {
	svc := &Service{
		logger:      logger,
		bulkIndexer: bi,
		cfg:         cfg,
	}
	if cfg.Server.RateLimit.Enabled {
		svc.rateLimit = rate.Limit(cfg.Server.RateLimit.RequestsPerSecond)
		svc.rateBurst = cfg.Server.RateLimit.Burst
		svc.limiters = make(map[string]*rate.Limiter)
	}
	return svc
}

func (s *Service) limiterForKey(key string) *rate.Limiter {
	if s.rateLimit <= 0 {
		return nil
	}
	if key == "" {
		key = "global"
	}
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()
	if limiter, ok := s.limiters[key]; ok {
		return limiter
	}
	limiter := rate.NewLimiter(s.rateLimit, s.rateBurst)
	s.limiters[key] = limiter
	return limiter
}

func (s *Service) SendTelemetry(ctx context.Context, packet *pb.TelemetryPacket) (*pb.Ack, error) {
	if packet == nil || packet.Metadata == nil {
		return nil, status.Error(codes.InvalidArgument, "missing metadata")
	}

	if packet.Metrics != nil {
		for _, point := range packet.Metrics.Points {
			doc := s.metricDocument(packet.Metadata, point)
			s.indexAsync(ctx, s.cfg.Elastic.IndexMetrics, doc)
		}
	}

	if packet.Logs != nil {
		for _, entry := range packet.Logs.Entries {
			doc := s.logDocument(packet.Metadata, entry)
			s.indexAsync(ctx, s.cfg.Elastic.IndexLogs, doc)
		}
	}

	return &pb.Ack{Success: true, Message: "Accepted"}, nil
}

func (s *Service) metricDocument(metadata *pb.ClientMetadata, point *pb.MetricPoint) map[string]any {
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

func (s *Service) logDocument(metadata *pb.ClientMetadata, entry *pb.LogEntry) map[string]any {
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

func (s *Service) indexAsync(ctx context.Context, index string, doc map[string]any) {
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

func decodeRequestBody(r *http.Request, message proto.Message) error {
	defer func() {
		if err := r.Body.Close(); err != nil {
			panic(err)
		}
	}()
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
			http.Error(w, "Invalid request format", http.StatusBadRequest)
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
		if !ok || !credentialsMatch(username, password, cfg) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Service) rateLimitMiddleware(next http.Handler) http.Handler {
	if s.rateLimit <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := s.limiterForKey(clientKeyFromRequest(r))
		if limiter != nil && !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins for demo purposes. In production, configure specific allowed origins.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (s *Service) HTTPServer(cfg *config.Config) *http.Server {
	handler := s.httpHandler()

	handler = s.rateLimitMiddleware(handler)

	if cfg.Server.BasicAuth.Enabled {
		handler = basicAuthMiddleware(handler, cfg.Server.BasicAuth)
	}

	// Add CORS middleware for web clients
	handler = corsMiddleware(handler)

	return &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.HTTPPort),
		Handler: handler,
	}
}

// gRPC
func (s *Service) grpcInterceptors(cfg *config.Config) []grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	if cfg.Server.BasicAuth.Enabled {
		interceptors = append(interceptors, s.basicAuthInterceptor(cfg.Server.BasicAuth))
	}

	if rl := s.rateLimitInterceptor(); rl != nil {
		interceptors = append(interceptors, rl)
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
	addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	grpcServer := s.GRPCServer(cfg)
	s.logger.Info("gRPC server listening", zap.String("addr", addr))
	return grpcServer, lis, nil
}

func (s *Service) basicAuthInterceptor(cfg config.BasicAuthConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
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
	if !credentialsMatch(parts[0], parts[1], cfg) {
		return errors.New("invalid credentials")
	}
	return nil
}

func credentialsMatch(username, password string, cfg config.BasicAuthConfig) bool {
	return subtle.ConstantTimeCompare([]byte(username), []byte(cfg.Username)) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(cfg.Password)) == 1
}

func clientKeyFromRequest(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func clientKeyFromContext(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		host, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			return p.Addr.String()
		}
		return host
	}
	return ""
}

func (s *Service) rateLimitInterceptor() grpc.UnaryServerInterceptor {
	if s.rateLimit <= 0 {
		return nil
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		limiter := s.limiterForKey(clientKeyFromContext(ctx))
		if limiter != nil && !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}
