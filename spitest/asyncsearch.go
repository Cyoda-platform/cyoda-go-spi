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
	job := newSearchJob(tid, "job-1")
	require.NoError(t, as.CreateJob(ctx, job))
	got, err := as.GetJob(ctx, "job-1")
	require.NoError(t, err)
	require.Equal(t, "job-1", got.ID)
	require.Equal(t, tid, got.TenantID)
}

func testASGetJobNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.AsyncSearchStore(ctx)
	_, err := as.GetJob(ctx, "missing")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASUpdateSucceeded(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	finish := time.Now().UTC()
	require.NoError(t, as.UpdateJobStatus(ctx, "j1", "SUCCESSFUL", 42, "", finish, 100))
	got, err := as.GetJob(ctx, "j1")
	require.NoError(t, err)
	require.Equal(t, "SUCCESSFUL", got.Status)
	require.Equal(t, 42, got.ResultCount)
	require.Equal(t, int64(100), got.CalcTimeMs)
	require.NotNil(t, got.FinishTime)
}

func testASUpdateFailed(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	require.NoError(t, as.UpdateJobStatus(ctx, "j1", "FAILED", 0, "boom", time.Now().UTC(), 10))
	got, _ := as.GetJob(ctx, "j1")
	require.Equal(t, "FAILED", got.Status)
	require.Equal(t, "boom", got.Error)
}

func testASResultsPagination(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	require.NoError(t, as.SaveResults(ctx, "j1", ids))

	page1, total, err := as.GetResultIDs(ctx, "j1", 0, 3)
	require.NoError(t, err)
	require.Equal(t, 8, total)
	require.Equal(t, ids[0:3], page1)

	page2, total, err := as.GetResultIDs(ctx, "j1", 3, 3)
	require.NoError(t, err)
	require.Equal(t, 8, total)
	require.Equal(t, ids[3:6], page2)
}

func testASCancel(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	require.NoError(t, as.Cancel(ctx, "j1"))
	got, _ := as.GetJob(ctx, "j1")
	require.Equal(t, "CANCELLED", got.Status)
	require.NoError(t, as.Cancel(ctx, "j1"), "re-cancelling a terminal job is a no-op")
}

func testASCancelNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	as, _ := h.Factory.AsyncSearchStore(ctx)
	err := as.Cancel(ctx, "missing")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASDeleteJob(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	require.NoError(t, as.DeleteJob(ctx, "j1"))
	_, err := as.GetJob(ctx, "j1")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASReapExpired(t *testing.T, h Harness) {
	tid := h.NewTenant()
	ctx := tenantContext(tid)
	as, _ := h.Factory.AsyncSearchStore(ctx)
	require.NoError(t, as.CreateJob(ctx, newSearchJob(tid, "j1")))
	// Move the job to a terminal state so ReapExpired considers it eligible.
	// Running jobs are intentionally skipped by the reaper (they may still
	// have live goroutines writing results).
	finishTime := h.Now().UTC()
	require.NoError(t, as.UpdateJobStatus(ctx, "j1", "SUCCESSFUL", 0, "", finishTime, 0))

	ttl := 10 * time.Millisecond
	h.AdvanceClock(ttl + 1*time.Millisecond)
	n, err := as.ReapExpired(ctx, ttl)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, 1)
	_, err = as.GetJob(ctx, "j1")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testASTenantIsolation(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	asA, _ := h.Factory.AsyncSearchStore(tenantContext(tA))
	asB, _ := h.Factory.AsyncSearchStore(tenantContext(tB))
	require.NoError(t, asA.CreateJob(tenantContext(tA), newSearchJob(tA, "shared-id")))
	_, err := asB.GetJob(tenantContext(tB), "shared-id")
	require.ErrorIs(t, err, spi.ErrNotFound)
}
