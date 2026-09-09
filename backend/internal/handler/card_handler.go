package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
	"github.com/harleylin/ptcg-bot/backend/internal/store"
	"github.com/harleylin/ptcg-bot/backend/internal/tcgdex"
)

type CardHandler struct {
	store  *store.Store
	tcgdex *tcgdex.Client
}

func NewCardHandler(s *store.Store, tc *tcgdex.Client) *CardHandler {
	return &CardHandler{store: s, tcgdex: tc}
}

func (h *CardHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	cached, err := h.store.GetCard(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cached != nil {
		var detail model.CardDetail
		json.Unmarshal([]byte(cached.DataJSON), &detail)
		detail.ImageURL = cached.ImageURL
		writeJSON(w, detail)
		return
	}

	http.Error(w, "card not found in cache", http.StatusNotFound)
}

type LookupRequest struct {
	Cards []LookupEntry `json:"cards"`
}

type LookupEntry struct {
	SetCode string `json:"setCode"`
	Number  string `json:"number"`
}

func (h *CardHandler) LookupCards(w http.ResponseWriter, r *http.Request) {
	var req LookupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	results := make(map[string]*model.CardDetail)

	for _, entry := range req.Cards {
		cardID := entry.SetCode + "-" + entry.Number
		if _, ok := results[cardID]; ok {
			continue
		}

		cached, _ := h.store.GetCard(cardID)
		if cached != nil {
			var detail model.CardDetail
			json.Unmarshal([]byte(cached.DataJSON), &detail)
			detail.ImageURL = cached.ImageURL
			results[cardID] = &detail
			continue
		}

		card, raw, err := h.tcgdex.GetCardRaw(entry.SetCode, entry.Number)
		if err != nil {
			continue
		}

		localNum, _ := strconv.Atoi(entry.Number)
		imageURL := tcgdex.GetImageURL(entry.SetCode, localNum)

		c := &model.Card{
			ID:       card.ID,
			LocalID:  card.LocalID,
			SetID:    entry.SetCode,
			Name:     card.Name,
			Category: card.Category,
			DataJSON: string(raw),
			ImageURL: imageURL,
		}
		h.store.SaveCard(c)

		var detail model.CardDetail
		json.Unmarshal(raw, &detail)
		detail.ImageURL = imageURL
		results[cardID] = &detail
	}

	writeJSON(w, results)
}

type PatchCardImageRequest struct {
	ID       string `json:"id"`
	ImageURL string `json:"imageUrl"`
}

func (h *CardHandler) PatchCardImage(w http.ResponseWriter, r *http.Request) {
	var req PatchCardImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ID == "" {
		http.Error(w, "card id is required", http.StatusBadRequest)
		return
	}

	cached, _ := h.store.GetCard(req.ID)
	if cached == nil {
		http.Error(w, "card not found", http.StatusNotFound)
		return
	}

	cached.ImageURL = req.ImageURL
	if err := h.store.SaveCard(cached); err != nil {
		http.Error(w, "failed to save card", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]string{"id": req.ID, "imageUrl": req.ImageURL})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
