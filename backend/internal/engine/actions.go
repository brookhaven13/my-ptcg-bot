package engine

import (
	"fmt"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
)

// ExecuteAction validates and executes a game action.
func (g *Game) ExecuteAction(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase == model.PhaseGameOver {
		return &model.ActionResult{Success: false, Message: "遊戲已結束"}, nil
	}

	if player != g.State.ActivePlayer && action.Type != model.ActionPlayPokemon {
		return &model.ActionResult{Success: false, Message: "不是你的回合"}, nil
	}

	switch action.Type {
	case model.ActionPlayPokemon:
		return g.playPokemon(player, action)
	case model.ActionAttachEnergy:
		return g.attachEnergy(player, action)
	case model.ActionEvolvePokemon:
		return g.evolvePokemon(player, action)
	case model.ActionPlayTrainer:
		return g.playTrainer(player, action)
	case model.ActionRetreat:
		return g.retreat(player, action)
	case model.ActionAttack:
		return g.attack(player, action)
	case model.ActionEndTurn:
		return g.endTurn(player)
	case model.ActionManualDraw:
		return g.manualDrawAction(player, action)
	case model.ActionDrawPrize:
		return g.drawPrizeAction(player, action)
	default:
		return &model.ActionResult{Success: false, Message: fmt.Sprintf("unknown action: %s", action.Type)}, nil
	}
}

func (g *Game) playPokemon(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase && g.State.Phase != model.PhaseSetup {
		return fail("目前無法放置寶可夢"), nil
	}

	ps := g.getPlayerState(player)
	card, idx := findInHand(ps, action.CardUID)
	if idx < 0 {
		return fail("手牌中找不到此卡"), nil
	}

	if !isBasicPokemon(card) {
		return fail("只能放置基本寶可夢"), nil
	}

	if len(ps.Bench) >= 5 {
		return fail("後備區已滿（最多 5 隻）"), nil
	}

	ps.Hand = removeFromSlice(ps.Hand, idx)
	bp := &model.BoardPokemon{
		Pokemon:        card,
		AttachedEnergy: []model.CardInstance{},
		Status:         []model.StatusEffect{},
		JustPlayed:     true,
	}
	ps.Bench = append(ps.Bench, bp)

	return &model.ActionResult{
		Success: true,
		State:   g.State,
		Events: []model.GameEvent{{
			Type:    "pokemon_played",
			Message: fmt.Sprintf("%s 將 %s 放到後備區", playerLabel(player), cardName(card)),
		}},
	}, nil
}

func (g *Game) attachEnergy(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法貼能量"), nil
	}

	if g.State.EnergyAttached {
		return fail("每回合只能貼一次能量"), nil
	}

	ps := g.getPlayerState(player)
	card, idx := findInHand(ps, action.CardUID)
	if idx < 0 {
		return fail("手牌中找不到此能量卡"), nil
	}

	if card.Card == nil || card.Card.Category != "Energy" {
		return fail("這不是能量卡"), nil
	}

	target := findPokemonOnField(ps, action.TargetUID)
	if target == nil {
		return fail("找不到目標寶可夢"), nil
	}

	ps.Hand = removeFromSlice(ps.Hand, idx)
	target.AttachedEnergy = append(target.AttachedEnergy, card)
	g.State.EnergyAttached = true

	return &model.ActionResult{
		Success: true,
		State:   g.State,
		Events: []model.GameEvent{{
			Type:    "energy_attached",
			Message: fmt.Sprintf("%s 將 %s 貼到 %s", playerLabel(player), cardName(card), cardName(target.Pokemon)),
		}},
	}, nil
}

func (g *Game) evolvePokemon(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法進化"), nil
	}

	if g.State.FirstTurn && g.State.TurnNumber == 1 {
		return fail("第一回合不能進化"), nil
	}

	ps := g.getPlayerState(player)
	evoCard, idx := findInHand(ps, action.CardUID)
	if idx < 0 {
		return fail("手牌中找不到此卡"), nil
	}

	if evoCard.Card == nil || evoCard.Card.Category != "Pokemon" {
		return fail("這不是寶可夢卡"), nil
	}

	target := findPokemonOnField(ps, action.TargetUID)
	if target == nil {
		return fail("找不到目標寶可夢"), nil
	}

	if target.JustPlayed || target.JustEvolved {
		return fail("此寶可夢本回合無法進化"), nil
	}

	targetName := cardName(target.Pokemon)
	if evoCard.Card.EvolveFrom != targetName {
		return fail(fmt.Sprintf("%s 無法從 %s 進化", cardName(evoCard), targetName)), nil
	}

	ps.Hand = removeFromSlice(ps.Hand, idx)

	oldPokemon := target.Pokemon
	target.Pokemon = evoCard
	target.EvolvedFrom = &model.BoardPokemon{Pokemon: oldPokemon}
	target.JustEvolved = true
	// Evolution removes all status effects
	target.Status = []model.StatusEffect{}

	return &model.ActionResult{
		Success: true,
		State:   g.State,
		Events: []model.GameEvent{{
			Type:    "pokemon_evolved",
			Message: fmt.Sprintf("%s 的 %s 進化為 %s", playerLabel(player), cardName(oldPokemon), cardName(evoCard)),
		}},
	}, nil
}

func (g *Game) playTrainer(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法使用訓練家卡"), nil
	}

	ps := g.getPlayerState(player)
	card, idx := findInHand(ps, action.CardUID)
	if idx < 0 {
		return fail("手牌中找不到此卡"), nil
	}

	if card.Card == nil {
		return fail("卡片資料不足"), nil
	}

	switch card.Card.Category {
	case "Supporter":
		if ps.SupporterPlayed {
			return fail("每回合只能使用一張支援者卡"), nil
		}
		ps.Hand = removeFromSlice(ps.Hand, idx)
		ps.Discard = append(ps.Discard, card)
		ps.SupporterPlayed = true

		events := g.resolveTrainerEffect(player, card)
		return &model.ActionResult{
			Success: true,
			State:   g.State,
			Events:  events,
		}, nil

	case "Item":
		ps.Hand = removeFromSlice(ps.Hand, idx)
		ps.Discard = append(ps.Discard, card)

		events := g.resolveTrainerEffect(player, card)
		return &model.ActionResult{
			Success: true,
			State:   g.State,
			Events:  events,
		}, nil

	case "Stadium":
		ps.Hand = removeFromSlice(ps.Hand, idx)
		// TODO: stadium stays in play, not discarded
		ps.Discard = append(ps.Discard, card)

		events := g.resolveTrainerEffect(player, card)
		return &model.ActionResult{
			Success: true,
			State:   g.State,
			Events:  events,
		}, nil

	default:
		return fail("這不是訓練家卡"), nil
	}
}

func (g *Game) resolveTrainerEffect(player string, card model.CardInstance) []model.GameEvent {
	// Generic trainer resolution — specific card effects can be added later
	return []model.GameEvent{{
		Type:    "trainer_played",
		Message: fmt.Sprintf("%s 使用了 %s", playerLabel(player), cardName(card)),
	}}
}

func (g *Game) retreat(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法撤退"), nil
	}

	ps := g.getPlayerState(player)
	if ps.Active == nil {
		return fail("沒有戰鬥寶可夢"), nil
	}

	// Check status
	for _, s := range ps.Active.Status {
		if s.Type == "asleep" || s.Type == "paralyzed" {
			return fail("寶可夢因狀態異常無法撤退"), nil
		}
	}

	retreatCost := 0
	if ps.Active.Pokemon.Card != nil {
		retreatCost = ps.Active.Pokemon.Card.Retreat
	}

	if len(ps.Active.AttachedEnergy) < retreatCost {
		return fail(fmt.Sprintf("能量不足，需要 %d 個能量才能撤退", retreatCost)), nil
	}

	// Find new active from bench
	newActiveIdx := -1
	for i, bp := range ps.Bench {
		if bp.Pokemon.UID == action.TargetUID {
			newActiveIdx = i
			break
		}
	}
	if newActiveIdx < 0 {
		return fail("後備區找不到指定的寶可夢"), nil
	}

	// Discard energy for retreat cost
	for i := 0; i < retreatCost && len(ps.Active.AttachedEnergy) > 0; i++ {
		discarded := ps.Active.AttachedEnergy[0]
		ps.Active.AttachedEnergy = ps.Active.AttachedEnergy[1:]
		ps.Discard = append(ps.Discard, discarded)
	}

	// Swap
	oldActive := ps.Active
	oldActiveName := cardName(oldActive.Pokemon)
	ps.Active.Status = []model.StatusEffect{} // retreat removes status
	ps.Active = ps.Bench[newActiveIdx]
	ps.Bench = append(ps.Bench[:newActiveIdx], ps.Bench[newActiveIdx+1:]...)
	ps.Bench = append(ps.Bench, oldActive)

	return &model.ActionResult{
		Success: true,
		State:   g.State,
		Events: []model.GameEvent{{
			Type:    "retreat",
			Message: fmt.Sprintf("%s 的 %s 撤退，%s 上場", playerLabel(player), oldActiveName, cardName(ps.Active.Pokemon)),
		}},
	}, nil
}

func (g *Game) endTurn(player string) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法結束回合"), nil
	}

	g.State.Phase = model.PhaseBetweenTurns
	events := g.processBetweenTurns()

	if g.State.Phase == model.PhaseGameOver {
		return &model.ActionResult{
			Success: true,
			State:   g.State,
			Events:  events,
		}, nil
	}

	// Switch active player
	g.State.ActivePlayer = g.getOpponent(player)
	if g.State.ActivePlayer == "player" && g.State.TurnNumber > 1 {
		g.State.FirstTurn = false
	}
	if g.State.ActivePlayer == "player" {
		g.State.TurnNumber++
	}

	// Reset per-turn flags
	g.State.EnergyAttached = false
	g.resetTurnFlags(g.State.ActivePlayer)

	g.State.Phase = model.PhaseDraw

	events = append(events, model.GameEvent{
		Type:    "turn_end",
		Message: fmt.Sprintf("第 %d 回合 — %s 的回合", g.State.TurnNumber, playerLabel(g.State.ActivePlayer)),
	})

	return &model.ActionResult{
		Success: true,
		State:   g.State,
		Events:  events,
	}, nil
}

func (g *Game) resetTurnFlags(player string) {
	ps := g.getPlayerState(player)
	ps.SupporterPlayed = false

	if ps.Active != nil {
		ps.Active.JustPlayed = false
		ps.Active.JustEvolved = false
	}
	for _, bp := range ps.Bench {
		bp.JustPlayed = false
		bp.JustEvolved = false
	}
}

// --- Action helpers ---

func findPokemonOnField(ps *model.PlayerState, uid string) *model.BoardPokemon {
	if ps.Active != nil && ps.Active.Pokemon.UID == uid {
		return ps.Active
	}
	for _, bp := range ps.Bench {
		if bp.Pokemon.UID == uid {
			return bp
		}
	}
	return nil
}

func (g *Game) manualDrawAction(player string, action model.GameAction) (*model.ActionResult, error) {
	if player != "player" {
		return fail("only player can manual draw"), nil
	}
	events, err := g.ManualDraw(action.CardUID)
	if err != nil {
		return fail(err.Error()), nil
	}
	return &model.ActionResult{Success: true, State: g.State, Events: events}, nil
}

func (g *Game) drawPrizeAction(player string, action model.GameAction) (*model.ActionResult, error) {
	if player != "player" {
		return fail("only player can draw prize"), nil
	}
	events, err := g.ManualDrawPrize(action.CardUID)
	if err != nil {
		return fail(err.Error()), nil
	}
	return &model.ActionResult{Success: true, State: g.State, Events: events}, nil
}

func fail(msg string) *model.ActionResult {
	return &model.ActionResult{Success: false, Message: msg}
}
