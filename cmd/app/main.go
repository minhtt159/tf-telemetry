package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/grpcserver"
	"github.com/threatfabric-devops/tf-telemetry/internal/httpserver"
	"github.com/threatfabric-devops/tf-telemetry/internal/indexer"
	"github.com/threatfabric-devops/tf-telemetry/internal/ingest"
	"github.com/threatfabric-devops/tf-telemetry/internal/logger"
	"github.com/threatfabric-devops/tf-telemetry/internal/middleware"
	"github.com/threatfabric-devops/tf-telemetry/internal/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck())
	}

	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.Logging.Level)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	_, bi, err := indexer.New(cfg, log)
	if err != nil {
		log.Fatal("failed to create indexer", zap.Error(err))
	}

	sender := ingest.NewSender(log, bi, cfg)
	svc := server.New(sender)
	limiter := middleware.NewRateLimiter(cfg.Server.RateLimit)

	grpcServer := grpcserver.New(cfg, svc, limiter)
	lis, addr, err := grpcserver.Listen(cfg)
	if err != nil {
		log.Fatal("failed to start gRPC server", zap.Error(err))
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Error("Failed to serve gRPC", zap.Error(err))
		}
	}()

	log.Info("gRPC server listening", zap.String("addr", addr))

	httpServer := httpserver.New(cfg, svc, limiter)
	go func() {
		log.Info("HTTP server listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to serve HTTP", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := bi.Close(ctx); err != nil {
		log.Error("Error closing bulk indexer", zap.Error(err))
	}

	httpServer.Shutdown(ctx)
	grpcServer.GracefulStop()
	if lis != nil {
		lis.Close()
	}
	log.Info("Shutdown complete")
}

func runHealthcheck() int {
	url := os.Getenv("HEALTHCHECK_URL")
	if url == "" {
		if cfg, err := config.Load("config.yaml"); err == nil {
			host := cfg.Server.BindAddress
			if host == "" {
				host = "127.0.0.1"
			}
			port := cfg.Server.HTTPPort
			if port == 0 {
				port = 8080
			}
			url = fmt.Sprintf("http://%s:%d/healthz", host, port)
		} else {
			url = "http://127.0.0.1:8080/healthz"
		}
	}
	timeout := 2 * time.Second
	if v := os.Getenv("HEALTHCHECK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			timeout = d
		}
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
