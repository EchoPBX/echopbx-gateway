package plugins

import (
	"encoding/json"
	"os"
	"plugin"
	"sync"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
	"go.uber.org/zap"
)

// PluginManifest describe el archivo plugins.json
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

// Manager controla los plugins cargados
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

// LoadManifest carga plugins.json y los inicializa
func (m *Manager) LoadManifest(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return err
	}

	for _, p := range manifest.Plugins {
		if err := m.loadPlugin(p); err != nil {
			m.log.Error("failed to load plugin",
				zap.String("name", p.Name),
				zap.Error(err))
		}
	}
	return nil
}

func (m *Manager) loadPlugin(entry PluginEntry) error {
	p, err := plugin.Open(entry.Server.Path)
	if err != nil {
		return err
	}
	sym, err := p.Lookup(entry.Server.Entry)
	if err != nil {
		return err
	}

	plug, ok := sym.(sdk.Plugin)
	if !ok {
		return err
	}

	ctx := sdk.NewContext(m.log.With(zap.String("plugin", entry.Name)), m.bus)
	if err := plug.Init(ctx); err != nil {
		return err
	}

	m.mu.Lock()
	m.plugins[entry.Name] = plug
	m.mu.Unlock()

	m.bus.Publish(sdk.Event{
		Type: "plugin.loaded",
		Data: map[string]any{
			"name":    entry.Name,
			"version": entry.Version,
			"time":    time.Now().Unix(),
		},
	})

	m.log.Info("plugin loaded",
		zap.String("name", entry.Name),
		zap.String("version", entry.Version))

	return nil
}

func (m *Manager) Reload(path string) {
	if err := m.LoadManifest(path); err != nil {
		m.log.Warn("plugin reload failed", zap.Error(err))
		return
	}
	m.bus.Publish(sdk.Event{
		Type: "plugins.reloaded",
		Data: map[string]any{
			"time": time.Now().Unix(),
		},
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
