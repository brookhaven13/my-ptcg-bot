package engine

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
)

type Game struct {
	State *model.GameState
	rng   *rand.Rand
}

func NewGame(id string, mode model.BattleMode, playerCards, aiCards []model.DeckCard, seed int64) *Game {
	rng := rand.New(rand.NewSource(seed))
	g := &Game{
		State: &model.GameState{
			ID:           id,
			Mode:         mode,
			Phase:        model.PhaseSetup,
			TurnNumber:   0,
			ActivePlayer: "player",
			FirstTurn:    true,
		},
		rng: rng,
	}

	if mode == model.ModePhysical {
		g.State.Player = buildPhysicalPlayerState(playerCards, "p")
	} else {
		g.State.Player = buildPlayerState(playerCards, "p", rng)
	}
	g.State.AI = buildPlayerState(aiCards, "a", rng)

	return g
}

func RestoreGame(stateJSON []byte) (*Game, error) {
	var state model.GameState
	if err := json.Unmarshal(stateJSON, &state); err != nil {
		return nil, fmt.Errorf("unmarshal game state: %w", err)
	}
	return &Game{
		State: &state,
		rng:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func buildPlayerState(deckCards []model.DeckCard, prefix string, rng *rand.Rand) model.PlayerState {
	var instances []model.CardInstance
	uid := 0
	for _, dc := range deckCards {
		for i := 0; i < dc.Quantity; i++ {
			uid++
			instances = append(instances, model.CardInstance{
				UID:    fmt.Sprintf("%s-%d", prefix, uid),
				CardID: dc.CardID,
				Card:   dc.Card,
			})
		}
	}

	rng.Shuffle(len(instances), func(i, j int) {
		instances[i], instances[j] = instances[j], instances[i]
	})

	return model.PlayerState{
		Deck:    instances,
		Hand:    []model.CardInstance{},
		Bench:   []*model.BoardPokemon{},
		Prizes:  []model.CardInstance{},
		Discard: []model.CardInstance{},
	}
}

func buildPhysicalPlayerState(deckCards []model.DeckCard, prefix string) model.PlayerState {
	var instances []model.CardInstance
	uid := 0
	for _, dc := range deckCards {
		for i := 0; i < dc.Quantity; i++ {
			uid++
			instances = append(instances, model.CardInstance{
				UID:    fmt.Sprintf("%s-%d", prefix, uid),
				CardID: dc.CardID,
				Card:   dc.Card,
			})
		}
	}
	// No shuffle — player has physical cards, we just track the pool
	return model.PlayerState{
		Deck:     []model.CardInstance{},
		Hand:     []model.CardInstance{},
		CardPool: instances,
		Bench:    []*model.BoardPokemon{},
		Prizes:   []model.CardInstance{},
		Discard:  []model.CardInstance{},
	}
}

// Setup performs the initial game setup:
// 1. Draw 7 cards each
// 2. Place 6 prize cards each
// 3. Validate both players have at least one Basic Pokemon in hand
// Returns events describing what happened.
func (g *Game) Setup() ([]model.GameEvent, error) {
	if g.State.Phase != model.PhaseSetup {
		return nil, fmt.Errorf("cannot setup: game is in phase %s", g.State.Phase)
	}

	var events []model.GameEvent

	if g.State.Mode == model.ModePhysical {
		// Physical mode: player manually reports hand cards
		// AI side still uses virtual mode
		aiMulligans := 0
		for {
			drawCards(&g.State.AI, 7)
			if hasBasicPokemon(g.State.AI.Hand) {
				break
			}
			aiMulligans++
			returnHandToDeck(&g.State.AI, g.rng)
			if aiMulligans > 10 {
				return nil, fmt.Errorf("AI mulliganed too many times")
			}
		}
		placePrizes(&g.State.AI, 6)
		g.State.Player.PrizesRemaining = 6

		events = append(events, model.GameEvent{
			Type:    "setup_physical",
			Message: "實體牌模式：請洗牌、抽 7 張手牌、放 6 張獎品卡，然後從卡牌池選取你的手牌",
		})
	} else {
		// Virtual mode: engine handles everything
		playerMulligans := 0
		for {
			drawCards(&g.State.Player, 7)
			if hasBasicPokemon(g.State.Player.Hand) {
				break
			}
			playerMulligans++
			events = append(events, model.GameEvent{
				Type:    "mulligan",
				Message: fmt.Sprintf("玩家沒有基本寶可夢，重洗手牌（第 %d 次）", playerMulligans),
			})
			returnHandToDeck(&g.State.Player, g.rng)
			if playerMulligans > 10 {
				return nil, fmt.Errorf("player mulliganed too many times — deck may have no Basic Pokemon")
			}
		}

		aiMulligans := 0
		for {
			drawCards(&g.State.AI, 7)
			if hasBasicPokemon(g.State.AI.Hand) {
				break
			}
			aiMulligans++
			events = append(events, model.GameEvent{
				Type:    "mulligan",
				Message: fmt.Sprintf("AI 沒有基本寶可夢，重洗手牌（第 %d 次）", aiMulligans),
			})
			returnHandToDeck(&g.State.AI, g.rng)
			if aiMulligans > 10 {
				return nil, fmt.Errorf("AI mulliganed too many times — deck may have no Basic Pokemon")
			}
		}

		if playerMulligans > 0 {
			drawCards(&g.State.AI, playerMulligans)
			events = append(events, model.GameEvent{
				Type:    "mulligan_draw",
				Message: fmt.Sprintf("AI 因玩家重洗 %d 次，多抽 %d 張", playerMulligans, playerMulligans),
			})
		}
		if aiMulligans > 0 {
			drawCards(&g.State.Player, aiMulligans)
			events = append(events, model.GameEvent{
				Type:    "mulligan_draw",
				Message: fmt.Sprintf("玩家因 AI 重洗 %d 次，多抽 %d 張", aiMulligans, aiMulligans),
			})
		}

		placePrizes(&g.State.Player, 6)
		placePrizes(&g.State.AI, 6)
		events = append(events, model.GameEvent{
			Type:    "prizes_placed",
			Message: "雙方各放置 6 張獎品卡",
		})
	}

	// Coin flip to decide who goes first
	if g.rng.Intn(2) == 0 {
		g.State.ActivePlayer = "player"
		events = append(events, model.GameEvent{
			Type:    "coin_flip",
			Message: "擲幣結果：玩家先攻",
		})
	} else {
		g.State.ActivePlayer = "ai"
		events = append(events, model.GameEvent{
			Type:    "coin_flip",
			Message: "擲幣結果：AI 先攻",
		})
	}

	g.State.Phase = model.PhaseSetup
	events = append(events, model.GameEvent{
		Type:    "setup_complete",
		Message: "請雙方放置戰鬥寶可夢與後備寶可夢",
	})

	return events, nil
}

// PlaceInitialPokemon places the active and bench Pokemon during setup.
func (g *Game) PlaceInitialPokemon(player string, activeUID string, benchUIDs []string) ([]model.GameEvent, error) {
	ps := g.getPlayerState(player)
	if ps == nil {
		return nil, fmt.Errorf("invalid player: %s", player)
	}

	// Place active Pokemon
	card, idx := findInHand(ps, activeUID)
	if idx < 0 {
		return nil, fmt.Errorf("card %s not in hand", activeUID)
	}
	if !isBasicPokemon(card) {
		return nil, fmt.Errorf("active Pokemon must be a Basic Pokemon")
	}
	ps.Hand = removeFromSlice(ps.Hand, idx)
	ps.Active = &model.BoardPokemon{
		Pokemon:        card,
		AttachedEnergy: []model.CardInstance{},
		Status:         []model.StatusEffect{},
	}

	var events []model.GameEvent
	playerLabel := "玩家"
	if player == "ai" {
		playerLabel = "AI"
	}
	events = append(events, model.GameEvent{
		Type:    "pokemon_placed",
		Message: fmt.Sprintf("%s 放置 %s 為戰鬥寶可夢", playerLabel, cardName(card)),
	})

	// Place bench Pokemon
	for _, uid := range benchUIDs {
		c, i := findInHand(ps, uid)
		if i < 0 {
			return nil, fmt.Errorf("card %s not in hand", uid)
		}
		if !isBasicPokemon(c) {
			return nil, fmt.Errorf("bench Pokemon must be Basic Pokemon")
		}
		if len(ps.Bench) >= 5 {
			return nil, fmt.Errorf("bench is full (max 5)")
		}
		ps.Hand = removeFromSlice(ps.Hand, i)
		ps.Bench = append(ps.Bench, &model.BoardPokemon{
			Pokemon:        c,
			AttachedEnergy: []model.CardInstance{},
			Status:         []model.StatusEffect{},
		})
		events = append(events, model.GameEvent{
			Type:    "pokemon_placed",
			Message: fmt.Sprintf("%s 放置 %s 到後備區", playerLabel, cardName(c)),
		})
	}

	return events, nil
}

// StartBattle transitions from setup to the first turn's draw phase.
func (g *Game) StartBattle() ([]model.GameEvent, error) {
	if g.State.Player.Active == nil || g.State.AI.Active == nil {
		return nil, fmt.Errorf("both players must place an active Pokemon before battle starts")
	}

	g.State.TurnNumber = 1
	g.State.FirstTurn = true
	g.State.Phase = model.PhaseDraw

	return []model.GameEvent{{
		Type:    "battle_start",
		Message: fmt.Sprintf("對戰開始！第 1 回合 — %s 的回合", playerLabel(g.State.ActivePlayer)),
	}}, nil
}

// DrawPhase draws a card for the active player.
// Virtual mode: auto-draw. Physical mode (player's turn): waits for manual_draw action.
func (g *Game) DrawPhase() ([]model.GameEvent, error) {
	if g.State.Phase != model.PhaseDraw {
		return nil, fmt.Errorf("not in draw phase")
	}

	// Physical mode — player side waits for manual draw
	if g.State.Mode == model.ModePhysical && g.State.ActivePlayer == "player" {
		g.State.Phase = model.PhaseMainPhase
		g.State.EnergyAttached = false
		ps := g.getPlayerState("player")
		ps.SupporterPlayed = false
		return []model.GameEvent{{
			Type:    "waiting_for_draw",
			Message: "請從卡牌池選取你抽到的卡片",
		}}, nil
	}

	// Virtual mode (or AI side in physical mode)
	ps := g.getPlayerState(g.State.ActivePlayer)

	if len(ps.Deck) == 0 {
		g.State.Phase = model.PhaseGameOver
		winner := g.getOpponent(g.State.ActivePlayer)
		g.State.Winner = winner
		g.State.WinReason = "opponent_deck_out"
		return []model.GameEvent{{
			Type:    "game_over",
			Message: fmt.Sprintf("%s 無牌可抽，%s 獲勝！", playerLabel(g.State.ActivePlayer), playerLabel(winner)),
		}}, nil
	}

	drawn := ps.Deck[0]
	ps.Deck = ps.Deck[1:]
	ps.Hand = append(ps.Hand, drawn)

	g.State.EnergyAttached = false
	ps.SupporterPlayed = false

	g.State.Phase = model.PhaseMainPhase

	return []model.GameEvent{{
		Type:    "draw_card",
		Message: fmt.Sprintf("%s 抽了一張卡", playerLabel(g.State.ActivePlayer)),
	}}, nil
}

// ManualDraw handles physical mode draw — player picks a card from the pool.
func (g *Game) ManualDraw(cardUID string) ([]model.GameEvent, error) {
	if g.State.Mode != model.ModePhysical {
		return nil, fmt.Errorf("manual draw only available in physical mode")
	}

	ps := g.getPlayerState("player")
	idx := -1
	for i, c := range ps.CardPool {
		if c.UID == cardUID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("card not found in card pool")
	}

	card := ps.CardPool[idx]
	ps.CardPool = append(ps.CardPool[:idx], ps.CardPool[idx+1:]...)
	ps.Hand = append(ps.Hand, card)

	return []model.GameEvent{{
		Type:    "manual_draw",
		Message: fmt.Sprintf("玩家抽到 %s", cardName(card)),
	}}, nil
}

// ManualDrawPrize handles physical mode prize draw — player picks from the pool.
func (g *Game) ManualDrawPrize(cardUID string) ([]model.GameEvent, error) {
	if g.State.Mode != model.ModePhysical {
		return nil, fmt.Errorf("draw prize only available in physical mode")
	}

	ps := g.getPlayerState("player")
	if ps.PrizesRemaining <= 0 {
		return nil, fmt.Errorf("no prizes remaining")
	}

	idx := -1
	for i, c := range ps.CardPool {
		if c.UID == cardUID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("card not found in card pool")
	}

	card := ps.CardPool[idx]
	ps.CardPool = append(ps.CardPool[:idx], ps.CardPool[idx+1:]...)
	ps.Hand = append(ps.Hand, card)
	ps.PrizesRemaining--

	var events []model.GameEvent
	events = append(events, model.GameEvent{
		Type:    "prize_drawn",
		Message: fmt.Sprintf("玩家抽取獎品卡 %s（剩餘 %d 張）", cardName(card), ps.PrizesRemaining),
	})

	if ps.PrizesRemaining == 0 {
		g.State.Phase = model.PhaseGameOver
		g.State.Winner = "player"
		g.State.WinReason = "all_prizes_taken"
		events = append(events, model.GameEvent{
			Type:    "game_over",
			Message: "玩家抽完所有獎品卡，獲勝！",
		})
	}

	return events, nil
}

// GetValidActions returns a list of valid actions the active player can take.
func (g *Game) GetValidActions() []model.ActionType {
	if g.State.Phase != model.PhaseMainPhase {
		return nil
	}

	ps := g.getPlayerState(g.State.ActivePlayer)
	var actions []model.ActionType

	// Can always end turn
	actions = append(actions, model.ActionEndTurn)

	// Play Basic Pokemon to bench
	if len(ps.Bench) < 5 && hasBasicPokemonInHand(*ps) {
		actions = append(actions, model.ActionPlayPokemon)
	}

	// Attach energy (once per turn)
	if !g.State.EnergyAttached && hasEnergyInHand(*ps) {
		actions = append(actions, model.ActionAttachEnergy)
	}

	// Play trainer
	if hasTrainerInHand(*ps) {
		actions = append(actions, model.ActionPlayTrainer)
	}

	// Evolve
	if canEvolveAny(*ps) && !g.State.FirstTurn {
		actions = append(actions, model.ActionEvolvePokemon)
	}

	// Attack (not on first turn of the game for the first player)
	if ps.Active != nil && canAttack(*ps) && !(g.State.FirstTurn && g.State.TurnNumber == 1) {
		actions = append(actions, model.ActionAttack)
	}

	// Retreat
	if ps.Active != nil && len(ps.Bench) > 0 && canRetreat(*ps) {
		actions = append(actions, model.ActionRetreat)
	}

	return actions
}

func (g *Game) getPlayerState(player string) *model.PlayerState {
	if player == "player" {
		return &g.State.Player
	}
	if player == "ai" {
		return &g.State.AI
	}
	return nil
}

func (g *Game) setPlayerState(player string, ps *model.PlayerState) {
	if player == "player" {
		g.State.Player = *ps
	} else if player == "ai" {
		g.State.AI = *ps
	}
}

func (g *Game) getOpponent(player string) string {
	if player == "player" {
		return "ai"
	}
	return "player"
}

// --- Helper functions ---

func drawCards(ps *model.PlayerState, n int) {
	for i := 0; i < n && len(ps.Deck) > 0; i++ {
		ps.Hand = append(ps.Hand, ps.Deck[0])
		ps.Deck = ps.Deck[1:]
	}
}

func placePrizes(ps *model.PlayerState, n int) {
	for i := 0; i < n && len(ps.Deck) > 0; i++ {
		ps.Prizes = append(ps.Prizes, ps.Deck[0])
		ps.Deck = ps.Deck[1:]
	}
}

func returnHandToDeck(ps *model.PlayerState, rng *rand.Rand) {
	ps.Deck = append(ps.Deck, ps.Hand...)
	ps.Hand = nil
	rng.Shuffle(len(ps.Deck), func(i, j int) {
		ps.Deck[i], ps.Deck[j] = ps.Deck[j], ps.Deck[i]
	})
}

func hasBasicPokemon(hand []model.CardInstance) bool {
	for _, c := range hand {
		if isBasicPokemon(c) {
			return true
		}
	}
	return false
}

func isPokemonCategory(cat string) bool {
	return cat == "Pokemon" || cat == "Pokémon" || strings.EqualFold(cat, "pokemon")
}

func isBasicPokemon(c model.CardInstance) bool {
	if c.Card == nil {
		return false
	}
	if !isPokemonCategory(c.Card.Category) {
		return false
	}
	stage := c.Card.Stage
	return stage == "Basic" || stage == "基礎" || stage == "basic" ||
		(stage == "" && c.Card.EvolveFrom == "")
}

func findInHand(ps *model.PlayerState, uid string) (model.CardInstance, int) {
	for i, c := range ps.Hand {
		if c.UID == uid {
			return c, i
		}
	}
	return model.CardInstance{}, -1
}

func removeFromSlice(s []model.CardInstance, idx int) []model.CardInstance {
	return append(s[:idx], s[idx+1:]...)
}

func cardName(c model.CardInstance) string {
	if c.Card != nil {
		return c.Card.Name
	}
	return c.CardID
}

func playerLabel(p string) string {
	if p == "player" {
		return "玩家"
	}
	return "AI"
}

func hasBasicPokemonInHand(ps model.PlayerState) bool {
	for _, c := range ps.Hand {
		if isBasicPokemon(c) {
			return true
		}
	}
	return false
}

func hasEnergyInHand(ps model.PlayerState) bool {
	for _, c := range ps.Hand {
		if c.Card != nil && c.Card.Category == "Energy" {
			return true
		}
	}
	return false
}

func hasTrainerInHand(ps model.PlayerState) bool {
	for _, c := range ps.Hand {
		if c.Card != nil && (c.Card.Category == "Supporter" || c.Card.Category == "Item" || c.Card.Category == "Stadium") {
			return true
		}
	}
	return false
}

func canEvolveAny(ps model.PlayerState) bool {
	evolvables := collectPokemonOnField(ps)
	for _, bp := range evolvables {
		if bp.JustPlayed || bp.JustEvolved {
			continue
		}
		pokeName := cardName(bp.Pokemon)
		for _, c := range ps.Hand {
			if c.Card != nil && c.Card.Category == "Pokemon" && c.Card.EvolveFrom == pokeName {
				return true
			}
		}
	}
	return false
}

func canAttack(ps model.PlayerState) bool {
	if ps.Active == nil || ps.Active.Pokemon.Card == nil {
		return false
	}
	attacks := ps.Active.Pokemon.Card.Attacks
	if len(attacks) == 0 {
		return false
	}
	for _, atk := range attacks {
		if hasEnoughEnergy(ps.Active, atk) {
			return true
		}
	}
	return false
}

func canRetreat(ps model.PlayerState) bool {
	if ps.Active == nil || ps.Active.Pokemon.Card == nil {
		return false
	}
	// Check paralyzed/asleep — can't retreat
	for _, s := range ps.Active.Status {
		if s.Type == "asleep" || s.Type == "paralyzed" {
			return false
		}
	}
	return len(ps.Active.AttachedEnergy) >= ps.Active.Pokemon.Card.Retreat
}

func hasEnoughEnergy(bp *model.BoardPokemon, atk model.Attack) bool {
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

func collectPokemonOnField(ps model.PlayerState) []*model.BoardPokemon {
	var result []*model.BoardPokemon
	if ps.Active != nil {
		result = append(result, ps.Active)
	}
	result = append(result, ps.Bench...)
	return result
}
