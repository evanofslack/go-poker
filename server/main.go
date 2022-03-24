package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func InitSocket() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	s := newServer()
	s.mountMiddleware()
	s.mountSocket()

	fmt.Println("listening...")
	http.ListenAndServe(s.port, s.router)
}
