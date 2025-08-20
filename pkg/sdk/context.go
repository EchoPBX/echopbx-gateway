package sdk

import "go.uber.org/zap"

type Context interface {
	Log() *zap.Logger
	Bus() Bus
	Config() map[string]interface{}
}
