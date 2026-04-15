package spitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func runKeyValueSuite(t *testing.T, h Harness) {
	t.Run("PutAndGet", func(t *testing.T) { testKVPutAndGet(t, h) })
	t.Run("Get/NotFound", func(t *testing.T) { testKVGetNotFound(t, h) })
	t.Run("Overwrite", func(t *testing.T) { testKVOverwrite(t, h) })
	t.Run("Delete", func(t *testing.T) { testKVDelete(t, h) })
	t.Run("List/Namespace", func(t *testing.T) { testKVListNamespace(t, h) })
	t.Run("TenantIsolation", func(t *testing.T) { testKVTenantIsolation(t, h) })
	t.Run("Value/BinarySafe", func(t *testing.T) { testKVBinarySafe(t, h) })
}

func testKVPutAndGet(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, err := h.Factory.KeyValueStore(ctx)
	require.NoError(t, err)
	require.NoError(t, kv.Put(ctx, "ns1", "k1", []byte("v1")))
	got, err := kv.Get(ctx, "ns1", "k1")
	require.NoError(t, err)
	require.Equal(t, []byte("v1"), got)
}

func testKVGetNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, _ := h.Factory.KeyValueStore(ctx)
	_, err := kv.Get(ctx, "ns", "missing")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testKVOverwrite(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, _ := h.Factory.KeyValueStore(ctx)
	require.NoError(t, kv.Put(ctx, "ns", "k", []byte("old")))
	require.NoError(t, kv.Put(ctx, "ns", "k", []byte("new")))
	got, err := kv.Get(ctx, "ns", "k")
	require.NoError(t, err)
	require.Equal(t, []byte("new"), got)
}

func testKVDelete(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, _ := h.Factory.KeyValueStore(ctx)
	require.NoError(t, kv.Put(ctx, "ns", "k", []byte("v")))
	require.NoError(t, kv.Delete(ctx, "ns", "k"))
	_, err := kv.Get(ctx, "ns", "k")
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testKVListNamespace(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, _ := h.Factory.KeyValueStore(ctx)
	require.NoError(t, kv.Put(ctx, "ns1", "a", []byte("1")))
	require.NoError(t, kv.Put(ctx, "ns1", "b", []byte("2")))
	require.NoError(t, kv.Put(ctx, "ns2", "c", []byte("3")))
	ns1, err := kv.List(ctx, "ns1")
	require.NoError(t, err)
	require.Len(t, ns1, 2)
	require.Equal(t, []byte("1"), ns1["a"])
	require.Equal(t, []byte("2"), ns1["b"])
	ns2, err := kv.List(ctx, "ns2")
	require.NoError(t, err)
	require.Len(t, ns2, 1)
}

func testKVTenantIsolation(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	kvA, _ := h.Factory.KeyValueStore(tenantContext(tA))
	kvB, _ := h.Factory.KeyValueStore(tenantContext(tB))
	require.NoError(t, kvA.Put(tenantContext(tA), "ns", "shared-key", []byte("A")))
	require.NoError(t, kvB.Put(tenantContext(tB), "ns", "shared-key", []byte("B")))
	gotA, _ := kvA.Get(tenantContext(tA), "ns", "shared-key")
	gotB, _ := kvB.Get(tenantContext(tB), "ns", "shared-key")
	require.Equal(t, []byte("A"), gotA)
	require.Equal(t, []byte("B"), gotB)
}

func testKVBinarySafe(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	kv, _ := h.Factory.KeyValueStore(ctx)
	payload := []byte{0x00, 0xFF, 0x01, 0x7F, 0x80, 0xDE, 0xAD, 0xBE, 0xEF, 0x00}
	require.NoError(t, kv.Put(ctx, "ns", "bin", payload))
	got, err := kv.Get(ctx, "ns", "bin")
	require.NoError(t, err)
	require.Equal(t, payload, got)
}
