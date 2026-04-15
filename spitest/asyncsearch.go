package spitest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func runAsyncSearchSuite(t *testing.T, h Harness) {
	t.Run("CreateAndGet", func(t *testing.T) { testASCreateAndGet(t, h) })
	t.Run("GetJob/NotFound", func(t *testing.T) { testASGetJobNotFound(t, h) })
	t.Run("UpdateStatus/Succeeded", func(t *testing.T) { testASUpdateSucceeded(t, h) })
	t.Run("UpdateStatus/Failed", func(t *testing.T) { testASUpdateFailed(t, h) })
	t.Run("SaveAndGetResults/Pagination", func(t *testing.T) { testASResultsPagination(t, h) })
	t.Run("Cancel", func(t *testing.T) { testASCancel(t, h) })
	t.Run("Cancel/NotFound", func(t *testing.T) { testASCancelNotFound(t, h) })
	t.Run("DeleteJob", func(t *testing.T) { testASDeleteJob(t, h) })
	t.Run("ReapExpired", func(t *testing.T) { testASReapExpired(t, h) })
	t.Run("TenantIsolation", func(t *testing.T) { testASTenantIsolation(t, h) })
}

func newSearchJob(tenantID spi.TenantID, id string) *spi.SearchJob {
	return &spi.SearchJob{
		ID:         id,
		TenantID:   tenantID,
		Status:     "RUNNING",
		ModelRef:   spi.ModelRef{EntityName: "m1", ModelVersion: "1"},
		CreateTime: time.Now().UTC(),
	}
}

func testASCreateAndGet(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, err := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, err)
	id := newID()
	job := newSearchJob(tid, id)
	require.NoError(t, as.CreateJob(ctx, job))
	got, err := as.GetJob(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, got.ID)
	require.Equal(t, tid, got.TenantID)
}

func testASGetJobNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.AsyncSearchStore(ctx)
	_, err := as.GetJob(ctx, newID()) // valid UUID, never written
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASUpdateSucceeded(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "Succeeded")
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	finish := time.Now().UTC()
	require.NoError(t, as.UpdateJobStatus(ctx, id, "SUCCESSFUL", 42, "", finish, 100))
	got, err := as.GetJob(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "SUCCESSFUL", got.Status)
	require.Equal(t, 42, got.ResultCount)
	require.Equal(t, int64(100), got.CalcTimeMs)
	require.NotNil(t, got.FinishTime)
}

func testASUpdateFailed(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "Failed")
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	require.NoError(t, as.UpdateJobStatus(ctx, id, "FAILED", 0, "boom", time.Now().UTC(), 10))
	got, _ := as.GetJob(ctx, id)
	require.Equal(t, "FAILED", got.Status)
	require.Equal(t, "boom", got.Error)
}

func testASResultsPagination(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "Pagination")
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	require.NoError(t, as.SaveResults(ctx, id, ids))

	page1, total, err := as.GetResultIDs(ctx, id, 0, 3)
	require.NoError(t, err)
	require.Equal(t, 8, total)
	require.Equal(t, ids[0:3], page1)

	page2, total, err := as.GetResultIDs(ctx, id, 3, 3)
	require.NoError(t, err)
	require.Equal(t, 8, total)
	require.Equal(t, ids[3:6], page2)
}

func testASCancel(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "Cancel")
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	require.NoError(t, as.Cancel(ctx, id))
	got, _ := as.GetJob(ctx, id)
	require.Equal(t, "CANCELLED", got.Status)
	require.NoError(t, as.Cancel(ctx, id), "re-cancelling a terminal job is a no-op")
}

func testASCancelNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.AsyncSearchStore(ctx)
	err := as.Cancel(ctx, newID()) // valid UUID, never written
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASDeleteJob(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	require.NoError(t, as.DeleteJob(ctx, id))
	_, err := as.GetJob(ctx, id)
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASReapExpired(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "ReapExpired")
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	id := newID()
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, id)))
	// Move the job to a terminal state so ReapExpired considers it eligible.
	// Running jobs are intentionally skipped by the reaper (they may still
	// have live goroutines writing results).
	finishTime := h.Now().UTC()
	require.NoError(t, as.UpdateJobStatus(ctx, id, "SUCCESSFUL", 0, "", finishTime, 0))

	ttl := 10 * time.Millisecond
	h.AdvanceClock(ttl + 1*time.Millisecond)
	n, err := as.ReapExpired(ctx, ttl)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
	_, err = as.GetJob(ctx, id)
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASTenantIsolation(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	id := newID()
	asA, _ := h.Factory.AsyncSearchStore(tenantContext(tA))
	asB, _ := h.Factory.AsyncSearchStore(tenantContext(tB))
	require.NoError(t, asA.CreateJob(tenantContext(tA), newSearchJob(tA, id)))
	_, err := asB.GetJob(tenantContext(tB), id)
	require.ErrorIs(t, err, spi.ErrNotFound)
}
