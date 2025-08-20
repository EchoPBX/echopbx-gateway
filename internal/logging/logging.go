package logging

import "go.uber.org/zap"

type Cfg struct {
	Level string
	JSON  bool
}

func New(c Cfg) *zap.Logger {
	cfg := zap.NewProductionConfig()
	if !c.JSON {
		cfg.Encoding = "console"
	}
	if c.Level != "" {
		_ = cfg.Level.UnmarshalText([]byte(c.Level))
	}
	l, _ := cfg.Build()
	return l
}
