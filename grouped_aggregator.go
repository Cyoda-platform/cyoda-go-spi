package spi

import (
	"context"
	"time"
)

// GroupedAggregator is an optional capability on a storage backend that
// answers a grouped-stats query natively (e.g. via SQL GROUP BY).
//
// May decline a specific request shape via ErrAggregationNotPushdownable;
// the caller (typically the service layer) should then fall through to
// the streaming-tally path via Iterable.
type GroupedAggregator interface {
	GroupedAggregate(
		ctx context.Context,
		model ModelRef,
		groupBy []GroupExpr,
		filter Filter,
		opts GroupedAggregationsOptions,
	) ([]GroupedAggregateBucket, error)
}

// GroupExprKind selects between the lifecycle state and a scalar data path.
type GroupExprKind int

const (
	// GroupExprState groups by the entity's lifecycle state.
	GroupExprState GroupExprKind = iota
	// GroupExprDataPath groups by a scalar JSONPath into entity data.
	GroupExprDataPath
)

// GroupExpr is one dimension of the group-by.
type GroupExpr struct {
	Kind GroupExprKind
	// Path is the JSONPath; only meaningful when Kind == GroupExprDataPath.
	Path string
}

// AggregateOp enumerates the supported per-bucket aggregations.
type AggregateOp string

const (
	AggSum AggregateOp = "sum"
	AggAvg AggregateOp = "avg"
	AggMin AggregateOp = "min"
	AggMax AggregateOp = "max"
	// AggStdev is sample standard deviation (n-1 denominator).
	AggStdev AggregateOp = "stdev"
)

// AggregateExpr is one requested aggregation.
type AggregateExpr struct {
	Op    AggregateOp
	Field string // scalar JSONPath
	// Alias is the response key. If blank, the server synthesizes
	// <op>_<field>.
	Alias string
}

// GroupedAggregationsOptions parameterizes the GroupedAggregate call.
type GroupedAggregationsOptions struct {
	PointInTime *time.Time
	// MaxBuckets is the result cardinality ceiling. The implementation
	// must return ErrGroupCardinalityExceeded if the result would exceed
	// this count.
	MaxBuckets   int
	Aggregations []AggregateExpr
}

// GroupKeyEntry is one (path, value) pair in a bucket's key.
type GroupKeyEntry struct {
	Path string
	// Value is the JSON-typed value: string for scalar/state values, nil
	// for missing/literal-null/non-scalar extracted values.
	Value any
}

// GroupedAggregateBucket is one row of the grouped-stats result.
type GroupedAggregateBucket struct {
	// GroupKey is ordered, matching the request groupBy order.
	GroupKey []GroupKeyEntry
	Count    int64
	// Aggregations maps alias to float64 or nil. nil means the bucket had
	// zero numeric samples for that field.
	Aggregations map[string]any
}
