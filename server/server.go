package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func InitServer() {
	s := newServer()
	s.mountMiddleware()
	s.mountSocket()

	fmt.Println("listening...")
	http.ListenAndServe(s.port, s.router)
}

type server struct {
	router *chi.Mux
	hub    *Hub
	port   string
}

func newServer() *server {
	s := &server{
		router: chi.NewRouter(),
		hub:    newHub(),
		port:   getPort(),
	}
	go s.hub.run()
	return s
}

func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
	}
	return ":" + port
}

func (s *server) mountMiddleware() {

	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*", "http://localhost"},
		AllowedMethods:   []string{"PUT, GET, POST, DELETE, OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           500,
	}))
}

func (s *server) mountSocket() {
	s.router.Route("/ws", func(r chi.Router) { r.Get("/", s.serveWebsocket) })
}

func (s *server) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	serveWs(s.hub, w, r)
}
