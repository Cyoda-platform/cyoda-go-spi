---
paths:
  - "**/*.go"
---
# Transaction-State Locking Discipline

Plugin implementations of `TransactionManager` must coordinate concurrent
access to `*spi.TransactionState`'s mutable fields via `tx.OpMu`. The full
contract lives on `TransactionState`'s godoc — this rule is the checklist
enforced at code review.

## When this rule applies

Any new method (or change to an existing method) in a TransactionManager
implementation that reads or writes any of:

- `tx.ReadSet`, `tx.WriteSet`, `tx.Buffer`, `tx.Deletes`
- `tx.RolledBack`, `tx.Closed`

The immutable fields (`tx.ID`, `tx.TenantID`, `tx.SnapshotTime`) do not
require locks once `Begin` has returned.

## Required posture

| Operation class | Lock posture | Examples |
|---|---|---|
| In-flight tx-path read or write | `tx.OpMu.RLock` | Save, CompareAndSave, Get, GetAll, GetAsAt, Delete, DeleteAll, Exists, Count, Savepoint |
| Closure (waits for in-flight to drain) | `tx.OpMu.Lock` | Commit, Rollback, RollbackToSavepoint |
| Tx-state-free, manager-state-only | manager mutex only | ReleaseSavepoint |

## Required code comment

Every method touching `*TransactionState` must declare its OpMu posture in
a `// Locking discipline:` comment block on the method godoc. Example:

```go
// Save persists an entity to the transaction's write buffer.
//
// Locking discipline: holds tx.OpMu.RLock for the duration of the
// tx-state read of tx.RolledBack and the writes to tx.Buffer / tx.WriteSet,
// so Commit/Rollback (which take tx.OpMu.Lock) cannot race with us.
// Lock order: tx.OpMu before factory.entityMu.
func (s *EntityStore) Save(ctx context.Context, e *spi.Entity) (...) {
    // ...
}
```

The comment is mandatory because the lock posture is not visible at the
call site. Reviewers verify it; the comment is the audit trail.

## Required test

Every new method touching `*TransactionState` must include a race-detector
regression test that exercises the new method against the matching
contender (Commit or Rollback) under `go test -race`. See
`plugins/memory/concurrency_*_test.go` in the cyoda-go repo for examples.

If the lock posture pairing is mutually exclusive (e.g. RLock vs Lock on
the same OpMu), the race detector cannot flag it directly — the test
becomes a contract-pin sentinel rather than a race reproducer. Document
the test class in its block comment.

## Tenant isolation (paired requirement)

Every method that mutates `*TransactionState` must verify
`uc.Tenant.ID == tx.TenantID` and reject mismatched-tenant callers, even
if the locking discipline is correct. The plugin is the data-access
boundary; tenant isolation cannot be deferred upward. Mirror the existing
Commit/Rollback pattern.

## Lock order invariant

Plugin implementations acquire locks in this order to avoid deadlock:

```
tx.OpMu  →  factory's per-store mutex  →  manager's per-tx-table mutex
```

Re-acquiring the manager mutex inside the OpMu region is permitted as long
as the order is preserved. Inverting the order (e.g. acquiring
factory.entityMu before tx.OpMu) is a defect — caught by review or by the
race detector if any code path holds the inverted order.

## Why

Pre-discipline, plugins drifted into races between in-flight ops and
Commit/Rollback (issues #176, #199 in cyoda-go). Each round of audit
surfaced another method that lacked OpMu coverage. The invariant that
every TransactionState-touching method declares its posture in a code
comment makes drift visible at review and prevents the iterative
whack-a-mole pattern from continuing.
