package indexer

import (
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"go.uber.org/zap"

	"github.com/threatfabric-devops/tf-telemetry/internal/config"
)

func New(cfg *config.Config, logger *zap.Logger) (*elasticsearch.Client, esutil.BulkIndexer, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Elastic.Addresses,
		Username:  cfg.Elastic.Username,
		Password:  cfg.Elastic.Password,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error creating ES client: %w", err)
	}

	flushBytes := 1024 * 1024
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
		return nil, nil, fmt.Errorf("error creating bulk indexer: %w", err)
	}

	return es, bi, nil
}
