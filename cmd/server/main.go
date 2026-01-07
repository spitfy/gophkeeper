package main

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"gophkeeper/internal/app/server/api"
	"gophkeeper/internal/app/server/config"
	"gophkeeper/internal/infrastructure/storage/postgres"
	"gophkeeper/internal/utils/logger"
	"gophkeeper/internal/utils/logger/sl"
	"net/http"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2/humacli"
)

// Options for the CLI. Pass `--port` or set the `SERVICE_PORT` env var.
type Options struct {
	Port int `help:"Port to listen on" short:"p" default:"8888"`
}

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.Env)

	storage, err := postgres.New(cfg)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	defer storage.Close()

	log.Info("starting gophkeeper", slog.String("env", cfg.Env), slog.String("version", "1.0"))

	router := api.New(storage, log)

	cli := humacli.New(func(hooks humacli.Hooks, options *Options) {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", options.Port),
			Handler: router,
		}

		hooks.OnStart(func() {
			log.Info("server starting", slog.Int("port", options.Port))
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Error("server failed", sl.Err(err))
			}
		})

		hooks.OnStop(func() {
			log.Info("shutting down server gracefully")
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			if err := server.Shutdown(ctx); err != nil {
				log.Error("server forced shutdown", sl.Err(err))
			}
			log.Info("server stopped")
		})
	})

	cli.Run()
}
