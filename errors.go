package spi

import "errors"

// ErrNotFound indicates the requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict indicates the write conflicts with a concurrent modification.
var ErrConflict = errors.New("conflict: entity has been modified")

// ErrEpochMismatch indicates the caller's shard epoch is stale relative to
// the cluster view. Retry after refreshing.
var ErrEpochMismatch = errors.New("shard epoch mismatch")

// ErrRetryExhausted indicates the plugin's retry budget for a
// transparently-retried operation was consumed without success.
// Returned by ExtendSchema when CYODA_SCHEMA_EXTEND_MAX_RETRIES
// attempts have completed without success AND the context was not
// cancelled. Callers may choose to retry at a higher level (with
// backoff) or surface the condition to the end user.
//
// Distinct from ErrConflict: ErrConflict means a single attempt hit
// a conflict; ErrRetryExhausted means the plugin exhausted its
// configured retry budget.
var ErrRetryExhausted = errors.New("retry budget exhausted")
