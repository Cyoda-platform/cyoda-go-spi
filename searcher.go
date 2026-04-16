package spi

import (
	"context"
	"time"
)

// Searcher is an optional interface for storage plugins that support
// search predicate pushdown (e.g. SQL WHERE clauses). Plugins that
// implement Searcher get native query execution; those that don't
// fall back to in-memory filtering.
type Searcher interface {
	Search(ctx context.Context, filter Filter, opts SearchOptions) ([]*Entity, error)
}

// SearchOptions configures pagination, ordering, and scoping for a search.
type SearchOptions struct {
	ModelName    string
	ModelVersion string
	PointInTime  *time.Time
	Limit        int
	Offset       int
	OrderBy      []OrderSpec
}

// OrderSpec defines a single sort clause for search results.
// When OrderBy is empty, the default order is entity_id ascending.
type OrderSpec struct {
	Path   string
	Source FieldSource
	Desc   bool
}
