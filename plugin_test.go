package spi

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// stubPlugin is a minimal Plugin for registry tests.
type stubPlugin struct{ name string }

func (s *stubPlugin) Name() string { return s.name }
func (s *stubPlugin) NewFactory(getenv func(string) string, opts ...FactoryOption) (StoreFactory, error) {
	return nil, fmt.Errorf("stub")
}

// stubDescribable is a stubPlugin that also implements DescribablePlugin.
type stubDescribable struct{ stubPlugin }

func (s *stubDescribable) ConfigVars() []ConfigVar {
	return []ConfigVar{{Name: "FOO", Description: "foo var"}}
}

// resetRegistry clears the plugin registry between tests.
func resetRegistry(t *testing.T) {
	t.Helper()
	registryMu.Lock()
	registry = map[string]Plugin{}
	registryMu.Unlock()
}

func TestRegister_StoresPlugin(t *testing.T) {
	resetRegistry(t)
	p := &stubPlugin{name: "memory"}
	Register(p)
	got, ok := GetPlugin("memory")
	if !ok {
		t.Fatal("GetPlugin(\"memory\") returned ok=false")
	}
	if got != p {
		t.Fatalf("got %v, want %v", got, p)
	}
}

func TestGetPlugin_ReturnsFalseForUnknown(t *testing.T) {
	resetRegistry(t)
	_, ok := GetPlugin("nonexistent")
	if ok {
		t.Fatal("expected ok=false for unknown plugin")
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	resetRegistry(t)
	Register(&stubPlugin{name: "memory"})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate Register, got none")
		}
	}()
	Register(&stubPlugin{name: "memory"})
}

func TestRegisteredPlugins_ReturnsAllNames(t *testing.T) {
	resetRegistry(t)
	Register(&stubPlugin{name: "memory"})
	Register(&stubPlugin{name: "postgres"})
	names := RegisteredPlugins()
	if len(names) != 2 {
		t.Fatalf("got %d names, want 2: %v", len(names), names)
	}
	have := map[string]bool{}
	for _, n := range names {
		have[n] = true
	}
	if !have["memory"] || !have["postgres"] {
		t.Fatalf("missing expected names in %v", names)
	}
}

func TestGetPlugin_ConcurrentReads(t *testing.T) {
	resetRegistry(t)
	Register(&stubPlugin{name: "memory"})
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := GetPlugin("memory"); !ok {
				t.Error("GetPlugin returned ok=false under concurrency")
			}
		}()
	}
	wg.Wait()
}

func TestDescribablePlugin_IsOptional(t *testing.T) {
	resetRegistry(t)
	plain := &stubPlugin{name: "plain"}
	described := &stubDescribable{stubPlugin: stubPlugin{name: "described"}}
	Register(plain)
	Register(described)

	if _, ok := any(plain).(DescribablePlugin); ok {
		t.Error("plain stubPlugin should not satisfy DescribablePlugin")
	}
	if _, ok := any(described).(DescribablePlugin); !ok {
		t.Error("stubDescribable should satisfy DescribablePlugin")
	}
}

func TestStartable_IsOptional(t *testing.T) {
	// Plain plugins do not implement Startable; assertion fails cleanly.
	var p Plugin = &stubPlugin{name: "plain"}
	if _, ok := p.(Startable); ok {
		t.Error("stubPlugin should not satisfy Startable")
	}
}

func TestFactoryOption_AppliesInOrder(t *testing.T) {
	b1 := &noopBroadcaster{}
	b2 := &noopBroadcaster{}
	cfg := ApplyFactoryOptions([]FactoryOption{
		WithClusterBroadcaster(b1),
		WithClusterBroadcaster(b2), // later option wins
	})
	if cfg.ClusterBroadcaster() != b2 {
		t.Fatalf("expected b2 to win, got %v", cfg.ClusterBroadcaster())
	}
}

func TestFactoryOption_NoneGivesNil(t *testing.T) {
	cfg := ApplyFactoryOptions(nil)
	if cfg.ClusterBroadcaster() != nil {
		t.Fatalf("expected nil broadcaster, got %v", cfg.ClusterBroadcaster())
	}
}

// Ensures the Plugin interface's NewFactory signature compiles.
var _ = func() Plugin {
	return (*stubPlugin)(nil)
}

func ensureStartable(s Startable, ctx context.Context) error { return s.Start(ctx) }

var _ = ensureStartable
