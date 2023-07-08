package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)


type Server struct {
	listener net.Listener
	server  *http.Server
	router *chi.Mux
	hub    *Hub
}

func New() *Server {

	port := getPort()
	addr := ":" + port

	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		fmt.Println("Failed to create listener")
	}

	s := &Server{
		listener: ln,
		server: &http.Server{Addr: addr},
		router: chi.NewRouter(),
		hub:    newHub(),
	}

	s.server.Handler = s.router
	s.mountMiddleware()
	s.mountSocket()
	s.mountStatus()

	return s
}

func (s *Server) Run() error {
	go s.hub.run()
	fmt.Println("running websockets hub")
	go s.server.Serve(s.listener)
	fmt.Println("http server listening...")
	return nil
}



func getPort() string {
	var port = os.Getenv("PORT")
	if port == "" {
		port = "8080"
		fmt.Println("No PORT env variable found, defaulting to: " + port)
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
		log.Fatal(err)
	}
}

func (s *Server) mountSocket() {
	s.router.Route("/ws", func(r chi.Router) { r.Get("/", s.serveWebsocket) })
}

func (s *Server) serveWebsocket(w http.ResponseWriter, r *http.Request) {
	serveWs(s.hub, w, r)
}
