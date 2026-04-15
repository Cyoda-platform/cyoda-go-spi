package spitest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

// runTransactionSuite covers TransactionManager. Each subtest gets a
// fresh tenant.
func runTransactionSuite(t *testing.T, h Harness) {
	t.Run("CommitVisibility", func(t *testing.T) { testTxCommitVisibility(t, h) })
	t.Run("RollbackDiscards", func(t *testing.T) { testTxRollbackDiscards(t, h) })
	t.Run("Join", func(t *testing.T) { testTxJoin(t, h) })
	t.Run("SubmitTime", func(t *testing.T) { testTxSubmitTime(t, h) })
	t.Run("Savepoint/ReleaseMergesWork", func(t *testing.T) { testTxSavepointRelease(t, h) })
	t.Run("Savepoint/RollbackToDiscards", func(t *testing.T) { testTxSavepointRollback(t, h) })
	t.Run("BeginAfterCommit", func(t *testing.T) { testTxBeginAfterCommit(t, h) })
}

// Writes in an open tx are invisible to outside readers; after Commit
// they are visible.
func testTxCommitVisibility(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)

	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)

	ent := newEntity(t, "m-commit", "e1", map[string]any{"k": "v"})
	_, err = es.Save(txCtx, ent)
	require.NoError(t, err)

	// Outside-tx read must not see the write yet.
	esOutside, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOutside.Get(ctx, "e1")
	require.ErrorIs(t, err, spi.ErrNotFound, "outside reader must not see uncommitted write")

	require.NoError(t, tm.Commit(ctx, txID))

	got, err := esOutside.Get(ctx, "e1")
	require.NoError(t, err)
	require.Equal(t, "e1", got.Meta.ID)
}

func testTxRollbackDiscards(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-rb", "e1", map[string]any{"k": 1}))
	require.NoError(t, err)

	require.NoError(t, tm.Rollback(ctx, txID))

	esOutside, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOutside.Get(ctx, "e1")
	require.ErrorIs(t, err, spi.ErrNotFound, "rolled-back write must never be visible")
}

func testTxJoin(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx1, err := tm.Begin(ctx)
	require.NoError(t, err)
	es1, err := h.Factory.EntityStore(txCtx1)
	require.NoError(t, err)
	_, err = es1.Save(txCtx1, newEntity(t, "m-join", "e1", map[string]any{"side": "A"}))
	require.NoError(t, err)

	txCtx2, err := tm.Join(ctx, txID)
	require.NoError(t, err)
	es2, err := h.Factory.EntityStore(txCtx2)
	require.NoError(t, err)
	got, err := es2.Get(txCtx2, "e1")
	require.NoError(t, err)
	require.Equal(t, "e1", got.Meta.ID, "second caller on same tx must see first caller's uncommitted write")

	require.NoError(t, tm.Rollback(ctx, txID))
}

func testTxSubmitTime(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	before := time.Now().UTC()
	txID, _, err := tm.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tm.Commit(ctx, txID))
	after := time.Now().UTC()

	submit, err := tm.GetSubmitTime(ctx, txID)
	require.NoError(t, err)
	require.False(t, submit.Before(before.Add(-5*time.Millisecond)),
		"submit time %v must not precede Begin (before=%v)", submit, before)
	require.False(t, submit.After(after.Add(5*time.Millisecond)),
		"submit time %v must not follow Commit-return (after=%v)", submit, after)
}

func testTxSavepointRelease(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", "pre", map[string]any{}))
	require.NoError(t, err)

	sp, err := tm.Savepoint(ctx, txID)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", "post", map[string]any{}))
	require.NoError(t, err)

	require.NoError(t, tm.ReleaseSavepoint(ctx, txID, sp))
	require.NoError(t, tm.Commit(ctx, txID))

	esOut, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOut.Get(ctx, "pre")
	require.NoError(t, err, "pre-savepoint write must survive release")
	_, err = esOut.Get(ctx, "post")
	require.NoError(t, err, "post-savepoint write must survive release")
}

func testTxSavepointRollback(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", "pre", map[string]any{}))
	require.NoError(t, err)

	sp, err := tm.Savepoint(ctx, txID)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", "post", map[string]any{}))
	require.NoError(t, err)

	require.NoError(t, tm.RollbackToSavepoint(ctx, txID, sp))
	require.NoError(t, tm.Commit(ctx, txID))

	esOut, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOut.Get(ctx, "pre")
	require.NoError(t, err, "pre-savepoint write must survive rollback-to-savepoint")
	_, err = esOut.Get(ctx, "post")
	require.ErrorIs(t, err, spi.ErrNotFound, "post-savepoint write must be discarded")
}

func testTxBeginAfterCommit(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, _, err := tm.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tm.Commit(ctx, txID))

	// Re-joining a committed tx must fail. We can't use errors.Is against
	// a specific sentinel because the SPI does not yet define one for
	// "transaction already terminated" — backends return their own errors
	// (e.g., "tx not found in registry"). This assertion is deliberately
	// loose; tightening it requires adding spi.ErrTxClosed or similar
	// to cyoda-go-spi, which is out of scope for Plan 5.
	_, err = tm.Join(ctx, txID)
	require.Error(t, err, "Join against committed txID must fail")
}
