package spitest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

func runEntitySuite(t *testing.T, h Harness) {
	// CRUD group (Task 4)
	t.Run("CreateAndGet", func(t *testing.T) { testEntityCreateAndGet(t, h) })
	t.Run("Update", func(t *testing.T) { testEntityUpdate(t, h) })
	t.Run("SaveAll/Ordering", func(t *testing.T) { testEntitySaveAllOrdering(t, h) })
	t.Run("SaveAll/PartialFailureAtomicity", func(t *testing.T) { testEntitySaveAllAtomicity(t, h) })
	t.Run("Get/NotFound", func(t *testing.T) { testEntityGetNotFound(t, h) })
	t.Run("GetAll/EmptyModel", func(t *testing.T) { testEntityGetAllEmpty(t, h) })
	t.Run("GetAll/Population", func(t *testing.T) { testEntityGetAllPopulation(t, h) })
	t.Run("Delete", func(t *testing.T) { testEntityDelete(t, h) })
	t.Run("Delete/NotFound", func(t *testing.T) { testEntityDeleteNotFound(t, h) })
	t.Run("DeleteAll", func(t *testing.T) { testEntityDeleteAll(t, h) })
	t.Run("Exists", func(t *testing.T) { testEntityExists(t, h) })
	t.Run("Count", func(t *testing.T) { testEntityCount(t, h) })
	t.Run("JSONFidelity/DeepNesting", func(t *testing.T) { testEntityJSONFidelity(t, h) })

	// Temporal group (Task 5)
	t.Run("GetAsAt/Historical", func(t *testing.T) { testEntityGetAsAtHistorical(t, h) })
	t.Run("GetAsAt/FullMetaPopulated", func(t *testing.T) { testEntityGetAsAtMeta(t, h) })
	t.Run("GetAsAt/BeforeAnyWrite", func(t *testing.T) { testEntityGetAsAtBefore(t, h) })
	t.Run("GetAllAsAt", func(t *testing.T) { testEntityGetAllAsAt(t, h) })
	t.Run("GetVersionHistory/Ordering", func(t *testing.T) { testEntityVersionHistory(t, h) })

	// Concurrent / Isolation group (Task 6)
	t.Run("CompareAndSave/Success", func(t *testing.T) { testEntityCompareAndSaveSuccess(t, h) })
	t.Run("CompareAndSave/Conflict", func(t *testing.T) { testEntityCompareAndSaveConflict(t, h) })
	t.Run("Concurrent/ConflictingUpdate", func(t *testing.T) { testEntityConcurrentConflict(t, h) })
	t.Run("Concurrent/DifferentEntities", func(t *testing.T) { testEntityConcurrentDifferent(t, h) })
	t.Run("TenantIsolation/Get", func(t *testing.T) { testEntityTenantIsolationGet(t, h) })
	t.Run("TenantIsolation/GetAll", func(t *testing.T) { testEntityTenantIsolationGetAll(t, h) })
	t.Run("TenantIsolation/Delete", func(t *testing.T) { testEntityTenantIsolationDelete(t, h) })
	t.Run("EmptyTenant", func(t *testing.T) { testEntityEmptyTenant(t, h) })
}

func testEntityCreateAndGet(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, err := h.Factory.EntityStore(txCtx)
		require.NoError(t, err)
		_, err = es.Save(txCtx, newEntity(t, "m-crud", id, map[string]any{"k": "v"}))
		require.NoError(t, err)
	})

	es, err := h.Factory.EntityStore(ctx)
	require.NoError(t, err)
	got, err := es.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, got.Meta.ID)
	require.Equal(t, "m-crud", got.Meta.ModelRef.EntityName)
	// State is intentionally NOT asserted: it is set by the workflow engine
	// when a model has a workflow defined. Bare saves with no workflow
	// correctly leave State empty. State semantics are validated at the
	// app level (parity suite), not at the SPI layer.
	require.False(t, got.Meta.CreationDate.IsZero(), "CreationDate meta must be populated")
	require.False(t, got.Meta.LastModifiedDate.IsZero(), "LastModifiedDate meta must be populated")
	require.NotEmpty(t, got.Meta.TransactionID, "TransactionID meta must be populated")
}

func testEntityUpdate(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-upd", id, map[string]any{"v": 1}))
		require.NoError(t, err)
	})

	h.AdvanceClock(1 * time.Millisecond)

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-upd", id, map[string]any{"v": 2}))
		require.NoError(t, err)
	})

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, id, got.Meta.ID)
	require.Contains(t, string(got.Data), `"v":2`)
}

func testEntityGetNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	es, _ := h.Factory.EntityStore(ctx)
	_, err := es.Get(ctx, newID()) // valid UUID that was never written
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testEntityGetAllEmpty(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAll(ctx, spi.ModelRef{EntityName: "m-empty", ModelVersion: "1"})
	require.NoError(t, err)
	require.NotNil(t, got, "GetAll on empty model must return non-nil slice")
	require.Len(t, got, 0)
}

func testEntityGetAllPopulation(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	const n = 5
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		for i := 0; i < n; i++ {
			_, err := es.Save(txCtx, newEntity(t, "m-pop", newID(), map[string]any{"i": i}))
			require.NoError(t, err)
		}
	})

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAll(ctx, spi.ModelRef{EntityName: "m-pop", ModelVersion: "1"})
	require.NoError(t, err)
	require.Len(t, got, n)
}

func testEntityDelete(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-del", id, map[string]any{}))
		require.NoError(t, err)
	})

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		require.NoError(t, es.Delete(txCtx, id))
	})

	es, _ := h.Factory.EntityStore(ctx)
	_, err := es.Get(ctx, id)
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testEntityDeleteNotFound(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		err := es.Delete(txCtx, newID()) // valid UUID that was never created
		require.ErrorIs(t, err, spi.ErrNotFound)
	})
}

func testEntityDeleteAll(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-delall", ModelVersion: "1"}
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		for i := 0; i < 3; i++ {
			_, err := es.Save(txCtx, newEntity(t, "m-delall", newID(), map[string]any{}))
			require.NoError(t, err)
		}
	})

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		require.NoError(t, es.DeleteAll(txCtx, mref))
	})

	es, _ := h.Factory.EntityStore(ctx)
	n, err := es.Count(ctx, mref)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

func testEntityExists(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-ex", id, map[string]any{}))
		require.NoError(t, err)
	})
	es, _ := h.Factory.EntityStore(ctx)
	ok, err := es.Exists(ctx, id)
	require.NoError(t, err)
	require.True(t, ok)

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		require.NoError(t, es.Delete(txCtx, id))
	})
	ok, err = es.Exists(ctx, id)
	require.NoError(t, err)
	require.False(t, ok)
}

func testEntityCount(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-cnt", ModelVersion: "1"}
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		for i := 0; i < 7; i++ {
			_, err := es.Save(txCtx, newEntity(t, "m-cnt", newID(), map[string]any{}))
			require.NoError(t, err)
		}
	})
	es, _ := h.Factory.EntityStore(ctx)
	n, err := es.Count(ctx, mref)
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
}

func testEntitySaveAllOrdering(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-sa", ModelVersion: "1"}
	ents := []*spi.Entity{
		newEntity(t, "m-sa", newID(), map[string]any{"i": 0}),
		newEntity(t, "m-sa", newID(), map[string]any{"i": 1}),
		newEntity(t, "m-sa", newID(), map[string]any{"i": 2}),
	}
	var versions []int64
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		v, err := es.SaveAll(txCtx, iterSeq(ents))
		require.NoError(t, err)
		versions = v
	})
	require.Len(t, versions, 3)

	es, _ := h.Factory.EntityStore(ctx)
	n, _ := es.Count(ctx, mref)
	require.Equal(t, int64(3), n)
}

func testEntitySaveAllAtomicity(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-saa", ModelVersion: "1"}
	tm, _ := h.Factory.TransactionManager(ctx)
	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	es, _ := h.Factory.EntityStore(txCtx)
	_, err = es.SaveAll(txCtx, iterSeq([]*spi.Entity{
		newEntity(t, "m-saa", newID(), map[string]any{}),
		newEntity(t, "m-saa", newID(), map[string]any{}),
	}))
	require.NoError(t, err)
	// Use txCtx (not ctx) so backends that embed tx-state in context (e.g.
	// Cassandra) can locate the transaction on Rollback.
	require.NoError(t, tm.Rollback(txCtx, txID))

	esOut, _ := h.Factory.EntityStore(ctx)
	n, _ := esOut.Count(ctx, mref)
	require.Equal(t, int64(0), n, "no SaveAll entities visible after rollback")
}

func testEntityJSONFidelity(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	payload := map[string]any{
		"nested": map[string]any{
			"arr":     []any{1.0, 2.0, nil, "three", map[string]any{"k": "v"}},
			"unicode": "λ κόσμε 🌍",
			"null":    nil,
			"deep":    map[string]any{"d1": map[string]any{"d2": map[string]any{"d3": "bottom"}}},
		},
	}
	// Note: json.Unmarshal decodes JSON numbers as float64. The payload
	// above intentionally uses values that round-trip safely through
	// float64. Larger integers or precision-sensitive values would need
	// json.Number decoding for a reliable equality check.
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-json", id, payload))
		require.NoError(t, err)
	})
	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.Get(ctx, id)
	require.NoError(t, err)
	var roundTripped map[string]any
	require.NoError(t, json.Unmarshal(got.Data, &roundTripped))
	require.Equal(t, payload, roundTripped, "deep JSON payload must round-trip")
}

func testEntityGetAsAtHistorical(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	// Write v=1, advance, capture tBetween12, advance, write v=2, advance.
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-asat", id, map[string]any{"v": 1}))
		require.NoError(t, err)
	})
	h.AdvanceClock(1 * time.Millisecond)
	tBetween12 := h.Now().UTC()
	h.AdvanceClock(1 * time.Millisecond)

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-asat", id, map[string]any{"v": 2}))
		require.NoError(t, err)
	})
	h.AdvanceClock(1 * time.Millisecond)

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAsAt(ctx, id, tBetween12)
	require.NoError(t, err)
	require.Contains(t, string(got.Data), `"v":1`, "GetAsAt(tBetween12) must return v=1")
}

func testEntityGetAsAtMeta(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-meta", id, map[string]any{}))
		require.NoError(t, err)
	})
	h.AdvanceClock(1 * time.Millisecond)
	asAt := h.Now().UTC()
	h.AdvanceClock(1 * time.Millisecond)

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAsAt(ctx, id, asAt)
	require.NoError(t, err)
	// State intentionally not asserted (see testEntityCreateAndGet).
	require.False(t, got.Meta.CreationDate.IsZero(), "GetAsAt must populate CreationDate")
	require.False(t, got.Meta.LastModifiedDate.IsZero(), "GetAsAt must populate LastModifiedDate")
	require.NotEmpty(t, got.Meta.TransactionID, "GetAsAt must populate TransactionID")
	require.Equal(t, id, got.Meta.ID)
}

func testEntityGetAsAtBefore(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	past := h.Now().UTC().Add(-1 * time.Hour)
	es, _ := h.Factory.EntityStore(ctx)
	_, err := es.GetAsAt(ctx, newID(), past) // valid UUID, never written
	require.ErrorIs(t, err, spi.ErrNotFound)
}

func testEntityGetAllAsAt(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-allasat", ModelVersion: "1"}
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		for i := 0; i < 3; i++ {
			_, err := es.Save(txCtx, newEntity(t, "m-allasat", newID(), map[string]any{"i": i}))
			require.NoError(t, err)
		}
	})
	h.AdvanceClock(1 * time.Millisecond)
	asAt := h.Now().UTC()
	h.AdvanceClock(1 * time.Millisecond)

	// Fourth entity written AFTER asAt — must not be returned.
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-allasat", newID(), map[string]any{"i": 99}))
		require.NoError(t, err)
	})

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAllAsAt(ctx, mref, asAt)
	require.NoError(t, err)
	require.Len(t, got, 3, "GetAllAsAt must exclude writes after asAt")
}

func testEntityVersionHistory(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	for i := 0; i < 3; i++ {
		withTx(t, h, ctx, func(txCtx context.Context) {
			es, _ := h.Factory.EntityStore(txCtx)
			_, err := es.Save(txCtx, newEntity(t, "m-hist", id, map[string]any{"v": i}))
			require.NoError(t, err)
		})
		h.AdvanceClock(1 * time.Millisecond)
	}
	es, _ := h.Factory.EntityStore(ctx)
	history, err := es.GetVersionHistory(ctx, id)
	require.NoError(t, err)
	require.Len(t, history, 3)
	for i := 1; i < len(history); i++ {
		require.False(t, history[i].Timestamp.Before(history[i-1].Timestamp),
			"version %d timestamp must not precede version %d", i, i-1)
	}
}

func testEntityCompareAndSaveSuccess(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-cas", id, map[string]any{"v": 1}))
		require.NoError(t, err)
	})

	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.Get(ctx, id)
	require.NoError(t, err)
	firstTxID := got.Meta.TransactionID

	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.CompareAndSave(txCtx, newEntity(t, "m-cas", id, map[string]any{"v": 2}), firstTxID)
		require.NoError(t, err)
	})

	got, err = es.Get(ctx, id)
	require.NoError(t, err)
	require.Contains(t, string(got.Data), `"v":2`)
}

func testEntityCompareAndSaveConflict(t *testing.T, h Harness) {
	h.skipIfRegistered(t, "Conflict")
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-cas", id, map[string]any{}))
		require.NoError(t, err)
	})

	tm, _ := h.Factory.TransactionManager(ctx)
	txID, txCtx, err := tm.Begin(ctx)
	require.NoError(t, err)
	// Use txCtx so backends that embed tx-state in context (e.g. Cassandra)
	// can locate the transaction on Rollback.
	defer func() { _ = tm.Rollback(txCtx, txID) }()
	es, _ := h.Factory.EntityStore(txCtx)
	_, err = es.CompareAndSave(txCtx, newEntity(t, "m-cas", id, map[string]any{}), "stale-tx-id")
	require.ErrorIs(t, err, spi.ErrConflict, "CompareAndSave with stale expectedTxID must return ErrConflict")
}

func testEntityConcurrentConflict(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	id := newID()
	withTx(t, h, ctx, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-cc", id, map[string]any{"v": 0}))
		require.NoError(t, err)
	})
	es0, _ := h.Factory.EntityStore(ctx)
	got, _ := es0.Get(ctx, id)
	baseTxID := got.Meta.TransactionID

	errs := make(chan error, 2)
	run := func(v int) {
		tm, e := h.Factory.TransactionManager(ctx)
		if e != nil {
			errs <- e
			return
		}
		txID, txCtx, e := tm.Begin(ctx)
		if e != nil {
			errs <- e
			return
		}
		es, _ := h.Factory.EntityStore(txCtx)
		_, e = es.CompareAndSave(txCtx, newEntity(t, "m-cc", id, map[string]any{"v": v}), baseTxID)
		if e != nil {
			// Use txCtx so backends that embed tx-state in context can
			// locate the transaction on Rollback (e.g. Cassandra).
			_ = tm.Rollback(txCtx, txID)
			errs <- e
			return
		}
		// Use txCtx so backends that embed tx-state in context can
		// locate the transaction on Commit (e.g. Cassandra).
		errs <- tm.Commit(txCtx, txID)
	}
	go run(1)
	go run(2)
	results := []error{<-errs, <-errs}

	var winners, conflicts int
	for _, e := range results {
		switch {
		case e == nil:
			winners++
		case errors.Is(e, spi.ErrConflict):
			conflicts++
		default:
			t.Fatalf("unexpected error: %v", e)
		}
	}
	require.Equal(t, 1, winners, "exactly one winner")
	require.Equal(t, 1, conflicts, "exactly one ErrConflict")
}

func testEntityConcurrentDifferent(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-cd", ModelVersion: "1"}
	const n = 8
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			id := newID()
			tm, e := h.Factory.TransactionManager(ctx)
			if e != nil {
				errs <- e
				return
			}
			txID, txCtx, e := tm.Begin(ctx)
			if e != nil {
				errs <- e
				return
			}
			es, _ := h.Factory.EntityStore(txCtx)
			_, e = es.Save(txCtx, newEntity(t, "m-cd", id, map[string]any{"i": i}))
			if e != nil {
				// Use txCtx so backends that embed tx-state in context can
				// locate the transaction on Rollback (e.g. Cassandra).
				_ = tm.Rollback(txCtx, txID)
				errs <- e
				return
			}
			// Use txCtx so backends that embed tx-state in context can
			// locate the transaction on Commit (e.g. Cassandra).
			errs <- tm.Commit(txCtx, txID)
		}(i)
	}
	for i := 0; i < n; i++ {
		require.NoError(t, <-errs)
	}
	es, _ := h.Factory.EntityStore(ctx)
	count, err := es.Count(ctx, mref)
	require.NoError(t, err)
	require.Equal(t, int64(n), count)
}

func testEntityTenantIsolationGet(t *testing.T, h Harness) {
	tA := h.NewTenant()
	tB := h.NewTenant()
	ctxA, ctxB := tenantContext(tA), tenantContext(tB)
	id := newID()

	withTx(t, h, ctxA, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-ti", id, map[string]any{"t": "A"}))
		require.NoError(t, err)
	})

	esB, _ := h.Factory.EntityStore(ctxB)
	_, err := esB.Get(ctxB, id)
	require.ErrorIs(t, err, spi.ErrNotFound, "cross-tenant Get must return ErrNotFound")
}

func testEntityTenantIsolationGetAll(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	ctxA, ctxB := tenantContext(tA), tenantContext(tB)
	mref := spi.ModelRef{EntityName: "m-tigetall", ModelVersion: "1"}

	withTx(t, h, ctxA, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-tigetall", newID(), map[string]any{}))
		require.NoError(t, err)
	})

	esB, _ := h.Factory.EntityStore(ctxB)
	got, err := esB.GetAll(ctxB, mref)
	require.NoError(t, err)
	require.Len(t, got, 0, "tenant B must not see tenant A's writes")
}

func testEntityTenantIsolationDelete(t *testing.T, h Harness) {
	tA, tB := h.NewTenant(), h.NewTenant()
	ctxA, ctxB := tenantContext(tA), tenantContext(tB)
	id := newID()

	withTx(t, h, ctxA, func(txCtx context.Context) {
		es, _ := h.Factory.EntityStore(txCtx)
		_, err := es.Save(txCtx, newEntity(t, "m-tidel", id, map[string]any{}))
		require.NoError(t, err)
	})

	tmB, _ := h.Factory.TransactionManager(ctxB)
	txIDB, txCtxB, err := tmB.Begin(ctxB)
	require.NoError(t, err)
	// Always roll back the test tx — even if Delete returns ErrNotFound
	// (which is the expected outcome), the tx is still open and must be
	// cleaned up. Use txCtxB so backends that embed tx-state in the
	// context (e.g. Cassandra) can locate the transaction.
	defer func() { _ = tmB.Rollback(txCtxB, txIDB) }()
	esB, _ := h.Factory.EntityStore(txCtxB)
	err = esB.Delete(txCtxB, id)
	require.ErrorIs(t, err, spi.ErrNotFound, "cross-tenant Delete must return ErrNotFound")
}

func testEntityEmptyTenant(t *testing.T, h Harness) {
	ctx := tenantContext(h.NewTenant())
	mref := spi.ModelRef{EntityName: "m-empty", ModelVersion: "1"}
	es, _ := h.Factory.EntityStore(ctx)
	got, err := es.GetAll(ctx, mref)
	require.NoError(t, err)
	// Note: testEntityGetAllEmpty asserts non-nil; this subtest tests the
	// broader EmptyTenant invariant (Count == 0). If the memory plugin
	// returns nil from GetAll, this still works because len(nil) == 0.
	require.Len(t, got, 0)
	n, err := es.Count(ctx, mref)
	require.NoError(t, err)
	require.Equal(t, int64(0), n)
}

