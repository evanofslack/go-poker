package main

import (
	"context"
	"log/slog"
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
		slog.Default().Error("load env", "error", err)
	}

	s, err := server.New()
	if err != nil {
		slog.Default().Error("create http server", "error", err)
		os.Exit(1)
	}
	if err := s.Run(); err != nil {
		slog.Default().Error("start http server", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Default().Info("Shutting down...")
}
