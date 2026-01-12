package indexer

import (
	"context"
	"encoding/json"

	"github.com/elastic/go-elasticsearch/v9/esutil"
	"go.uber.org/zap"
)

// NullBulkIndexer is a no-op implementation of BulkIndexer for demo/testing purposes
// It logs all indexed documents to the provided logger
type NullBulkIndexer struct {
	logger *zap.Logger
}

func (n *NullBulkIndexer) Add(ctx context.Context, item esutil.BulkIndexerItem) error {
	if n.logger != nil {
		// Read and log the document body
		if item.Body != nil {
			var doc map[string]any
			decoder := json.NewDecoder(item.Body)
			if err := decoder.Decode(&doc); err == nil {
				n.logger.Debug("null indexer: document received",
					zap.String("index", item.Index),
					zap.String("action", item.Action),
					zap.Any("document", doc),
				)
			} else {
				n.logger.Debug("null indexer: document received (raw)",
					zap.String("index", item.Index),
					zap.String("action", item.Action),
				)
			}
		}
	}
	return nil
}

func (n *NullBulkIndexer) Close(ctx context.Context) error {
	if n.logger != nil {
		n.logger.Debug("null indexer: closed")
	}
	return nil
}

func (n *NullBulkIndexer) Stats() esutil.BulkIndexerStats {
	return esutil.BulkIndexerStats{}
}

// NewNull returns a NullBulkIndexer that logs documents instead of indexing them
func NewNull() esutil.BulkIndexer {
	return &NullBulkIndexer{}
}

// NewNullWithLogger returns a NullBulkIndexer with logging enabled
func NewNullWithLogger(logger *zap.Logger) esutil.BulkIndexer {
	return &NullBulkIndexer{logger: logger}
}
