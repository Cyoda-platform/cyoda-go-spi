package spitest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

// Harness bundles a StoreFactory under test with the hooks the conformance
// suite needs. Plugin authors construct one in their test function and
// pass it to StoreFactoryConformance.
type Harness struct {
	// Factory is the StoreFactory under test. StoreFactoryConformance
	// calls Factory.Close when the suite finishes.
	Factory spi.StoreFactory

	// AdvanceClock moves the plugin's virtual clock forward by d.
	// Contract: after AdvanceClock returns, every subsequent timestamp
	// the plugin assigns strictly dominates every timestamp assigned
	// before the call. d must be > 0; d <= 0 panics.
	AdvanceClock func(d time.Duration)

	// NewTenant returns a fresh tenant ID unique within this process.
	// The harness invokes this at the start of every subtest; no subtest
	// reuses another's tenant. Optional; defaults to a uuid-based generator.
	NewTenant func() spi.TenantID
}

// StoreFactoryConformance runs the full conformance suite against h.
// Plugin authors call this from a single top-level test function.
func StoreFactoryConformance(t *testing.T, h Harness) {
	t.Helper()
	mustBeSet(t, h.Factory != nil, "Harness.Factory must be set")
	mustBeSet(t, h.AdvanceClock != nil, "Harness.AdvanceClock must be set")
	if h.NewTenant == nil {
		h.NewTenant = defaultNewTenant
	}
	t.Cleanup(func() { _ = h.Factory.Close() })

	t.Run("Transaction", func(t *testing.T) { runTransactionSuite(t, h) })
	t.Run("Entity", func(t *testing.T) { runEntitySuite(t, h) })
	t.Run("Model", func(t *testing.T) { runModelSuite(t, h) })
	t.Run("KeyValue", func(t *testing.T) { runKeyValueSuite(t, h) })
	t.Run("Message", func(t *testing.T) { runMessageSuite(t, h) })
	t.Run("Workflow", func(t *testing.T) { runWorkflowSuite(t, h) })
	t.Run("Audit", func(t *testing.T) { runAuditSuite(t, h) })
	t.Run("AsyncSearch", func(t *testing.T) { runAsyncSearchSuite(t, h) })
}

func defaultNewTenant() spi.TenantID {
	return spi.TenantID("conformance-" + uuid.NewString())
}

// tenantContext returns a background context carrying a synthetic
// UserContext for the given tenant, sufficient for plugin tenant
// resolution.
func tenantContext(tenant spi.TenantID) context.Context {
	return spi.WithUserContext(context.Background(), &spi.UserContext{
		UserID:   "conformance-test",
		UserName: "conformance",
		Tenant:   spi.Tenant{ID: tenant, Name: string(tenant)},
	})
}

// mustBeSet is a local tiny assertion so this file doesn't depend on testify
// at the entry-point level. Per-subtest files may use testify.
func mustBeSet(t *testing.T, cond bool, msg string) {
	t.Helper()
	if !cond {
		t.Fatal(msg)
	}
}

func runMessageSuite(t *testing.T, h Harness)     { t.Skip("not implemented yet") }
func runWorkflowSuite(t *testing.T, h Harness)    { t.Skip("not implemented yet") }
func runAuditSuite(t *testing.T, h Harness)       { t.Skip("not implemented yet") }
func runAsyncSearchSuite(t *testing.T, h Harness) { t.Skip("not implemented yet") }
