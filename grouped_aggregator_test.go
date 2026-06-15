package spi_test

import (
	"testing"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func TestGroupedAggregatorContract(t *testing.T) {
	// Compile-time signature checks; runtime semantics are exercised by
	// plugin parity tests in cyoda-go's e2e/parity registry.
	var _ spi.GroupedAggregator = (spi.GroupedAggregator)(nil)

	// GroupExpr usage check (compile-only: literals using both Kind values).
	var _ = spi.GroupExpr{Kind: spi.GroupExprState}
	var _ = spi.GroupExpr{Kind: spi.GroupExprDataPath, Path: "$.x"}

	// AggregateOp enumeration check — guards against an empty constant.
	for _, op := range []spi.AggregateOp{
		spi.AggSum, spi.AggAvg, spi.AggMin, spi.AggMax, spi.AggStdev,
	} {
		if op == "" {
			t.Fatalf("aggregate op is empty")
		}
	}
}
