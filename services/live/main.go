package main

import (
	"log/slog"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {

	err := godotenv.Load(".env")
	if err != nil {
		slog.Info("failed to load env file", "error", err.Error())
	}

	hub, err := NewHub()
	if err != nil {
		log.Error().Err(err).Msg("failed to create hub")
		return
	}

	http.Handle("/ws", hub)

	go hub.run()

	log.Fatal().Err(http.ListenAndServe(":8000", nil))
}
