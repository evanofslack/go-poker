package server

import (
	"encoding/json"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const (
	staticDir = "/app/out"
)

type Server struct {
	listener net.Listener
	server   *http.Server
	router   *chi.Mux
	hub      *Hub
}

func New() (*Server, error) {
	port := getPort()
	addr := ":" + port

	registerExtensions()

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	hub, err := newHub()
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: ln,
		server:   &http.Server{Addr: addr},
		router:   chi.NewRouter(),
		hub:      hub,
	}

	s.server.Handler = s.router
	s.mountMiddleware()
	s.mountSocket()
	s.mountStatus()
	s.mountStatic()

	return s, nil
}

func (s *Server) Run() error {
	go s.hub.run()
	slog.Default().Info("running websocket hub")
	go s.server.Serve(s.listener)
	slog.Default().Info("starting http server")
	return nil
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		slog.Default().Info("No port env variable, using default", "port", port)
	}
	return port
}

func (s *Server) mountMiddleware() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*", "http://localhost", "ws://*"},
		AllowedMethods:   []string{"PUT, GET, POST, DELETE, OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           500,
	}))
}

func (s *Server) mountStatus() {
	s.router.Route("/ping", func(r chi.Router) { r.Get("/", s.ping) })
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	if err := enc.Encode("message: pong"); err != nil {
		slog.Default().Error("encode ping", "error", err)
	}
}

func (s *Server) mountSocket() {
	s.router.Route("/ws", func(r chi.Router) { r.Get("/", s.serveWebsocket) })
}

func (s *Server) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	serveWs(s.hub, w, r)
}

func (s *Server) mountStatic() {
	s.router.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API and WebSocket routes
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/ws/") {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(staticDir, r.URL.Path)
		_, err := os.Stat(path)
		if err == nil {
			ext := filepath.Ext(path)
			if mimeType := mime.TypeByExtension(ext); mimeType != "" {
				w.Header().Set("Content-Type", mimeType)
			}
			http.ServeFile(w, r, path)
			return
		}

		// If file not found, serve index.html for client-side routing
		indexPath := filepath.Join(staticDir, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, indexPath)
	}))
}

func registerExtensions() {
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".map", "application/json")
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".ttf", "font/ttf")
}
