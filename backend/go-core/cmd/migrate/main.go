package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/config"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config validation failed")
	}

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	log.Info().Msg("running database migrations...")
	if err := migrations.Run(context.Background(), db); err != nil {
		log.Fatal().Err(err).Msg("database migrations failed")
	}
	log.Info().Msg("database migrations completed successfully")
}
