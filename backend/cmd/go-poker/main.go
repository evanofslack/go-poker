package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/evanofslack/go-poker/server"
	"github.com/joho/godotenv"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	s := server.New()
	if err := s.Run(); err != nil {
		err = fmt.Errorf("Failed to start http server: %w", err)
		log.Println(err)
		os.Exit(1)
	}

	<-ctx.Done()
	log.Println("Got shutdown signal, starting graceful shutdown")
}
