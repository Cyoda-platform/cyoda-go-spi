package spi

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelsAreDistinct(t *testing.T) {
	sentinels := []error{ErrNotFound, ErrConflict, ErrEpochMismatch}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %d (%v) should not match %d (%v)", i, a, j, b)
			}
		}
	}
}

func TestSentinelsAreMatchedAfterWrap(t *testing.T) {
	wrapped := fmt.Errorf("store layer: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Fatal("wrapped ErrNotFound should match via errors.Is")
	}
}
