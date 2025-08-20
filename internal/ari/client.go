package ari

import (
	"context"
	"crypto/tls"

	"net/http"
	"net/url"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/internal/events"
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Client struct {
	cfg  *config.Config
	log  *zap.Logger
	bus  *events.Bus
	conn *websocket.Conn
	fake bool
}

func NewClient(cfg *config.Config, log *zap.Logger, bus *events.Bus) (*Client, error) {
	return &Client{cfg: cfg, log: log, bus: bus, fake: cfg.ARI.Fake}, nil
}

func (c *Client) Run(ctx context.Context) {
	if c.fake {
		c.runFake(ctx)
		return
	}
	u, _ := url.Parse(c.cfg.ARI.URL)
	d := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: c.cfg.ARI.Insecure}}
	for {
		conn, _, err := d.Dial(u.String(), http.Header{"User-Agent": {"echopbx-gw"}})
		if err != nil {
			c.log.Warn("ari dial failed", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}
		c.conn = conn
		c.log.Info("ARI connected")
		for {
			var msg map[string]any
			if err := conn.ReadJSON(&msg); err != nil {
				c.log.Warn("ARI read", zap.Error(err))
				break
			}
			c.bus.Publish(sdk.Event{Type: "ari.event", Data: msg})
		}
		conn.Close()
		time.Sleep(1 * time.Second)
	}
}

func (c *Client) runFake(ctx context.Context) {
	t := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-t.C:
			c.bus.Publish(sdk.Event{Type: "ari.event", Data: map[string]any{"type": "StasisStart", "ts": t.Unix()}})
		}
	}
}

func (c *Client) Reload(cfg *config.Config) { c.cfg = cfg }
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
