package spitest

import (
	"context"
	"encoding/json"
	"iter"
	"testing"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

// spiCtx aliases context.Context for readability in helper signatures.
type spiCtx = context.Context

// newEntity builds a minimal Entity with the given model name, id, and
// JSON-serializable payload. Used across all subtest files.
//
// Note: spi.Entity has fields {Meta EntityMeta, Data []byte} — the ID and
// ModelRef live INSIDE Meta. Callers read them back via got.Meta.ID etc.
// ModelRef.EntityName is the model name; ModelRef.ModelVersion is the version string.
func newEntity(t *testing.T, modelName, id string, payload map[string]any) *spi.Entity {
	t.Helper()
	buf, err := json.Marshal(payload)
	require.NoError(t, err)
	return &spi.Entity{
		Meta: spi.EntityMeta{
			ID:       id,
			ModelRef: spi.ModelRef{EntityName: modelName, ModelVersion: "1"},
		},
		Data: buf,
	}
}

// withTx begins a tx, runs fn with the tx-scoped context, then commits.
// On any error (from fn or Commit), rolls back and fails the test.
//
// Use withTx ONLY for tests that need a committed baseline before the
// real assertion (e.g., "save N entities, then GetAll returns N").
// Tests that must inspect IN-FLIGHT transaction state (CommitVisibility,
// RollbackDiscards, Join, Savepoint variants) cannot use withTx — they
// need explicit control over the Begin/Commit lifecycle.
func withTx(t *testing.T, h Harness, ctx spiCtx, fn func(txCtx spiCtx)) {
	t.Helper()
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)
	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	done := false
	defer func() {
		if !done {
			_ = tm.Rollback(ctx, txID)
		}
	}()
	fn(txCtx)
	require.NoError(t, tm.Commit(ctx, txID))
	done = true
}

// iterSeq wraps a slice into an iter.Seq, matching the SaveAll interface.
func iterSeq[T any](items []T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, it := range items {
			if !yield(it) {
				return
			}
		}
	}
}
