package main

import (
	"log"

	"github.com/harleylin/ptcg-bot/backend/internal/config"
	"github.com/harleylin/ptcg-bot/backend/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
