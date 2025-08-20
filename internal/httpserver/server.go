package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/internal/jwt"
	"github.com/EchoPBX/echopbx-gateway/pkg/sdk"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Server struct {
	cfg *config.Config
	log *zap.Logger
	bus sdk.Bus
	r   *chi.Mux
	jwt *jwt.Validator
}

func New(cfg *config.Config, log *zap.Logger, bus sdk.Bus) *Server {
	v, _ := jwt.NewValidator(cfg.Auth.JWTPublicKeys, cfg.Auth.Issuer, cfg.Auth.Audience)
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}))
	s := &Server{cfg: cfg, log: log, bus: bus, r: r, jwt: v}
	s.routes()
	return s
}

func (s *Server) Router() http.Handler      { return s.r }
func (s *Server) Reload(cfg *config.Config) { s.cfg = cfg }

func (s *Server) routes() {
	s.r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	s.r.Get("/v1/info", s.auth(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"name": "echopbx-gateway", "time": time.Now().UTC()}
		_ = json.NewEncoder(w).Encode(resp)
	}))

	s.r.Get("/v1/calls", s.auth(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	}))

	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	s.r.Get("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		// (MVP) sin auth en WS
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			s.log.Warn("ws upgrade failed", zap.Error(err))
			return
		}

		// suscripción al bus
		ch := s.bus.Subscribe()

		// escritor: empuja eventos al cliente
		go func() {
			defer func() {
				s.bus.Unsubscribe(ch)
				_ = conn.Close()
			}()
			for ev := range ch {
				// si el cliente se fue, WriteJSON devolverá error y salimos
				if err := conn.WriteJSON(ev); err != nil {
					s.log.Debug("ws write error", zap.Error(err))
					return
				}
			}
		}()

		// lector mínimo para detectar cierre del cliente (control frames)
		conn.SetReadLimit(1024)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				// cualquier error de lectura implica cierre del lado cliente
				return
			}
		}
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("Authorization")
		if tok == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}
		if len(tok) > 7 && tok[:7] == "Bearer " {
			tok = tok[7:]
		}
		if _, err := s.jwt.Verify(tok); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
