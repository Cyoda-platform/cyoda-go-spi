package spi

// ClusterBroadcaster delivers opaque payloads to peer nodes on a named
// topic. Semantics are fire-and-forget, best-effort: no ordering, no
// anti-entropy, no persistence. Payloads that need ordering or delivery
// guarantees should use a backend-internal transport (e.g. a message
// broker) rather than this interface.
//
// Broadcast is non-blocking; it enqueues the payload and returns.
// Subscribe registers a handler called for every message received on
// the topic. Handlers run on the broadcaster's goroutines; they must
// not block indefinitely.
//
// Typical use: a plugin needs eventually-consistent cluster-wide
// notifications (cache invalidation, clock gossip, topology hints).
type ClusterBroadcaster interface {
	Broadcast(topic string, payload []byte)
	Subscribe(topic string, handler func(payload []byte))
}
