package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/harleylin/ptcg-bot/backend/internal/engine"
	"github.com/harleylin/ptcg-bot/backend/internal/model"
	"github.com/harleylin/ptcg-bot/backend/internal/store"
	"github.com/harleylin/ptcg-bot/backend/internal/ws"
)

type BattleHandler struct {
	store *store.Store
	hub   *ws.Hub
	mu    sync.RWMutex
	games map[string]*engine.Game
}

func NewBattleHandler(s *store.Store) *BattleHandler {
	bh := &BattleHandler{
		store: s,
		games: make(map[string]*engine.Game),
	}
	bh.hub = ws.NewHub(bh.handleWSMessage)
	return bh
}

func (bh *BattleHandler) Hub() *ws.Hub {
	return bh.hub
}

type StartBattleRequest struct {
	PlayerDeckID int              `json:"playerDeckId"`
	AIDeckID     int              `json:"aiDeckId"`
	Mode         model.BattleMode `json:"mode"`
}

type StartBattleResponse struct {
	GameID string           `json:"gameId"`
	State  *model.GameState `json:"state"`
	Events []model.GameEvent `json:"events"`
}

func (bh *BattleHandler) StartBattle(w http.ResponseWriter, r *http.Request) {
	var req StartBattleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Mode == "" {
		req.Mode = model.ModeVirtual
	}

	playerDeck, err := bh.loadDeckWithCards(req.PlayerDeckID)
	if err != nil {
		http.Error(w, "failed to load player deck: "+err.Error(), http.StatusBadRequest)
		return
	}
	aiDeck, err := bh.loadDeckWithCards(req.AIDeckID)
	if err != nil {
		http.Error(w, "failed to load AI deck: "+err.Error(), http.StatusBadRequest)
		return
	}

	gameID := fmt.Sprintf("game-%d", time.Now().UnixNano())
	game := engine.NewGame(gameID, req.Mode, playerDeck.Cards, aiDeck.Cards, time.Now().UnixNano())
	game.State.PlayerDeckID = req.PlayerDeckID
	game.State.AIDeckID = req.AIDeckID

	events, err := game.Setup()
	if err != nil {
		http.Error(w, "failed to setup game: "+err.Error(), http.StatusInternalServerError)
		return
	}

	bh.mu.Lock()
	bh.games[gameID] = game
	bh.mu.Unlock()

	bh.saveGameState(game)
	log.Printf("[battle] game %s started (mode=%s)", gameID, req.Mode)

	resp := StartBattleResponse{
		GameID: gameID,
		State:  game.State,
		Events: events,
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

func (bh *BattleHandler) ResumeGame(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameId")

	// Check if already loaded in memory
	bh.mu.RLock()
	game, ok := bh.games[gameID]
	bh.mu.RUnlock()

	if ok {
		writeJSON(w, StartBattleResponse{
			GameID: gameID,
			State:  game.State,
		})
		return
	}

	// Load from DB
	saved, err := bh.store.GetGame(gameID)
	if err != nil {
		http.Error(w, "failed to load game: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if saved == nil {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}
	if saved.Status != "in_progress" {
		http.Error(w, "game is already finished", http.StatusBadRequest)
		return
	}

	game, err = engine.RestoreGame([]byte(saved.StateJSON))
	if err != nil {
		http.Error(w, "failed to restore game state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	bh.mu.Lock()
	bh.games[gameID] = game
	bh.mu.Unlock()

	log.Printf("[battle] game %s resumed (turn=%d, phase=%s)", gameID, game.State.TurnNumber, game.State.Phase)

	writeJSON(w, StartBattleResponse{
		GameID: gameID,
		State:  game.State,
	})
}

func (bh *BattleHandler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameId")

	bh.mu.Lock()
	delete(bh.games, gameID)
	bh.mu.Unlock()

	if err := bh.store.DeleteGame(gameID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (bh *BattleHandler) GetGameState(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameId")

	bh.mu.RLock()
	game, ok := bh.games[gameID]
	bh.mu.RUnlock()

	if !ok {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}

	writeJSON(w, game.State)
}

func (bh *BattleHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameId")

	bh.mu.RLock()
	_, ok := bh.games[gameID]
	bh.mu.RUnlock()

	if !ok {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}

	bh.hub.HandleWebSocket(w, r, gameID)
}

func (bh *BattleHandler) handleWSMessage(gameID string, player string, msg ws.IncomingMessage) {
	bh.mu.Lock()
	game, ok := bh.games[gameID]
	if !ok {
		bh.mu.Unlock()
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": "game not found"},
		})
		return
	}
	bh.mu.Unlock()

	switch msg.Type {
	case "sync":
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type: "state_update",
			Payload: map[string]any{
				"state":        game.State,
				"validActions": game.GetValidActions(),
			},
		})
		return
	case "place_pokemon":
		bh.handlePlacePokemon(game, gameID, player, msg)
	case "start_battle":
		bh.handleStartBattle(game, gameID)
	case "draw":
		bh.handleDraw(game, gameID)
	case "action":
		bh.handleAction(game, gameID, player, msg)
	default:
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": "unknown message type: " + msg.Type},
		})
	}
}

func (bh *BattleHandler) handlePlacePokemon(game *engine.Game, gameID, player string, msg ws.IncomingMessage) {
	events, err := game.PlaceInitialPokemon(player, msg.ActiveUID, msg.BenchUIDs)
	if err != nil {
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": err.Error()},
		})
		return
	}

	// AI auto-place: pick first basic Pokemon as active, rest to bench
	if game.State.AI.Active == nil {
		bh.autoPlaceAI(game, gameID)
	}

	bh.saveGameState(game)
	bh.hub.SendToGame(gameID, ws.OutgoingMessage{
		Type: "state_update",
		Payload: map[string]any{
			"state":  game.State,
			"events": events,
		},
	})
}

func (bh *BattleHandler) autoPlaceAI(game *engine.Game, gameID string) {
	ai := &game.State.AI
	var activeUID string
	var benchUIDs []string

	for _, c := range ai.Hand {
		if isBasicPokemonCard(c) {
			if activeUID == "" {
				activeUID = c.UID
			} else if len(benchUIDs) < 5 {
				benchUIDs = append(benchUIDs, c.UID)
			}
		}
	}

	if activeUID != "" {
		events, err := game.PlaceInitialPokemon("ai", activeUID, benchUIDs)
		if err != nil {
			log.Printf("[battle] AI place error: %v", err)
			return
		}
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type: "ai_action",
			Payload: map[string]any{
				"description": "AI 放置了寶可夢",
				"events":      events,
			},
		})
	}
}

func (bh *BattleHandler) handleStartBattle(game *engine.Game, gameID string) {
	events, err := game.StartBattle()
	if err != nil {
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": err.Error()},
		})
		return
	}

	bh.saveGameState(game)
	bh.hub.SendToGame(gameID, ws.OutgoingMessage{
		Type: "state_update",
		Payload: map[string]any{
			"state":  game.State,
			"events": events,
		},
	})

	// Auto draw phase
	bh.handleDraw(game, gameID)
}

func (bh *BattleHandler) handleDraw(game *engine.Game, gameID string) {
	if game.State.Phase != model.PhaseDraw {
		return
	}

	events, err := game.DrawPhase()
	if err != nil {
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": err.Error()},
		})
		return
	}

	bh.saveGameState(game)
	bh.hub.SendToGame(gameID, ws.OutgoingMessage{
		Type: "state_update",
		Payload: map[string]any{
			"state":        game.State,
			"events":       events,
			"validActions": game.GetValidActions(),
		},
	})

	// If AI's turn, trigger AI actions
	if game.State.ActivePlayer == "ai" && game.State.Phase == model.PhaseMainPhase {
		bh.runAITurn(game, gameID)
	}
}

func (bh *BattleHandler) handleAction(game *engine.Game, gameID, player string, msg ws.IncomingMessage) {
	action := model.GameAction{
		Type:        model.ActionType(msg.Action),
		CardUID:     msg.CardUID,
		TargetUID:   msg.TargetUID,
		AttackIndex: msg.AttackIndex,
		Position:    msg.Position,
		BenchIndex:  msg.BenchIndex,
	}

	result, err := game.ExecuteAction("player", action)
	if err != nil {
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": err.Error()},
		})
		return
	}

	if !result.Success {
		bh.hub.SendToGame(gameID, ws.OutgoingMessage{
			Type:    "error",
			Payload: map[string]string{"message": result.Message},
		})
		return
	}

	bh.saveGameState(game)
	bh.hub.SendToGame(gameID, ws.OutgoingMessage{
		Type: "state_update",
		Payload: map[string]any{
			"state":        game.State,
			"events":       result.Events,
			"validActions": game.GetValidActions(),
		},
	})

	// After player ends turn or attacks, check if it's now AI's draw phase
	if game.State.ActivePlayer == "ai" && game.State.Phase == model.PhaseDraw {
		bh.handleDraw(game, gameID)
	}
}

// runAITurn performs a simple AI turn (placeholder until Gemini integration in Phase 7)
func (bh *BattleHandler) runAITurn(game *engine.Game, gameID string) {
	ai := &game.State.AI
	var allEvents []model.GameEvent

	// Simple AI: attach energy if possible, then attack if possible, else end turn
	// Attach energy
	if !game.State.EnergyAttached && ai.Active != nil {
		for _, c := range ai.Hand {
			if c.Card != nil && c.Card.Category == "Energy" {
				result, _ := game.ExecuteAction("ai", model.GameAction{
					Type:      model.ActionAttachEnergy,
					CardUID:   c.UID,
					TargetUID: ai.Active.Pokemon.UID,
				})
				if result != nil && result.Success {
					allEvents = append(allEvents, result.Events...)
				}
				break
			}
		}
	}

	// Play basic Pokemon to bench
	for _, c := range ai.Hand {
		if isBasicPokemonCard(c) && len(ai.Bench) < 5 {
			result, _ := game.ExecuteAction("ai", model.GameAction{
				Type:    model.ActionPlayPokemon,
				CardUID: c.UID,
			})
			if result != nil && result.Success {
				allEvents = append(allEvents, result.Events...)
			}
		}
	}

	// Try to attack
	attacked := false
	if ai.Active != nil && ai.Active.Pokemon.Card != nil {
		attacks := ai.Active.Pokemon.Card.Attacks
		for i, atk := range attacks {
			if hasEnoughEnergyForAttack(ai.Active, atk) {
				result, _ := game.ExecuteAction("ai", model.GameAction{
					Type:        model.ActionAttack,
					AttackIndex: i,
				})
				if result != nil && result.Success {
					allEvents = append(allEvents, result.Events...)
					attacked = true
					break
				}
			}
		}
	}

	if !attacked {
		result, _ := game.ExecuteAction("ai", model.GameAction{Type: model.ActionEndTurn})
		if result != nil && result.Success {
			allEvents = append(allEvents, result.Events...)
		}
	}

	bh.saveGameState(game)
	bh.hub.SendToGame(gameID, ws.OutgoingMessage{
		Type: "ai_action",
		Payload: map[string]any{
			"state":        game.State,
			"events":       allEvents,
			"validActions": game.GetValidActions(),
		},
	})

	// If game continues and it's player's draw phase
	if game.State.Phase == model.PhaseDraw && game.State.ActivePlayer == "player" {
		bh.handleDraw(game, gameID)
	}
}

func (bh *BattleHandler) loadDeckWithCards(deckID int) (*model.Deck, error) {
	deck, err := bh.store.GetDeck(deckID)
	if err != nil {
		return nil, err
	}

	for i, dc := range deck.Cards {
		cached, _ := bh.store.GetCard(dc.CardID)
		if cached == nil {
			log.Printf("[battle] WARNING: card %s not found in DB", dc.CardID)
			continue
		}

		var detail model.CardDetail
		json.Unmarshal([]byte(cached.DataJSON), &detail)
		detail.ImageURL = cached.ImageURL
		detail.Name = cached.Name
		detail.ID = cached.ID
		if detail.Category == "" && cached.Category != "" {
			detail.Category = cached.Category
		}
		detail.Category = normalizeCategory(detail.Category)
		deck.Cards[i].Card = &detail

		log.Printf("[battle] loaded card %s: name=%s category=%s stage=%s",
			dc.CardID, detail.Name, detail.Category, detail.Stage)
	}

	return deck, nil
}

func isBasicPokemonCard(c model.CardInstance) bool {
	if c.Card == nil {
		return false
	}
	isPokemon := c.Card.Category == "Pokemon" || c.Card.Category == "Pokémon"
	isBasic := c.Card.Stage == "Basic" || c.Card.Stage == "基礎" ||
		(c.Card.Stage == "" && c.Card.EvolveFrom == "")
	return isPokemon && isBasic
}

func hasEnoughEnergyForAttack(bp *model.BoardPokemon, atk model.Attack) bool {
	if len(atk.Cost) == 0 {
		return true
	}
	available := map[string]int{}
	for _, e := range bp.AttachedEnergy {
		if e.Card != nil && len(e.Card.Types) > 0 {
			available[e.Card.Types[0]]++
		} else {
			available["Colorless"]++
		}
	}
	needed := map[string]int{}
	for _, c := range atk.Cost {
		needed[c]++
	}
	totalAvailable := 0
	for _, v := range available {
		totalAvailable += v
	}
	colorlessNeeded := needed["Colorless"]
	delete(needed, "Colorless")
	for t, n := range needed {
		if available[t] < n {
			return false
		}
		totalAvailable -= n
	}
	return totalAvailable >= colorlessNeeded
}

func (bh *BattleHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	games, err := bh.store.ListActiveGames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if games == nil {
		games = []store.GameSummary{}
	}
	writeJSON(w, games)
}

func (bh *BattleHandler) saveGameState(game *engine.Game) {
	stateJSON, err := json.Marshal(game.State)
	if err != nil {
		log.Printf("[battle] failed to marshal game state: %v", err)
		return
	}

	status := "in_progress"
	if game.State.Phase == model.PhaseGameOver {
		if game.State.Winner == "player" {
			status = "player_won"
		} else {
			status = "ai_won"
		}
	}

	if err := bh.store.SaveGame(
		game.State.ID,
		game.State.PlayerDeckID,
		game.State.AIDeckID,
		string(game.State.Mode),
		string(stateJSON),
		status,
	); err != nil {
		log.Printf("[battle] failed to save game %s: %v", game.State.ID, err)
	}
}
