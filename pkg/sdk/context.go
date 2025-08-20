package sdk

import (
	"go.uber.org/zap"
)

// Context se pasa a cada plugin para interactuar con el core.
type Context struct {
	log *zap.Logger
	bus Bus
}

func NewContext(log *zap.Logger, bus Bus) *Context {
	return &Context{log: log, bus: bus}
}

func (c *Context) Log() *zap.Logger { return c.log }
func (c *Context) Bus() Bus         { return c.bus }
