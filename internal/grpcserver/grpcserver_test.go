package grpcserver

import (
	"context"
	"encoding/base64"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/minhtt159/tf-telemetry/internal/config"
	"github.com/minhtt159/tf-telemetry/internal/gen/pb"
	"github.com/minhtt159/tf-telemetry/internal/middleware"
)

const bufSize = 1024 * 1024

type authCollector struct {
	pb.UnimplementedCollectorServer
}

func (a *authCollector) SendTelemetry(ctx context.Context, _ *pb.TelemetryPacket) (*pb.Ack, error) {
	return &pb.Ack{Success: true}, nil
}

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.BasicAuth.Username = "user"
	cfg.Server.BasicAuth.Password = "pass"
	return cfg
}

func dialBufConn(t *testing.T, srv *grpc.Server, lis *bufconn.Listener) pb.CollectorClient {
	t.Helper()
	go func() {
		if err := srv.Serve(lis); err != nil {
			t.Logf("serve error: %v", err)
		}
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("failed to dial bufconn: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("failed to close connection: %v", err)
		}
		srv.Stop()
	})
	return pb.NewCollectorClient(conn)
}

func TestNew_RegistersCollector(t *testing.T) {
	cfg := &config.Config{}
	srv := New(cfg, &authCollector{}, nil)
	info := srv.GetServiceInfo()
	if len(info) == 0 {
		t.Fatalf("collector service not registered")
	}
}

func TestNew_BasicAuthInterceptor(t *testing.T) {
	cfg := testConfig()
	cfg.Server.BasicAuth.Enabled = true
	srv := New(cfg, &authCollector{}, nil)
	lis := bufconn.Listen(bufSize)
	client := dialBufConn(t, srv, lis)

	ctx := context.Background()
	_, err := client.SendTelemetry(ctx, &pb.TelemetryPacket{Metadata: &pb.ClientMetadata{}})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected unauthenticated without credentials, got %v", status.Code(err))
	}

	auth := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Basic "+auth))
	if _, err := client.SendTelemetry(ctx, &pb.TelemetryPacket{Metadata: &pb.ClientMetadata{}}); err != nil {
		t.Fatalf("expected success with auth, got %v", err)
	}
}

func TestNew_RateLimitInterceptor(t *testing.T) {
	cfg := &config.Config{}
	limiter := middleware.NewRateLimiter(config.RateLimitConfig{Enabled: true, RequestsPerSecond: 1, Burst: 1})
	srv := New(cfg, &authCollector{}, limiter)
	lis := bufconn.Listen(bufSize)
	client := dialBufConn(t, srv, lis)

	packet := &pb.TelemetryPacket{Metadata: &pb.ClientMetadata{InstallationId: []byte{0x01}}}
	ctx := context.Background()
	if _, err := client.SendTelemetry(ctx, packet); err != nil {
		t.Fatalf("first request should pass, got %v", err)
	}
	if _, err := client.SendTelemetry(ctx, packet); status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("expected rate limit error, got %v", err)
	}
}
