package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evanofslack/go-poker/server"
	"github.com/joho/godotenv"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := godotenv.Load(); err != nil {
		slog.Default().Info("no .env file loaded", "error", err)
	}

	s, err := server.New()
	if err != nil {
		slog.Default().Error("create http server", "error", err)
		os.Exit(1)
	}

	select {
	case err := <-s.Run():
		slog.Default().Error("http server", "error", err)
		os.Exit(1)
	case <-ctx.Done():
	}

	slog.Default().Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		slog.Default().Error("shutdown http server", "error", err)
	}
}
