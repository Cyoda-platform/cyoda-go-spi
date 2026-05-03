package spi

import (
	"context"
	"sync"
	"time"
)

// TransactionState holds the state of an active SSI transaction.
// All processor execution is sequential (no goroutines) — see
// docs/superpowers/specs/2026-04-01-workflow-processor-execution-design.md.
// SAVEPOINTs snapshot/restore these maps for ASYNC_NEW_TX rollback isolation.
//
// # Concurrency contract
//
// Plugin implementations of [TransactionManager] must coordinate concurrent
// access to TransactionState's mutable fields using OpMu. Two distinct
// concerns:
//
//  1. Cross-class serialisation — plugin's responsibility, enforced via
//     OpMu. In-flight tx-path operations (Save, Get, Delete, Savepoint,
//     etc.) hold OpMu.RLock; closure operations (Commit, Rollback,
//     RollbackToSavepoint) hold OpMu.Lock. This guarantees Commit/Rollback
//     wait for in-flight ops to drain before mutating or closing the tx.
//     Every method that reads or writes ReadSet, WriteSet, Buffer, Deletes,
//     RolledBack, or Closed must acquire OpMu in the appropriate posture.
//
//  2. Within-class serialisation — application's responsibility, NOT
//     enforced by OpMu. If the application fires multiple in-flight ops on
//     the same tx concurrently (e.g. Save + Get from two goroutines), it
//     must coordinate them externally. OpMu.RLock allows multiple readers,
//     so two RLock-holding ops can race on the underlying map accesses.
//     See [TransactionManager.Join] for the application-side contract.
//
// # Lock posture per field
//
//   - ReadSet, WriteSet, Buffer, Deletes: read or written under OpMu.RLock
//     by in-flight ops; iterated or replaced under OpMu.Lock by Commit /
//     Rollback / RollbackToSavepoint.
//   - RolledBack, Closed: written under OpMu.Lock by Rollback / Commit
//     (typically in a defer); read under OpMu.RLock by every in-flight op
//     so the op fails fast on a closed tx.
//   - ID, TenantID, SnapshotTime: immutable after [TransactionManager.Begin]
//     returns; safe to read without locks.
//
// # Lock order
//
// Plugin implementations must acquire locks in this order to avoid deadlock:
//
//	tx.OpMu  →  factory's per-store mutex  →  manager's per-tx-table mutex
//
// Re-acquiring the manager mutex inside the OpMu region (e.g. Commit
// re-takes manager.mu briefly to update the committed log while still
// holding OpMu.Lock) is permitted as long as the order is preserved.
//
// # Required reading for plugin authors
//
// New methods that touch *TransactionState must declare their OpMu posture
// in a code comment ("Locking discipline: ..."). See
// .claude/rules/tx-state-locking.md for the checklist enforced at code
// review.
type TransactionState struct {
	ID           string
	TenantID     TenantID
	SnapshotTime time.Time
	ReadSet      map[string]bool    // entity IDs read; access under OpMu (see godoc)
	WriteSet     map[string]bool    // entity IDs written; access under OpMu
	Buffer       map[string]*Entity // staged writes; access under OpMu
	Deletes      map[string]bool    // staged deletes; access under OpMu
	RolledBack   bool               // closure flag; written under OpMu.Lock, read under OpMu.RLock
	OpMu         sync.RWMutex       // see TransactionState godoc above for full contract
	Closed       bool               // closure flag; written under OpMu.Lock, read under OpMu.RLock
}

const txContextKey contextKey = "transaction"

// WithTransaction returns a new context carrying the given transaction state.
func WithTransaction(ctx context.Context, tx *TransactionState) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

// GetTransaction returns the transaction state from the context, or nil if none.
func GetTransaction(ctx context.Context) *TransactionState {
	tx, _ := ctx.Value(txContextKey).(*TransactionState)
	return tx
}
