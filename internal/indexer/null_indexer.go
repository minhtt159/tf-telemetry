package indexer

import (
	"context"

	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// NullBulkIndexer is a no-op implementation of BulkIndexer for demo/testing purposes
type NullBulkIndexer struct{}

func (n *NullBulkIndexer) Add(ctx context.Context, item esutil.BulkIndexerItem) error {
	// No-op: just return success without actually indexing
	return nil
}

func (n *NullBulkIndexer) Close(ctx context.Context) error {
	// No-op: nothing to close
	return nil
}

func (n *NullBulkIndexer) Stats() esutil.BulkIndexerStats {
	return esutil.BulkIndexerStats{}
}

// NewNull returns a NullBulkIndexer that doesn't actually index anything
func NewNull() esutil.BulkIndexer {
	return &NullBulkIndexer{}
}
