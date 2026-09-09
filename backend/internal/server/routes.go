package server

import (
	"github.com/go-chi/chi/v5"

	"github.com/harleylin/ptcg-bot/backend/internal/handler"
	"github.com/harleylin/ptcg-bot/backend/internal/store"
	"github.com/harleylin/ptcg-bot/backend/internal/tcgdex"
)

func RegisterRoutes(r chi.Router, s *store.Store, tc *tcgdex.Client) {
	cardHandler := handler.NewCardHandler(s, tc)
	deckHandler := handler.NewDeckHandler(s, tc)

	r.Route("/api", func(r chi.Router) {
		r.Get("/cards/{id}", cardHandler.GetCard)
		r.Patch("/cards/image", cardHandler.PatchCardImage)
		r.Post("/cards/lookup", cardHandler.LookupCards)

		r.Post("/decks", deckHandler.CreateDeck)
		r.Get("/decks", deckHandler.ListDecks)
		r.Get("/decks/{id}", deckHandler.GetDeck)
		r.Put("/decks/{id}", deckHandler.UpdateDeck)
		r.Delete("/decks/{id}", deckHandler.DeleteDeck)
	})
}
