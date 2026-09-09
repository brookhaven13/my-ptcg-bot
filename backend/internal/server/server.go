package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/harleylin/ptcg-bot/backend/internal/config"
	"github.com/harleylin/ptcg-bot/backend/internal/store"
	"github.com/harleylin/ptcg-bot/backend/internal/tcgdex"
)

func Run(cfg *config.Config) error {
	s, err := store.New(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	defer s.Close()

	tc := tcgdex.NewClient()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	RegisterRoutes(r, s, tc)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server starting on %s", addr)
	return http.ListenAndServe(addr, r)
}
