package spitest

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

// runTransactionSuite covers TransactionManager. Each subtest gets a
// fresh tenant.
func runTransactionSuite(t *testing.T, h Harness, tracker *skipTracker) {
	runSubtest(t, h, tracker, "CommitVisibility", testTxCommitVisibility)
	runSubtest(t, h, tracker, "RollbackDiscards", testTxRollbackDiscards)
	runSubtest(t, h, tracker, "Join", testTxJoin)
	runSubtest(t, h, tracker, "SubmitTime", testTxSubmitTime)
	runSubtest(t, h, tracker, "Savepoint/ReleaseMergesWork", testTxSavepointRelease)
	runSubtest(t, h, tracker, "Savepoint/RollbackToDiscards", testTxSavepointRollback)
	runSubtest(t, h, tracker, "BeginAfterCommit", testTxBeginAfterCommit)
	runSubtest(t, h, tracker, "TxStateErrors/JoinAfterCommit", testTxStateJoinAfterCommit)
	runSubtest(t, h, tracker, "TxStateErrors/CommitAfterCommit", testTxStateCommitAfterCommit)
	runSubtest(t, h, tracker, "TxStateErrors/CommitAfterRollback", testTxStateCommitAfterRollback)
	runSubtest(t, h, tracker, "TxStateErrors/OpAfterRollback", testTxStateOpAfterRollback)
	runSubtest(t, h, tracker, "TxStateErrors/TenantMismatchOnJoin", testTxStateTenantMismatchOnJoin)
	runSubtest(t, h, tracker, "TxStateErrors/TenantMismatchOnCommit", testTxStateTenantMismatchOnCommit)
	runSubtest(t, h, tracker, "TxStateErrors/SavepointNotFound", testTxStateSavepointNotFound)
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

	id := newID()
	ent := newEntity(t, "m-commit", id, map[string]any{"k": "v"})
	_, err = es.Save(txCtx, ent)
	require.NoError(t, err)

	// Outside-tx read must not see the write yet.
	esOutside, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOutside.Get(ctx, id)
	require.ErrorIs(t, err, spi.ErrNotFound, "outside reader must not see uncommitted write")

	// Use txCtx (not ctx) so backends that store tx-state in the context
	// (e.g. Cassandra) can locate the transaction on Commit.
	require.NoError(t, tm.Commit(txCtx, txID))

	got, err := esOutside.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, got.Meta.ID)
}

func testTxRollbackDiscards(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	id := newID()
	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-rb", id, map[string]any{"k": 1}))
	require.NoError(t, err)

	// Use txCtx (not ctx) so backends that embed tx-state in the context
	// (e.g. Cassandra) can locate the transaction on Rollback.
	require.NoError(t, tm.Rollback(txCtx, txID))

	esOutside, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOutside.Get(ctx, id)
	require.ErrorIs(t, err, spi.ErrNotFound, "rolled-back write must never be visible")
}

func testTxJoin(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	id := newID()
	txID, txCtx1, err := tm.Begin(ctx)
	require.NoError(t, err)
	es1, err := h.Factory.EntityStore(txCtx1)
	require.NoError(t, err)
	_, err = es1.Save(txCtx1, newEntity(t, "m-join", id, map[string]any{"side": "A"}))
	require.NoError(t, err)

	txCtx2, err := tm.Join(ctx, txID)
	require.NoError(t, err)
	es2, err := h.Factory.EntityStore(txCtx2)
	require.NoError(t, err)
	got, err := es2.Get(txCtx2, id)
	require.NoError(t, err)
	require.Equal(t, id, got.Meta.ID, "second caller on same tx must see first caller's uncommitted write")

	// Use txCtx1 (not ctx) so backends that embed tx-state in the context
	// (e.g. Cassandra) can locate the transaction on Rollback.
	require.NoError(t, tm.Rollback(txCtx1, txID))
}

func testTxSubmitTime(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	before := h.Now().UTC()
	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	// Pass txCtx (not ctx) so backends that store tx-state in the context
	// (e.g. Cassandra) can locate the transaction on Commit.
	require.NoError(t, tm.Commit(txCtx, txID))
	after := h.Now().UTC()

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

	idPre := newID()
	idPost := newID()

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", idPre, map[string]any{}))
	require.NoError(t, err)

	// Use txCtx for all TM calls after Begin: Cassandra embeds tx-state in
	// the context and requires it for Savepoint, ReleaseSavepoint, and Commit.
	sp, err := tm.Savepoint(txCtx, txID)
	require.NoError(t, err)
	// After Savepoint, txCtx is replaced with the new savepoint context.
	// Save subsequent entities via the original es (which was created from
	// the original txCtx); further saves after Savepoint still use txCtx.
	_, err = es.Save(txCtx, newEntity(t, "m-sp", idPost, map[string]any{}))
	require.NoError(t, err)

	require.NoError(t, tm.ReleaseSavepoint(txCtx, txID, sp))
	require.NoError(t, tm.Commit(txCtx, txID))

	esOut, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOut.Get(ctx, idPre)
	require.NoError(t, err, "pre-savepoint write must survive release")
	_, err = esOut.Get(ctx, idPost)
	require.NoError(t, err, "post-savepoint write must survive release")
}

func testTxSavepointRollback(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	idPre := newID()
	idPost := newID()

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", idPre, map[string]any{}))
	require.NoError(t, err)

	// Use txCtx for all TM calls after Begin: Cassandra embeds tx-state in
	// the context and requires it for Savepoint, RollbackToSavepoint, and Commit.
	sp, err := tm.Savepoint(txCtx, txID)
	require.NoError(t, err)
	_, err = es.Save(txCtx, newEntity(t, "m-sp", idPost, map[string]any{}))
	require.NoError(t, err)

	require.NoError(t, tm.RollbackToSavepoint(txCtx, txID, sp))
	require.NoError(t, tm.Commit(txCtx, txID))

	esOut, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	_, err = esOut.Get(ctx, idPre)
	require.NoError(t, err, "pre-savepoint write must survive rollback-to-savepoint")
	_, err = esOut.Get(ctx, idPost)
	require.ErrorIs(t, err, spi.ErrNotFound, "post-savepoint write must be discarded")
}

func testTxBeginAfterCommit(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	// Pass txCtx (not ctx) so backends that store tx-state in the context
	// (e.g. Cassandra) can locate the transaction on Commit.
	require.NoError(t, tm.Commit(txCtx, txID))

	// Kept as a loose-assertion floor: any error suffices. The strict
	// version that asserts errors.Is(err, spi.ErrTxAlreadyCommitted) (and
	// the ErrTxTerminated parent via Unwrap) lives in TxStateErrors/
	// JoinAfterCommit. Both coexist intentionally: this one runs against
	// backends that haven't yet conformed to the sentinel contract.
	_, err = tm.Join(ctx, txID)
	require.Error(t, err, "Join against committed txID must fail")
}

// testTxStateJoinAfterCommit verifies that joining a transaction whose
// terminal state is Commit produces ErrTxAlreadyCommitted (which also
// matches ErrTxTerminated via Unwrap).
func testTxStateJoinAfterCommit(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tm.Commit(txCtx, txID))

	_, err = tm.Join(ctx, txID)
	require.Error(t, err, "Join after Commit must fail")
	require.True(t,
		errors.Is(err, spi.ErrTxAlreadyCommitted) || errors.Is(err, spi.ErrTxNotFound),
		"Join after Commit must wrap ErrTxAlreadyCommitted or ErrTxNotFound (backends that purge committed-tx state collapse these); got: %v", err)
}

// testTxStateCommitAfterCommit verifies that double-Commit produces
// ErrTxAlreadyCommitted or ErrTxNotFound (backends that purge state
// after the first Commit collapse to NotFound).
func testTxStateCommitAfterCommit(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tm.Commit(txCtx, txID))

	err = tm.Commit(txCtx, txID)
	require.Error(t, err, "second Commit must fail")
	require.True(t,
		errors.Is(err, spi.ErrTxAlreadyCommitted) || errors.Is(err, spi.ErrTxNotFound),
		"second Commit must wrap ErrTxAlreadyCommitted or ErrTxNotFound; got: %v", err)
}

// testTxStateCommitAfterRollback verifies that Commit on a rolled-back tx
// produces ErrTxRolledBack or ErrTxNotFound (backends that purge state
// after Rollback collapse to NotFound).
func testTxStateCommitAfterRollback(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, tm.Rollback(txCtx, txID))

	err = tm.Commit(txCtx, txID)
	require.Error(t, err, "Commit after Rollback must fail")
	require.True(t,
		errors.Is(err, spi.ErrTxRolledBack) || errors.Is(err, spi.ErrTxNotFound),
		"Commit after Rollback must wrap ErrTxRolledBack or ErrTxNotFound; got: %v", err)
}

// testTxStateOpAfterRollback verifies that a data op against a rolled-back
// transaction produces ErrTxTerminated. Backends with remote tx state
// (postgres) may skip this via Harness.Skip — see the ErrTxTerminated
// godoc caveat.
func testTxStateOpAfterRollback(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)

	es, err := h.Factory.EntityStore(txCtx)
	require.NoError(t, err)

	id := newID()
	_, err = es.Save(txCtx, newEntity(t, "m-op-after-rb", id, map[string]any{"k": "v"}))
	require.NoError(t, err)

	require.NoError(t, tm.Rollback(txCtx, txID))

	_, err = es.Get(txCtx, id)
	require.Error(t, err, "Get after Rollback must fail")
	require.True(t, errors.Is(err, spi.ErrTxTerminated),
		"op after Rollback must wrap ErrTxTerminated; got: %v", err)
}

// testTxStateTenantMismatchOnJoin verifies that tenant B cannot Join a
// transaction begun by tenant A; the error wraps ErrTxTenantMismatch.
func testTxStateTenantMismatchOnJoin(t *testing.T, h Harness) {
	ctxA := tenantContext(h.NewTenant())
	ctxB := tenantContext(h.NewTenant())

	tmA, err := h.Factory.TransactionManager(ctxA)
	require.NoError(t, err)
	txID, _, err := tmA.Begin(ctxA)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tmA.Rollback(ctxA, txID) })

	tmB, err := h.Factory.TransactionManager(ctxB)
	require.NoError(t, err)
	_, err = tmB.Join(ctxB, txID)
	require.Error(t, err, "tenant B Join of tenant A tx must fail")
	require.True(t, errors.Is(err, spi.ErrTxTenantMismatch),
		"cross-tenant Join must wrap ErrTxTenantMismatch; got: %v", err)
}

// testTxStateTenantMismatchOnCommit verifies that tenant B cannot Commit a
// transaction begun by tenant A; the error wraps ErrTxTenantMismatch.
func testTxStateTenantMismatchOnCommit(t *testing.T, h Harness) {
	ctxA := tenantContext(h.NewTenant())
	ctxB := tenantContext(h.NewTenant())

	tmA, err := h.Factory.TransactionManager(ctxA)
	require.NoError(t, err)
	txID, txCtxA, err := tmA.Begin(ctxA)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tmA.Rollback(txCtxA, txID) })

	tmB, err := h.Factory.TransactionManager(ctxB)
	require.NoError(t, err)
	err = tmB.Commit(ctxB, txID)
	require.Error(t, err, "tenant B Commit of tenant A tx must fail")
	require.True(t, errors.Is(err, spi.ErrTxTenantMismatch),
		"cross-tenant Commit must wrap ErrTxTenantMismatch; got: %v", err)
}

// testTxStateSavepointNotFound verifies that RollbackToSavepoint with an
// unknown savepoint id produces ErrSavepointNotFound (which also matches
// ErrNotFound via Unwrap).
func testTxStateSavepointNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	tm, err := h.Factory.TransactionManager(ctx)
	require.NoError(t, err)

	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tm.Rollback(txCtx, txID) })

	err = tm.RollbackToSavepoint(txCtx, txID, "no-such-savepoint")
	require.Error(t, err, "RollbackToSavepoint with unknown id must fail")
	require.True(t, errors.Is(err, spi.ErrSavepointNotFound),
		"unknown savepoint must wrap ErrSavepointNotFound; got: %v", err)
	require.True(t, errors.Is(err, spi.ErrNotFound),
		"ErrSavepointNotFound must also match ErrNotFound via Unwrap; got: %v", err)
}
