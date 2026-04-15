package spitest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func runAuditSuite(t *testing.T, h Harness) {
	t.Run("RecordAndGet", func(t *testing.T) { testAuditRecordAndGet(t, h) })
	t.Run("GetEvents/Ordering", func(t *testing.T) { testAuditGetEventsOrdering(t, h) })
	t.Run("GetEvents/NotFound", func(t *testing.T) { testAuditGetEventsNotFound(t, h) })
	t.Run("GetEventsByTransaction", func(t *testing.T) { testAuditGetByTx(t, h) })
	t.Run("TenantIsolation", func(t *testing.T) { testAuditTenantIsolation(t, h) })
}

func newSMEvent(txID, state, details string) spi.StateMachineEvent {
	return spi.StateMachineEvent{
		EventType:     spi.SMEventTransitionMade,
		TransactionID: txID,
		State:         state,
		Details:       details,
		Timestamp:     time.Now().UTC(),
	}
}

func testAuditRecordAndGet(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, err := h.Factory.StateMachineAuditStore(ctx)
	require.NoError(t, err)
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx1", "B", "A->B")))
	events, err := as.GetEvents(ctx, "e1")
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "B", events[0].State)
	require.Equal(t, "tx1", events[0].TransactionID)
}

func testAuditGetEventsOrdering(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.StateMachineAuditStore(ctx)
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx1", "B", "A->B")))
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx2", "C", "B->C")))
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx3", "D", "C->D")))
	events, err := as.GetEvents(ctx, "e1")
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, "B", events[0].State)
	require.Equal(t, "C", events[1].State)
	require.Equal(t, "D", events[2].State)
}

func testAuditGetEventsNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.StateMachineAuditStore(ctx)
	events, err := as.GetEvents(ctx, "never")
	require.NoError(t, err)
	require.Len(t, events, 0, "unknown entity returns empty slice, not ErrNotFound")
}

func testAuditGetByTx(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.StateMachineAuditStore(ctx)
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx1", "B", "A->B")))
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx2", "C", "B->C")))
	require.NoError(t, as.Record(ctx, "e1", newSMEvent("tx1", "D", "C->D"))) // tx1 reused
	events, err := as.GetEventsByTransaction(ctx, "e1", "tx1")
	require.NoError(t, err)
	require.Len(t, events, 2)
}

func testAuditTenantIsolation(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	asA, _ := h.Factory.StateMachineAuditStore(tenantContext(tA))
	asB, _ := h.Factory.StateMachineAuditStore(tenantContext(tB))
	require.NoError(t, asA.Record(tenantContext(tA), "e1", newSMEvent("tx", "B", "A->B")))
	events, err := asB.GetEvents(tenantContext(tB), "e1")
	require.NoError(t, err)
	require.Len(t, events, 0)
}
