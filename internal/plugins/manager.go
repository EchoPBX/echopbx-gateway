package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"plugin"
	"sync"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
	"go.uber.org/zap"
)

type PluginManifest struct {
	Plugins []PluginEntry `json:"plugins"`
}

type PluginEntry struct {
	Name    string       `json:"name"`
	Version string       `json:"version"`
	Server  PluginServer `json:"server"`
	UI      *PluginUI    `json:"ui,omitempty"`
}

type PluginServer struct {
	Path  string `json:"path"`
	Entry string `json:"entry"`
}

type PluginUI struct {
	Path string `json:"path"`
}

type Manager struct {
	cfg     *config.Config
	log     *zap.Logger
	bus     sdk.Bus
	mu      sync.RWMutex
	plugins map[string]sdk.Plugin
}

func NewManager(cfg *config.Config, log *zap.Logger, bus sdk.Bus) *Manager {
	return &Manager{
		cfg:     cfg,
		log:     log,
		bus:     bus,
		plugins: make(map[string]sdk.Plugin),
	}
}

func (m *Manager) LoadManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	var man PluginManifest
	if err := json.Unmarshal(data, &man); err != nil {
		return fmt.Errorf("unmarshal manifest: %w", err)
	}

	m.mu.Lock()
	for name, p := range m.plugins {
		if err := p.Stop(); err != nil {
			m.log.Warn("plugin stop failed", zap.String("name", name), zap.Error(err))
		}
	}
	m.plugins = make(map[string]sdk.Plugin)
	m.mu.Unlock()

	for _, entry := range man.Plugins {
		if err := m.loadOne(entry); err != nil {
			m.log.Error("load plugin failed",
				zap.String("name", entry.Name),
				zap.String("path", entry.Server.Path),
				zap.Error(err))
			continue
		}
	}

	return nil
}

func (m *Manager) loadOne(entry PluginEntry) error {
	if entry.Server.Path == "" || entry.Server.Entry == "" {
		return errors.New("invalid server path/entry")
	}

	m.log.Info("loading plugin", zap.String("name", entry.Name), zap.String("path", entry.Server.Path))

	plug, err := plugin.Open(entry.Server.Path)
	if err != nil {
		return fmt.Errorf("plugin.Open: %w", err)
	}
	sym, err := plug.Lookup(entry.Server.Entry)
	if err != nil {
		return fmt.Errorf("lookup(%s): %w", entry.Server.Entry, err)
	}

	pl, ok := sym.(sdk.Plugin)
	if !ok {
		return fmt.Errorf("symbol %s does not implement sdk.Plugin", entry.Server.Entry)
	}

	ctx := newPluginContext(
		m.log.Named("plugin."+entry.Name),
		m.bus,
		map[string]interface{}{},
	)

	if err := pl.Init(ctx); err != nil {
		return fmt.Errorf("plugin init: %w", err)
	}

	m.mu.Lock()
	m.plugins[entry.Name] = pl
	m.mu.Unlock()

	m.bus.Publish(sdk.Event{
		Type: "plugin.loaded",
		Data: map[string]interface{}{
			"name":    entry.Name,
			"version": entry.Version,
			"time":    time.Now().Unix(),
		},
	})

	m.log.Info("plugin initialized", zap.String("name", entry.Name), zap.String("version", entry.Version))
	return nil
}

func (m *Manager) Reload(path string) {
	if err := m.LoadManifest(path); err != nil {
		m.log.Warn("plugin reload failed", zap.Error(err))
		return
	}
	m.bus.Publish(sdk.Event{
		Type: "plugins.reloaded",
		Data: map[string]interface{}{"time": time.Now().Unix()},
	})
}

func (m *Manager) Shutdown() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for name, p := range m.plugins {
		if err := p.Stop(); err != nil {
			m.log.Warn("plugin stop failed", zap.String("name", name), zap.Error(err))
		}
	}
}
