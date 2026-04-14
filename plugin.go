package spi

import (
	"context"
	"fmt"
	"sync"
)

// Plugin is the storage-backend contract. Implementations register
// themselves at init time by calling Register.
type Plugin interface {
	Name() string
	NewFactory(getenv func(string) string, opts ...FactoryOption) (StoreFactory, error)
}

// DescribablePlugin is an optional Plugin capability: it exposes the
// configuration variables the plugin reads, so --help can render them.
type DescribablePlugin interface {
	Plugin
	ConfigVars() []ConfigVar
}

// Startable is an optional StoreFactory capability: the core calls
// Start after HTTP handlers are registered but before serving traffic.
// Plugins that need background goroutines (shard managers, consumers)
// implement this.
type Startable interface {
	Start(ctx context.Context) error
}

// ConfigVar documents a single environment variable a plugin reads.
type ConfigVar struct {
	Name        string
	Description string
	Default     string
	Required    bool
}

// FactoryOption configures a storage factory during NewFactory.
// Plugins receive options via the variadic parameter and resolve them
// with ApplyFactoryOptions.
type FactoryOption func(*factoryConfig)

// factoryConfig is the mutable, unexported accumulator for options.
type factoryConfig struct {
	broadcaster ClusterBroadcaster
}

// FactoryConfig is the read-only view plugins see after resolution.
type FactoryConfig struct {
	inner factoryConfig
}

// ClusterBroadcaster returns the cluster broadcaster supplied via
// WithClusterBroadcaster, or nil if none was supplied.
func (c FactoryConfig) ClusterBroadcaster() ClusterBroadcaster {
	return c.inner.broadcaster
}

// WithClusterBroadcaster injects the cluster broadcaster for plugins
// that use ClusterBroadcaster for cluster-wide notifications.
func WithClusterBroadcaster(b ClusterBroadcaster) FactoryOption {
	return func(c *factoryConfig) { c.broadcaster = b }
}

// ApplyFactoryOptions resolves the variadic options into a read-only
// FactoryConfig. Plugins call this inside NewFactory.
func ApplyFactoryOptions(opts []FactoryOption) FactoryConfig {
	cfg := &factoryConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return FactoryConfig{inner: *cfg}
}

// --- Registry ---

var (
	registryMu sync.RWMutex
	registry   = map[string]Plugin{}
)

// Register adds p to the plugin registry. Register panics if another
// plugin has already been registered under the same Name — a naming
// collision at init time is always a programmer error. Matches the
// database/sql.Register convention.
func Register(p Plugin) {
	registryMu.Lock()
	defer registryMu.Unlock()
	name := p.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("spi: Register called twice for plugin %q", name))
	}
	registry[name] = p
}

// GetPlugin returns the registered plugin with the given name.
func GetPlugin(name string) (Plugin, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// RegisteredPlugins returns the names of all currently registered
// plugins, unordered.
func RegisteredPlugins() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
