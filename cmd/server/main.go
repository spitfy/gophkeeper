package main

import (
	"golang.org/x/exp/slog"
	"gophkeeper/internal/config"
	"gophkeeper/internal/storage/postgres"
	"gophkeeper/internal/utils/logger"
	"gophkeeper/internal/utils/logger/sl"
	"os"
)

func main() {
	cfg := config.MustLoad()
	log := logger.New(cfg.Env)

	storage, err := postgres.New(cfg)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	defer storage.Close()

	log.Info(
		"starting gophkeeper",
		slog.String("env", cfg.Env),
		slog.String("version", "1.0"),
	)
}
