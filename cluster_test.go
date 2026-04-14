package spi

import "testing"

// noopBroadcaster is a trivial ClusterBroadcaster used to prove the
// interface is satisfiable with a small in-memory implementation.
type noopBroadcaster struct {
	handlers map[string][]func([]byte)
}

func (n *noopBroadcaster) Broadcast(topic string, payload []byte) {
	for _, h := range n.handlers[topic] {
		h(payload)
	}
}

func (n *noopBroadcaster) Subscribe(topic string, handler func(payload []byte)) {
	if n.handlers == nil {
		n.handlers = map[string][]func([]byte){}
	}
	n.handlers[topic] = append(n.handlers[topic], handler)
}

var _ ClusterBroadcaster = (*noopBroadcaster)(nil)

func TestClusterBroadcaster_RoundTrip(t *testing.T) {
	b := &noopBroadcaster{}
	got := ""
	b.Subscribe("t", func(p []byte) { got = string(p) })
	b.Broadcast("t", []byte("hello"))
	if got != "hello" {
		t.Fatalf("got %q, want %q", got, "hello")
	}
}
