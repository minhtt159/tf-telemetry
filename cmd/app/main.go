// Package main provides the entry point for tf-telemetry server.
//
//	@title			tf-telemetry API
//	@version		1.0
//	@description	Telemetry collection server for mobile and web applications
//
//	@contact.name Minh Tran
//	@contact.url	https://github.com/minhtt159/tf-telemetry
//
//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT
//
//	@host			localhost:8080
//	@BasePath		/
//
//	@securityDefinitions.basic	BasicAuth
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/minhtt159/tf-telemetry/internal/config"
	"github.com/minhtt159/tf-telemetry/internal/grpcserver"
	"github.com/minhtt159/tf-telemetry/internal/httpserver"
	"github.com/minhtt159/tf-telemetry/internal/indexer"
	"github.com/minhtt159/tf-telemetry/internal/ingest"
	"github.com/minhtt159/tf-telemetry/internal/logger"
	"github.com/minhtt159/tf-telemetry/internal/middleware"
	"github.com/minhtt159/tf-telemetry/internal/server"

	_ "github.com/minhtt159/tf-telemetry/docs/swagger" // swagger docs
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(httpserver.RunHealthcheck(cfg))
	}

	log, err := logger.NewWithConfig(logger.Config{
		Level:            cfg.Logging.Level,
		Encoding:         cfg.Logging.Encoding,
		OutputPaths:      cfg.Logging.OutputPaths,
		ErrorOutputPaths: cfg.Logging.ErrorOutputPaths,
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := log.Sync(); err != nil {
			panic(err)
		}
	}()

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

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("Error shutting down HTTP server", zap.Error(err))
	}
	grpcServer.GracefulStop()
	if lis != nil {
		if err := lis.Close(); err != nil {
			log.Fatal("Error closing gRPC listener", zap.Error(err))
		}
	}
	log.Info("Shutdown complete")
}
