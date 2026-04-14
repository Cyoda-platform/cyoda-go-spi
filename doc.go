// Package spi defines the storage-plugin contract for cyoda-go.
//
// It contains the interfaces and value types that any storage backend
// (memory, postgres, cassandra, redis, ...) must implement, plus the
// plugin registration machinery used by the core to discover backends
// at startup.
//
// Dependencies: standard library only. Plugin authors depend only on
// this module; they do not depend on cyoda-go itself.
//
// Submodule spi/predicate holds the search-predicate AST. Plugins that
// translate predicates to their own query dialect may import it; plugins
// with no search semantics may ignore it.
package spi
