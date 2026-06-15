package spi_test

import (
	"testing"
	"time"

	spi "github.com/cyoda-platform/cyoda-go-spi"
)

// TestIterableContract verifies the Iterable/Iterator interfaces compile
// and the expected method set is present. Runtime behavior is tested by
// plugin implementations in their own repos.
func TestIterableContract(t *testing.T) {
	// Compile-time signature checks; runtime semantics are exercised by
	// plugin parity tests in cyoda-go's e2e/parity registry.
	var _ spi.Iterable = (spi.Iterable)(nil)
	var _ spi.Iterator = (spi.Iterator)(nil)

	// IterateOptions usage check.
	var opts spi.IterateOptions
	now := time.Now()
	opts.PointInTime = &now
}
