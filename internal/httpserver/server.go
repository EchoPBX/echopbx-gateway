package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/EchoPBX/echopbx-gateway/internal/config"
	"github.com/EchoPBX/echopbx-gateway/internal/events"
	"github.com/EchoPBX/echopbx-gateway/internal/jwt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Server struct {
	cfg *config.Config
	log *zap.Logger
	bus *events.Bus
	r   *chi.Mux
	jwt *jwt.Validator
}

func New(cfg *config.Config, log *zap.Logger, bus *events.Bus) *Server {
	v, _ := jwt.NewValidator(cfg.Auth.JWTPublicKeys, cfg.Auth.Issuer, cfg.Auth.Audience)
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}}))
	s := &Server{cfg: cfg, log: log, bus: bus, r: r, jwt: v}
	s.routes()
	return s
}

func (s *Server) Router() http.Handler      { return s.r }
func (s *Server) Reload(cfg *config.Config) { s.cfg = cfg }

func (s *Server) routes() {
	s.r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); w.Write([]byte("ok")) })
	s.r.Get("/v1/info", s.auth(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"name": "echopbx-gateway", "time": time.Now().UTC()}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	s.r.Get("/v1/calls", s.auth(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	}))

	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	s.r.Get("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		// auth por query o header (MVP: opcional)
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ch := s.bus.Subscribe()
		go func() {
			for ev := range ch {
				_ = conn.WriteJSON(ev)
			}
			conn.Close()
		}()
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("Authorization")
		if tok == "" {
			http.Error(w, "missing token", 401)
			return
		}
		if len(tok) > 7 && tok[:7] == "Bearer " {
			tok = tok[7:]
		}
		if _, err := s.jwt.Verify(tok); err != nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		next(w, r)
	}
}
