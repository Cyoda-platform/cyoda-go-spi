package spitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func runWorkflowSuite(t *testing.T, h Harness) {
	t.Run("SaveAndGet", func(t *testing.T) { testWfSaveAndGet(t, h) })
	t.Run("Get/EmptyModel", func(t *testing.T) { testWfGetEmpty(t, h) })
	t.Run("Overwrite", func(t *testing.T) { testWfOverwrite(t, h) })
	t.Run("Delete", func(t *testing.T) { testWfDelete(t, h) })
	t.Run("TenantIsolation", func(t *testing.T) { testWfTenantIsolation(t, h) })
}

func newWorkflowDef(name string) spi.WorkflowDefinition {
	return spi.WorkflowDefinition{Name: name, Version: "1", InitialState: "initial", Active: true,
		States: map[string]spi.StateDefinition{"initial": {}}}
}

func testWfSaveAndGet(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	ws, err := h.Factory.WorkflowStore(ctx)
	require.NoError(t, err)
	mref := spi.ModelRef{EntityName: "m1", ModelVersion: "1"}
	defs := []spi.WorkflowDefinition{newWorkflowDef("wf1"), newWorkflowDef("wf2")}
	require.NoError(t, ws.Save(ctx, mref, defs))
	got, err := ws.Get(ctx, mref)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func testWfGetEmpty(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	ws, _ := h.Factory.WorkflowStore(ctx)
	got, err := ws.Get(ctx, spi.ModelRef{EntityName: "never", ModelVersion: "1"})
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Len(t, got, 0)
}

func testWfOverwrite(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	ws, _ := h.Factory.WorkflowStore(ctx)
	mref := spi.ModelRef{EntityName: "m1", ModelVersion: "1"}
	require.NoError(t, ws.Save(ctx, mref, []spi.WorkflowDefinition{newWorkflowDef("wf1")}))
	require.NoError(t, ws.Save(ctx, mref, []spi.WorkflowDefinition{newWorkflowDef("wf2"), newWorkflowDef("wf3")}))
	got, err := ws.Get(ctx, mref)
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func testWfDelete(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	ws, _ := h.Factory.WorkflowStore(ctx)
	mref := spi.ModelRef{EntityName: "m1", ModelVersion: "1"}
	require.NoError(t, ws.Save(ctx, mref, []spi.WorkflowDefinition{newWorkflowDef("wf1")}))
	require.NoError(t, ws.Delete(ctx, mref))
	got, err := ws.Get(ctx, mref)
	require.NoError(t, err)
	require.Len(t, got, 0)
}

func testWfTenantIsolation(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	wsA, _ := h.Factory.WorkflowStore(tenantContext(tA))
	wsB, _ := h.Factory.WorkflowStore(tenantContext(tB))
	mref := spi.ModelRef{EntityName: "m1", ModelVersion: "1"}
	require.NoError(t, wsA.Save(tenantContext(tA), mref, []spi.WorkflowDefinition{newWorkflowDef("wf1")}))
	got, err := wsB.Get(tenantContext(tB), mref)
	require.NoError(t, err)
	require.Len(t, got, 0)
}
