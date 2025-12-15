// Package grpcserver wires the telemetry collector into a gRPC server.
package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/gen/pb"
	"github.com/threatfabric-devops/tf-telemetry/internal/middleware"
)

// New returns a configured gRPC server with the provided collector service.
func New(cfg *config.Config, svc pb.CollectorServer, limiter *middleware.RateLimiter) *grpc.Server {
	var interceptors []grpc.UnaryServerInterceptor

	if cfg.Server.BasicAuth.Enabled {
		interceptors = append(interceptors, middleware.BasicAuthUnary(cfg.Server.BasicAuth))
	}

	if rl := middleware.RateLimitUnary(limiter); rl != nil {
		interceptors = append(interceptors, rl)
	}

	opts := []grpc.ServerOption{}
	if len(interceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(interceptors...))
	}

	server := grpc.NewServer(opts...)
	pb.RegisterCollectorServer(server, svc)
	return server
}

// Listen opens a TCP listener for the configured gRPC port.
func Listen(cfg *config.Config) (net.Listener, string, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.BindAddress, cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	return lis, addr, err
}
