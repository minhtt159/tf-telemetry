// Package indexer provides functionality to create and configure an Elasticsearch client and bulk
package indexer

import (
	"time"

	elasticsearch "github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"
	"go.uber.org/zap"

	"github.com/minhtt159/tf-telemetry/internal/config"
)

const defaultFlushBytes = 1024 * 1024

func New(cfg *config.Config, logger *zap.Logger) (*elasticsearch.Client, esutil.BulkIndexer, error) {
	// Check if Elasticsearch is configured (has at least one address)
	if len(cfg.Elastic.Addresses) == 0 || cfg.Elastic.Addresses[0] == "" {
		logger.Info("Elasticsearch not configured, using null indexer (data will not be persisted)")
		return nil, NewNullWithLogger(logger), nil
	}

	// Attempt to create ES client. Falls back to null indexer on error for graceful degradation.
	// This allows the server to start and accept telemetry even when ES is unavailable.
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Elastic.Addresses,
		Username:  cfg.Elastic.Username,
		Password:  cfg.Elastic.Password,
	})
	if err != nil {
		logger.Warn("Failed to create Elasticsearch client, using null indexer", zap.Error(err))
		return nil, NewNullWithLogger(logger), nil
	}

	flushBytes := defaultFlushBytes
	if cfg.Elastic.BatchSize > 0 {
		flushBytes = cfg.Elastic.BatchSize
	}

	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         "",
		Client:        es,
		NumWorkers:    2,
		FlushBytes:    flushBytes,
		FlushInterval: time.Duration(cfg.Elastic.FlushInterval) * time.Second,
	})
	if err != nil {
		logger.Warn("Failed to create bulk indexer, using null indexer", zap.Error(err))
		return nil, NewNullWithLogger(logger), nil
	}

	return es, bi, nil
}
