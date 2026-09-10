package engine

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
)

// attack executes an attack action.
func (g *Game) attack(player string, action model.GameAction) (*model.ActionResult, error) {
	if g.State.Phase != model.PhaseMainPhase {
		return fail("目前無法攻擊"), nil
	}

	if g.State.FirstTurn && g.State.TurnNumber == 1 {
		return fail("遊戲第一回合不能攻擊"), nil
	}

	ps := g.getPlayerState(player)
	if ps.Active == nil || ps.Active.Pokemon.Card == nil {
		return fail("沒有戰鬥寶可夢"), nil
	}

	// Check paralyzed — can't attack
	for _, s := range ps.Active.Status {
		if s.Type == "paralyzed" {
			return fail("寶可夢因麻痺無法攻擊"), nil
		}
	}

	// Check asleep — can't attack
	for _, s := range ps.Active.Status {
		if s.Type == "asleep" {
			return fail("寶可夢正在睡眠，無法攻擊"), nil
		}
	}

	attacks := ps.Active.Pokemon.Card.Attacks
	if action.AttackIndex < 0 || action.AttackIndex >= len(attacks) {
		return fail("無效的攻擊招式"), nil
	}

	atk := attacks[action.AttackIndex]
	if !hasEnoughEnergy(ps.Active, atk) {
		return fail("能量不足，無法使用此招式"), nil
	}

	opponent := g.getOpponent(player)
	ops := g.getPlayerState(opponent)
	if ops.Active == nil {
		return fail("對手沒有戰鬥寶可夢"), nil
	}

	var events []model.GameEvent

	// Check confused — flip coin, if tails deal 30 to self
	isConfused := false
	for _, s := range ps.Active.Status {
		if s.Type == "confused" {
			isConfused = true
			break
		}
	}

	if isConfused {
		coinFlip := g.rng.Intn(2)
		if coinFlip == 0 {
			events = append(events, model.GameEvent{
				Type:    "confusion_check",
				Message: fmt.Sprintf("%s 的 %s 混亂中，擲幣結果：正面，可以攻擊", playerLabel(player), cardName(ps.Active.Pokemon)),
			})
		} else {
			ps.Active.DamageCounters += 3 // 30 damage = 3 counters
			events = append(events, model.GameEvent{
				Type:    "confusion_self_damage",
				Message: fmt.Sprintf("%s 的 %s 混亂中，擲幣結果：反面，對自己造成 30 傷害", playerLabel(player), cardName(ps.Active.Pokemon)),
			})

			// Check self-KO
			koEvents := g.checkKO(player, ps, ps.Active)
			events = append(events, koEvents...)

			g.State.Phase = model.PhaseBetweenTurns
			btEvents := g.processBetweenTurns()
			events = append(events, btEvents...)

			if g.State.Phase != model.PhaseGameOver {
				g.advanceTurn(player)
				events = append(events, model.GameEvent{
					Type:    "turn_end",
					Message: fmt.Sprintf("第 %d 回合 — %s 的回合", g.State.TurnNumber, playerLabel(g.State.ActivePlayer)),
				})
			}

			return &model.ActionResult{Success: true, State: g.State, Events: events}, nil
		}
	}

	// Calculate damage
	baseDamage := parseDamage(atk.Damage)
	damage := baseDamage

	// Weakness: ×2
	if ops.Active.Pokemon.Card != nil {
		for _, w := range ops.Active.Pokemon.Card.Weaknesses {
			attackerTypes := ps.Active.Pokemon.Card.Types
			for _, t := range attackerTypes {
				if strings.EqualFold(t, w.Type) {
					damage *= 2
					events = append(events, model.GameEvent{
						Type:    "weakness",
						Message: fmt.Sprintf("弱點！傷害 ×2"),
					})
				}
			}
		}

		// Resistance: -30
		for _, r := range ops.Active.Pokemon.Card.Resistances {
			attackerTypes := ps.Active.Pokemon.Card.Types
			for _, t := range attackerTypes {
				if strings.EqualFold(t, r.Type) {
					reduction := parseResistance(r.Value)
					damage -= reduction
					if damage < 0 {
						damage = 0
					}
					events = append(events, model.GameEvent{
						Type:    "resistance",
						Message: fmt.Sprintf("抗性！傷害 -%d", reduction),
					})
				}
			}
		}
	}

	events = append(events, model.GameEvent{
		Type:    "attack",
		Message: fmt.Sprintf("%s 的 %s 使用 %s，造成 %d 傷害", playerLabel(player), cardName(ps.Active.Pokemon), atk.Name, damage),
	})

	// Apply damage
	damageCounters := damage / 10
	ops.Active.DamageCounters += damageCounters

	// Check KO
	koEvents := g.checkKO(opponent, ops, ops.Active)
	events = append(events, koEvents...)

	if g.State.Phase == model.PhaseGameOver {
		return &model.ActionResult{Success: true, State: g.State, Events: events}, nil
	}

	// Move to between turns
	g.State.Phase = model.PhaseBetweenTurns
	btEvents := g.processBetweenTurns()
	events = append(events, btEvents...)

	if g.State.Phase != model.PhaseGameOver {
		g.advanceTurn(player)
		events = append(events, model.GameEvent{
			Type:    "turn_end",
			Message: fmt.Sprintf("第 %d 回合 — %s 的回合", g.State.TurnNumber, playerLabel(g.State.ActivePlayer)),
		})
	}

	return &model.ActionResult{Success: true, State: g.State, Events: events}, nil
}

// checkKO checks if the active Pokemon is knocked out and handles prizes/promotion.
func (g *Game) checkKO(defender string, ps *model.PlayerState, target *model.BoardPokemon) []model.GameEvent {
	if target.Pokemon.Card == nil || target != ps.Active {
		return nil
	}

	hp := target.Pokemon.Card.HP
	totalDamage := target.DamageCounters * 10
	if totalDamage < hp {
		return nil
	}

	var events []model.GameEvent
	events = append(events, model.GameEvent{
		Type:    "knockout",
		Message: fmt.Sprintf("%s 的 %s 被擊倒！", playerLabel(defender), cardName(target.Pokemon)),
	})

	// Move KO'd Pokemon + attached energy to discard
	ps.Discard = append(ps.Discard, target.Pokemon)
	for _, e := range target.AttachedEnergy {
		ps.Discard = append(ps.Discard, e)
	}

	// Attacker draws a prize
	attacker := g.getOpponent(defender)
	aps := g.getPlayerState(attacker)
	if len(aps.Prizes) > 0 {
		prize := aps.Prizes[0]
		aps.Prizes = aps.Prizes[1:]
		aps.Hand = append(aps.Hand, prize)
		events = append(events, model.GameEvent{
			Type:    "prize_drawn",
			Message: fmt.Sprintf("%s 抽取一張獎品卡（剩餘 %d 張）", playerLabel(attacker), len(aps.Prizes)),
		})

		// Win by prizes
		if len(aps.Prizes) == 0 {
			g.State.Phase = model.PhaseGameOver
			g.State.Winner = attacker
			g.State.WinReason = "all_prizes_taken"
			events = append(events, model.GameEvent{
				Type:    "game_over",
				Message: fmt.Sprintf("%s 抽完所有獎品卡，獲勝！", playerLabel(attacker)),
			})
			return events
		}
	}

	// Promote from bench
	if len(ps.Bench) > 0 {
		ps.Active = ps.Bench[0]
		ps.Bench = ps.Bench[1:]
		events = append(events, model.GameEvent{
			Type:    "promote",
			Message: fmt.Sprintf("%s 的 %s 從後備區上場", playerLabel(defender), cardName(ps.Active.Pokemon)),
		})
	} else {
		ps.Active = nil
		g.State.Phase = model.PhaseGameOver
		g.State.Winner = attacker
		g.State.WinReason = "no_pokemon_left"
		events = append(events, model.GameEvent{
			Type:    "game_over",
			Message: fmt.Sprintf("%s 沒有後備寶可夢，%s 獲勝！", playerLabel(defender), playerLabel(attacker)),
		})
	}

	return events
}

// processBetweenTurns handles status effects at end of turn.
func (g *Game) processBetweenTurns() []model.GameEvent {
	var events []model.GameEvent

	for _, player := range []string{"player", "ai"} {
		ps := g.getPlayerState(player)
		if ps.Active == nil {
			continue
		}

		var newStatus []model.StatusEffect
		for _, s := range ps.Active.Status {
			switch s.Type {
			case "poisoned":
				ps.Active.DamageCounters += 1 // 10 damage
				events = append(events, model.GameEvent{
					Type:    "poison_damage",
					Message: fmt.Sprintf("%s 的 %s 中毒，受到 10 傷害", playerLabel(player), cardName(ps.Active.Pokemon)),
				})
				koEvents := g.checkKO(player, ps, ps.Active)
				events = append(events, koEvents...)
				if g.State.Phase == model.PhaseGameOver {
					return events
				}
				newStatus = append(newStatus, s) // poison persists

			case "burned":
				// Flip coin: heads = no damage, tails = 20 damage
				if g.rng.Intn(2) == 1 {
					ps.Active.DamageCounters += 2 // 20 damage
					events = append(events, model.GameEvent{
						Type:    "burn_damage",
						Message: fmt.Sprintf("%s 的 %s 灼傷，擲幣反面，受到 20 傷害", playerLabel(player), cardName(ps.Active.Pokemon)),
					})
					koEvents := g.checkKO(player, ps, ps.Active)
					events = append(events, koEvents...)
					if g.State.Phase == model.PhaseGameOver {
						return events
					}
				} else {
					events = append(events, model.GameEvent{
						Type:    "burn_check",
						Message: fmt.Sprintf("%s 的 %s 灼傷，擲幣正面，不受傷害", playerLabel(player), cardName(ps.Active.Pokemon)),
					})
				}
				newStatus = append(newStatus, s) // burn persists

			case "asleep":
				// Flip coin: heads = wake up
				if g.rng.Intn(2) == 0 {
					events = append(events, model.GameEvent{
						Type:    "sleep_check",
						Message: fmt.Sprintf("%s 的 %s 擲幣正面，醒來了！", playerLabel(player), cardName(ps.Active.Pokemon)),
					})
					// Don't add back to newStatus
				} else {
					events = append(events, model.GameEvent{
						Type:    "sleep_check",
						Message: fmt.Sprintf("%s 的 %s 擲幣反面，繼續睡眠", playerLabel(player), cardName(ps.Active.Pokemon)),
					})
					newStatus = append(newStatus, s)
				}

			case "paralyzed":
				// Paralysis wears off at end of the paralyzed player's turn
				events = append(events, model.GameEvent{
					Type:    "paralysis_end",
					Message: fmt.Sprintf("%s 的 %s 麻痺解除", playerLabel(player), cardName(ps.Active.Pokemon)),
				})
				// Don't add back

			case "confused":
				newStatus = append(newStatus, s) // confusion persists

			default:
				newStatus = append(newStatus, s)
			}
		}
		if ps.Active != nil {
			ps.Active.Status = newStatus
		}
	}

	return events
}

func (g *Game) advanceTurn(currentPlayer string) {
	g.State.ActivePlayer = g.getOpponent(currentPlayer)
	if g.State.ActivePlayer == "player" {
		g.State.TurnNumber++
	}
	if g.State.TurnNumber > 2 {
		g.State.FirstTurn = false
	}
	g.State.EnergyAttached = false
	g.resetTurnFlags(g.State.ActivePlayer)
	g.State.Phase = model.PhaseDraw
}

// --- Damage parsing helpers ---

func parseDamage(v any) int {
	if v == nil {
		return 0
	}
	switch d := v.(type) {
	case float64:
		return int(d)
	case int:
		return d
	case string:
		s := strings.TrimRight(d, "+×x-?")
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0
		}
		return n
	}
	return 0
}

func parseResistance(v string) int {
	s := strings.TrimLeft(v, "-")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 30 // default
	}
	return n
}
