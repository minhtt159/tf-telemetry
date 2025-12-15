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

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
	"github.com/threatfabric-devops/tf-telemetry/internal/indexer"
	"github.com/threatfabric-devops/tf-telemetry/internal/logger"
	"github.com/threatfabric-devops/tf-telemetry/internal/server"
)

func main() {
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

	svc := server.New(log, bi, cfg)

	grpcServer, lis, err := svc.StartGRPC(cfg)
	if err != nil {
		log.Fatal("failed to start gRPC server", zap.Error(err))
	}

	go func() {
		if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Error("Failed to serve gRPC", zap.Error(err))
		}
	}()

	httpServer := svc.HTTPServer(cfg)
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
