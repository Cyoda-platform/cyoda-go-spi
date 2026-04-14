package spi

import "errors"

// ErrNotFound indicates the requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict indicates the write conflicts with a concurrent modification.
var ErrConflict = errors.New("conflict: entity has been modified")

// ErrEpochMismatch indicates the caller's shard epoch is stale relative to
// the cluster view. Retry after refreshing.
var ErrEpochMismatch = errors.New("shard epoch mismatch")
