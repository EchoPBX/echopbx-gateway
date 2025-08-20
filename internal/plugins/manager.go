package plugins

import (
	"encoding/json"
	"os"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/internal/events"
	"go.uber.org/zap"
)

type Manifest struct {
	Plugins []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Path    string `json:"path"` // /var/lib/echopbx/plugins/<name>/<version>/server/
		Enabled bool   `json:"enabled"`
	} `json:"plugins"`
}

type Manager struct {
	cfg *config.Config
	log *zap.Logger
	bus *events.Bus
	m   Manifest
}

func NewManager(cfg *config.Config, log *zap.Logger, bus *events.Bus) *Manager {
	return &Manager{cfg: cfg, log: log, bus: bus}
}

func (m *Manager) LoadManifest(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &m.m)
}

func (m *Manager) Reload(path string) {
	if err := m.LoadManifest(path); err != nil {
		m.log.Warn("plugin reload failed", zap.Error(err))
	}
	m.bus.Publish(events.Event{Type: "plugins.reloaded", Data: time.Now().Unix()})
}

func (m *Manager) Shutdown() {}
