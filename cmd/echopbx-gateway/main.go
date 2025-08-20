package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/ari"
	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/internal/events"
	"github.com/EchoPBX/echopbx-gateway/internal/httpserver"
	"github.com/EchoPBX/echopbx-gateway/internal/logging"
	"github.com/EchoPBX/echopbx-gateway/internal/plugins"
	"github.com/EchoPBX/echopbx-gateway/internal/reloader"
	"go.uber.org/zap"
)

func main() {
	cfgPath := os.Getenv("ECHOPBX_CONFIG")
	if cfgPath == "" {
		cfgPath = "/etc/echopbx/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		panic(err)
	}

	logger := logging.New(logging.Cfg{
		Level: cfg.Logging.Level,
		JSON:  cfg.Logging.JSON,
	})
	defer logger.Sync()

	// Banner
	fmt.Println(`
  ______     _           _____  ______   __
 |  ____|   | |         |  __ \|  _ \ \ / /
 | |__   ___| |__   ___ | |__) | |_) \ V / 
 |  __| / __| '_ \ / _ \|  ___/|  _ < > <  
 | |___| (__| | | | (_) | |    | |_) / . \ 
 |______\___|_| |_|\___/|_|    |____/_/ \_\
                                 
EchoPBX Gateway — Asterisk ARI/AMI bridge
------------------------------------------
Config:  ` + cfgPath + `
`)

	bus := events.NewBus()

	srv := httpserver.New(cfg, logger, bus)
	ariClient, err := ari.NewClient(cfg, logger, bus)
	if err != nil {
		logger.Fatal("ari client", zap.Error(err))
	}
	pluginMgr := plugins.NewManager(cfg, logger, bus)
	_ = pluginMgr.LoadManifest("/etc/echopbx/plugins.json")

	// ARI loop
	ctx, cancel := context.WithCancel(context.Background())
	go ariClient.Run(ctx)

	// Hot reload con SIGHUP
	reloader.OnSIGHUP(func() {
		newCfg, err := config.Load(cfgPath)
		if err != nil {
			logger.Warn("config reload failed", zap.Error(err))
			return
		}
		ariClient.Reload(newCfg)
		srv.Reload(newCfg)
		pluginMgr.Reload("/etc/echopbx/plugins.json")
		cfg = newCfg
		logger.Info("reloaded config and plugins")
	})

	addr := fmt.Sprintf("%s:%d", cfg.HTTP.Bind, cfg.HTTP.Port)
	httpSrv := &http.Server{
		Addr:    addr,
		Handler: srv.Router(),
	}

	// HTTP server
	go func() {
		if cfg.HTTP.TLS.Enabled {
			if err := httpSrv.ListenAndServeTLS(cfg.HTTP.TLS.Cert, cfg.HTTP.TLS.Key); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal("http tls", zap.Error(err))
			}
		} else {
			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("http", zap.Error(err))
			}
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down...")
	cancel()
	ariClient.Close()

	ctxTimeout, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	_ = httpSrv.Shutdown(ctxTimeout)
	pluginMgr.Shutdown()
	logger.Info("bye")
}
