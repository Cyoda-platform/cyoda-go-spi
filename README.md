# cyoda-go-spi

Storage-plugin contract for [cyoda-go](https://github.com/cyoda-platform/cyoda-go).

This module defines the interfaces and value types that any storage
backend must implement. Plugin authors depend only on this module.

## Packages

- `spi` — core interfaces, value types, sentinel errors, `UUIDGenerator`,
  `ClusterBroadcaster`, and `Plugin` registration machinery.
- `spi/predicate` — search-predicate AST types and JSON parse/marshal.

## Dependencies

Standard library only.

## Versioning

Semantic versioning. Breaking changes bump the major version. Pre-1.0,
breaking changes may occur in minor releases.

## License

Apache 2.0.
