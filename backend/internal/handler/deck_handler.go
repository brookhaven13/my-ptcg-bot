package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
	"github.com/harleylin/ptcg-bot/backend/internal/store"
	"github.com/harleylin/ptcg-bot/backend/internal/tcgdex"
)

type DeckHandler struct {
	store  *store.Store
	tcgdex *tcgdex.Client
}

func NewDeckHandler(s *store.Store, tc *tcgdex.Client) *DeckHandler {
	return &DeckHandler{store: s, tcgdex: tc}
}

type CreateDeckRequest struct {
	Name    string `json:"name"`
	Owner   string `json:"owner"`
	RawList string `json:"rawList"`
}

func (h *DeckHandler) CreateDeck(w http.ResponseWriter, r *http.Request) {
	var req CreateDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Owner != "player" && req.Owner != "ai" {
		http.Error(w, "owner must be 'player' or 'ai'", http.StatusBadRequest)
		return
	}

	entries, err := parseDeckList(req.RawList)
	if err != nil {
		http.Error(w, "failed to parse deck list: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("[CreateDeck] parsed %d entries from rawList (%d bytes)", len(entries), len(req.RawList))

	deck := &model.Deck{
		Name:    req.Name,
		Owner:   req.Owner,
		RawList: req.RawList,
	}

	deck.Cards = h.resolveCards(entries)

	if err := h.store.CreateDeck(deck); err != nil {
		http.Error(w, "failed to create deck: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, deck)
}

func (h *DeckHandler) UpdateDeck(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	var req CreateDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entries, err := parseDeckList(req.RawList)
	if err != nil {
		http.Error(w, "failed to parse deck list: "+err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("[UpdateDeck] parsed %d entries from rawList (%d bytes)", len(entries), len(req.RawList))

	deck := &model.Deck{
		ID:      id,
		Name:    req.Name,
		RawList: req.RawList,
	}

	deck.Cards = h.resolveCards(entries)

	if err := h.store.UpdateDeck(deck); err != nil {
		http.Error(w, "failed to update deck: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, deck)
}

func (h *DeckHandler) resolveCards(entries []deckEntry) []model.DeckCard {
	var cards []model.DeckCard
	for _, entry := range entries {
		cardID := entry.SetCode + "-" + entry.Number

		cached, _ := h.store.GetCard(cardID)
		if cached != nil && cached.SetID == entry.SetCode && cached.LocalID == entry.Number && cached.Name == entry.Name {
			cards = append(cards, model.DeckCard{CardID: cardID, Quantity: entry.Quantity})
			continue
		}

		imageURL := ""
		if entry.LocalNum > 0 {
			imageURL = tcgdex.GetImageURL(entry.SetCode, entry.LocalNum)
		}

		category := entry.Category
		dataJSON := "{}"

		tcgCard, raw, err := h.tcgdex.SearchCardByName(entry.Name)
		if err == nil && tcgCard != nil {
			if tcgCard.Category != "" {
				category = tcgCard.Category
			}
			dataJSON = string(raw)

			if imageURL == "" && tcgCard.Image != "" {
				imageURL = tcgCard.Image + "/high.png"
			}
		}

		c := &model.Card{
			ID:       cardID,
			LocalID:  entry.Number,
			SetID:    entry.SetCode,
			Name:     entry.Name,
			Category: category,
			DataJSON: dataJSON,
			ImageURL: imageURL,
		}
		h.store.SaveCard(c)

		cards = append(cards, model.DeckCard{
			CardID:   cardID,
			Quantity: entry.Quantity,
		})
	}
	return cards
}

func (h *DeckHandler) ListDecks(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		owner = "player"
	}

	decks, err := h.store.ListDecks(owner)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if decks == nil {
		decks = []model.Deck{}
	}

	writeJSON(w, decks)
}

func (h *DeckHandler) GetDeck(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	deck, err := h.store.GetDeck(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, dc := range deck.Cards {
		cached, _ := h.store.GetCard(dc.CardID)
		if cached != nil {
			var detail model.CardDetail
			json.Unmarshal([]byte(cached.DataJSON), &detail)
			detail.ImageURL = cached.ImageURL
			detail.Name = cached.Name
			detail.ID = cached.ID
			deck.Cards[i].Card = &detail
		}
	}

	writeJSON(w, deck)
}

func (h *DeckHandler) DeleteDeck(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid deck id", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteDeck(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type deckEntry struct {
	Quantity int
	Name     string
	SetCode  string
	Number   string
	LocalNum int
	Category string // Pokemon, Energy, Supporter, Item, Stadium
}

var sectionCategoryMap = map[string]string{
	"寶可夢卡": "Pokemon",
	"能量卡":  "Energy",
	"支援者卡": "Supporter",
	"物品卡":  "Item",
	"競技場卡": "Stadium",
}

// parseDeckList 解析分組牌組格式
// 寶可夢卡：名稱\t擴充標記\t編號/總數\t數量
// 其他類別：名稱\t數量
func parseDeckList(raw string) ([]deckEntry, error) {
	var entries []deckEntry
	seen := make(map[string]int)
	currentCategory := "Pokemon"

	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 偵測分組標頭 [寶可夢卡]、[能量卡] 等
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := line[len("[") : len(line)-len("]")]
			if cat, ok := sectionCategoryMap[sectionName]; ok {
				currentCategory = cat
			}
			continue
		}

		// 跳過標題行（相容舊格式）
		if strings.Contains(line, "張數") || strings.Contains(line, "收集編號") {
			continue
		}

		fields := strings.Split(line, "\t")

		switch currentCategory {
		case "Pokemon":
			entry := parsePokemonLine(fields)
			if entry != nil {
				entry.Category = "Pokemon"
				addEntry(&entries, seen, *entry)
			} else if simple := parseSimpleLine(fields, "Pokemon"); simple != nil {
				addEntry(&entries, seen, *simple)
			} else {
				log.Printf("[deck-parser] skipped line in Pokemon section: %q", line)
			}

		default:
			// 非寶可夢區也支援 4 欄格式（直接從 TW 網站貼上）
			if entry := parsePokemonLine(fields); entry != nil {
				entry.Category = currentCategory
				addEntry(&entries, seen, *entry)
			} else if entry := parseSimpleLine(fields, currentCategory); entry != nil {
				addEntry(&entries, seen, *entry)
			} else {
				log.Printf("[deck-parser] skipped line in %s section: %q", currentCategory, line)
			}
		}
	}

	return entries, nil
}

// parsePokemonLine 解析寶可夢卡：名稱\t擴充標記\t編號/總數\t數量
func parsePokemonLine(fields []string) *deckEntry {
	if len(fields) < 4 {
		fields = strings.Fields(strings.Join(fields, " "))
		if len(fields) < 4 {
			return nil
		}
	}

	name := strings.TrimSpace(fields[0])
	setCode := strings.TrimSpace(fields[1])
	numberField := strings.TrimSpace(fields[2])
	qtyField := strings.TrimSpace(fields[3])

	qty, err := strconv.Atoi(qtyField)
	if err != nil {
		return nil
	}

	number := numberField
	localNum := 0
	if parts := strings.SplitN(numberField, "/", 2); len(parts) >= 1 {
		number = parts[0]
		localNum, _ = strconv.Atoi(parts[0])
	}

	return &deckEntry{
		Quantity: qty,
		Name:     name,
		SetCode:  setCode,
		Number:   number,
		LocalNum: localNum,
	}
}

// parseSimpleLine 解析能量/訓練家卡：名稱\t數量
func parseSimpleLine(fields []string, category string) *deckEntry {
	if len(fields) < 2 {
		fields = strings.Fields(strings.Join(fields, " "))
		if len(fields) < 2 {
			return nil
		}
	}

	name := strings.TrimSpace(fields[0])
	qtyField := strings.TrimSpace(fields[len(fields)-1])

	qty, err := strconv.Atoi(qtyField)
	if err != nil {
		return nil
	}

	return &deckEntry{
		Quantity: qty,
		Name:     name,
		SetCode:  category,
		Number:   name,
		Category: category,
	}
}

func addEntry(entries *[]deckEntry, seen map[string]int, e deckEntry) {
	key := e.SetCode + "-" + e.Number
	if idx, ok := seen[key]; ok {
		(*entries)[idx].Quantity += e.Quantity
	} else {
		seen[key] = len(*entries)
		*entries = append(*entries, e)
	}
}
