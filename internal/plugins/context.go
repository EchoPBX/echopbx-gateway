package plugins

import (
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
	"go.uber.org/zap"
)

type pluginContext struct {
	log    *zap.Logger
	bus    sdk.Bus
	config map[string]interface{}
}

func newPluginContext(log *zap.Logger, bus sdk.Bus, cfg map[string]interface{}) sdk.Context {
	return &pluginContext{log: log, bus: bus, config: cfg}
}

func (c *pluginContext) Log() *zap.Logger               { return c.log }
func (c *pluginContext) Bus() sdk.Bus                   { return c.bus }
func (c *pluginContext) Config() map[string]interface{} { return c.config }
