package spi_test

import (
	"context"
	"testing"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func TestGroupedAggregatorContract(t *testing.T) {
	var _ spi.GroupedAggregator = (spi.GroupedAggregator)(nil)

	g := spi.GroupExpr{Kind: spi.GroupExprState}
	if g.Kind != spi.GroupExprState {
		t.Fatalf("GroupExprState mismatch")
	}
	g2 := spi.GroupExpr{Kind: spi.GroupExprDataPath, Path: "$.x"}
	if g2.Kind != spi.GroupExprDataPath || g2.Path != "$.x" {
		t.Fatalf("GroupExprDataPath mismatch")
	}

	for _, op := range []spi.AggregateOp{
		spi.AggSum, spi.AggAvg, spi.AggMin, spi.AggMax, spi.AggStdev,
	} {
		if op == "" {
			t.Fatalf("aggregate op is empty")
		}
	}

	var ga spi.GroupedAggregator
	if ga != nil {
		_, _ = ga.GroupedAggregate(
			context.Background(),
			spi.ModelRef{},
			[]spi.GroupExpr{{Kind: spi.GroupExprState}},
			spi.Filter{},
			spi.GroupedAggregationsOptions{MaxBuckets: 10000},
		)
	}
}
