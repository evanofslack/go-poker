package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// default location of the built frontend in the docker image
const defaultStaticDir = "/app/out"

type Server struct {
	listener  net.Listener
	server    *http.Server
	router    *chi.Mux
	hub       *Hub
	staticDir string
	ready     atomic.Bool // false once shutdown starts
	stop      context.CancelFunc

	// serves /metrics on its own address, nil when metrics are off
	metrics         *http.Server
	metricsListener net.Listener

	// serves the admin page on its own address, nil when it's off
	admin         *http.Server
	adminListener net.Listener
}

func New() (*Server, error) {
	loadConfig()
	port := getPort()
	addr := ":" + port

	registerExtensions()

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	store, err := openStore(context.Background(), databaseURL)
	if err != nil {
		ln.Close()
		return nil, err
	}
	hub := newHub(store)
	if err := hub.restore(context.Background()); err != nil {
		ln.Close()
		return nil, err
	}
	slog.Default().Info("restored tables", "store", fmt.Sprintf("%T", store), "count", len(hub.allTables()))

	s := &Server{
		listener: ln,
		server: &http.Server{
			Addr:              addr,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		},
		router:    chi.NewRouter(),
		hub:       hub,
		staticDir: getStaticDir(),
	}

	s.ready.Store(true)
	s.server.Handler = s.router
	s.mountMiddleware()
	s.mountSocket()
	s.mountStatus()
	s.mountStatic()

	if metricsAddr != "" {
		if s.metricsListener, err = net.Listen("tcp4", metricsAddr); err != nil {
			ln.Close()
			return nil, err
		}
		mux := http.NewServeMux()
		mux.Handle("/metrics", metricsHandler(hub))
		s.metrics = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	}

	if adminAddr != "" {
		if s.adminListener, err = net.Listen("tcp4", adminAddr); err != nil {
			ln.Close()
			if s.metricsListener != nil {
				s.metricsListener.Close()
			}
			return nil, err
		}
		s.admin = &http.Server{Handler: adminRouter(hub, s.staticDir), ReadHeaderTimeout: 10 * time.Second}
	}

	return s, nil
}

// Run starts the hub and http server, and returns a channel that receives
// the error if the server stops unexpectedly.
func (s *Server) Run() <-chan error {
	go s.hub.run()
	slog.Default().Info("running websocket hub")
	ctx, stop := context.WithCancel(context.Background())
	s.stop = stop
	go s.hub.prune(ctx)

	errc := make(chan error, 3)
	go func() {
		if err := s.server.Serve(s.listener); !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()
	slog.Default().Info("starting http server")
	if s.metrics != nil {
		go func() {
			if err := s.metrics.Serve(s.metricsListener); !errors.Is(err, http.ErrServerClosed) {
				errc <- err
			}
		}()
		slog.Default().Info("serving metrics", "addr", metricsAddr)
	}
	if s.admin != nil {
		go func() {
			if err := s.admin.Serve(s.adminListener); !errors.Is(err, http.ErrServerClosed) {
				errc <- err
			}
		}()
		slog.Default().Info("serving admin", "addr", adminAddr)
	}
	return errc
}

// Shutdown fails readiness, tells clients the server is restarting and closes
// their websockets, then stops accepting connections and waits for in-flight
// requests.
func (s *Server) Shutdown(ctx context.Context) error {
	s.ready.Store(false)
	if s.stop != nil {
		s.stop()
	}
	s.hub.shutdown(ctx)
	if c, ok := s.hub.store.(io.Closer); ok {
		c.Close()
	}
	if s.metrics != nil {
		s.metrics.Shutdown(ctx)
	}
	if s.admin != nil {
		s.admin.Shutdown(ctx)
	}
	return s.server.Shutdown(ctx)
}

func getStaticDir() string {
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		return dir
	}
	return defaultStaticDir
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
}

func (s *Server) mountStatus() {
	s.router.Route("/ping", func(r chi.Router) { r.Get("/", s.ping) })
	s.router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, http.StatusOK, "ok")
	})
	s.router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if s.ready.Load() {
			writeStatus(w, http.StatusOK, "ok")
		} else {
			writeStatus(w, http.StatusServiceUnavailable, "shutting down")
		}
	})
}

func writeStatus(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": status}); err != nil {
		slog.Default().Error("encode status", "error", err)
	}
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
	s.router.Handle("/*", staticHandler(s.staticDir, true))
}

// staticHandler serves the static site in dir. hideAdmin leaves out the admin
// page, which is only on the admin listener.
func staticHandler(dir string, hideAdmin bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		// Skip API and WebSocket routes
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/ws/") ||
			hideAdmin && (p == "/admin" || p == "/admin.html" || strings.HasPrefix(p, "/admin/")) {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(dir, p)
		if _, err := os.Stat(path); err != nil {
			// a page, like /admin for admin.html
			path += ".html"
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			ext := filepath.Ext(path)
			if mimeType := mime.TypeByExtension(ext); mimeType != "" {
				w.Header().Set("Content-Type", mimeType)
			}
			http.ServeFile(w, r, path)
			return
		}

		// If file not found, serve index.html for client-side routing
		indexPath := filepath.Join(dir, "index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFile(w, r, indexPath)
	})
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
	mime.AddExtensionType(".mp3", "audio/mpeg")
}
