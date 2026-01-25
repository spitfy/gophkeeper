package main

import (
	"context"
	"fmt"
	"gophkeeper/internal/app/server/api"
	"gophkeeper/internal/app/server/config"
	"gophkeeper/internal/infrastructure/migration"
	"gophkeeper/internal/utils/logger"
	"gophkeeper/internal/utils/logger/sl"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/exp/slog"

	"github.com/danielgtaylor/huma/v2/humacli"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.Env)
	pool, err := pgxpool.New(context.Background(), cfg.DB.DatabaseURI)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	defer pool.Close()

	mg := migration.NewMigration(cfg)
	err = mg.Up()
	if err != nil {
		log.Error("failed to run migrations", sl.Err(err))
		os.Exit(1)
	}

	log.Info("starting gophkeeper", slog.String("env", cfg.Env), slog.String("version", "1.0"))

	router := api.New(pool, log)

	cli := humacli.New(func(hooks humacli.Hooks, _ *struct{}) {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Server.RunPort),
			Handler: router,
		}

		hooks.OnStart(func() {
			log.Info("server starting", slog.Int("port", cfg.Server.RunPort))
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
