package model

type GamePhase string

const (
	PhaseSetup        GamePhase = "SETUP"
	PhaseDraw         GamePhase = "DRAW"
	PhaseMainPhase    GamePhase = "MAIN_PHASE"
	PhaseAttack       GamePhase = "ATTACK"
	PhaseBetweenTurns GamePhase = "BETWEEN_TURNS"
	PhaseGameOver     GamePhase = "GAME_OVER"
)

type BattleMode string

const (
	ModeVirtual  BattleMode = "virtual"
	ModePhysical BattleMode = "physical"
)

type GameStatus string

const (
	StatusInProgress GameStatus = "in_progress"
	StatusPlayerWon  GameStatus = "player_won"
	StatusAIWon      GameStatus = "ai_won"
)

type GameState struct {
	ID             string      `json:"id"`
	Mode           BattleMode  `json:"mode"`
	Phase          GamePhase   `json:"phase"`
	TurnNumber     int         `json:"turnNumber"`
	ActivePlayer   string      `json:"activePlayer"` // "player" or "ai"
	Player         PlayerState `json:"player"`
	AI             PlayerState `json:"ai"`
	Winner         string      `json:"winner,omitempty"`
	WinReason      string      `json:"winReason,omitempty"`
	FirstTurn      bool        `json:"firstTurn"`
	EnergyAttached bool        `json:"energyAttached"` // per-turn flag
	PlayerDeckID   int         `json:"playerDeckId,omitempty"`
	AIDeckID       int         `json:"aiDeckId,omitempty"`
}

type PlayerState struct {
	Deck            []CardInstance  `json:"deck"`
	Hand            []CardInstance  `json:"hand"`
	Active          *BoardPokemon   `json:"active"`
	Bench           []*BoardPokemon `json:"bench"`
	Prizes          []CardInstance  `json:"prizes"`
	Discard         []CardInstance  `json:"discard"`
	CardPool        []CardInstance  `json:"cardPool,omitempty"` // physical mode: all undrawn cards
	PrizesRemaining int             `json:"prizesRemaining"`    // physical mode: how many prizes left (don't know which)
	SupporterPlayed bool            `json:"supporterPlayed"`
}

type CardInstance struct {
	UID    string      `json:"uid"`
	CardID string      `json:"cardId"`
	Card   *CardDetail `json:"card,omitempty"`
}

type BoardPokemon struct {
	Pokemon        CardInstance   `json:"pokemon"`
	AttachedEnergy []CardInstance `json:"attachedEnergy"`
	DamageCounters int            `json:"damageCounters"` // each counter = 10 HP damage
	Status         []StatusEffect `json:"status"`
	EvolvedFrom    *BoardPokemon  `json:"evolvedFrom,omitempty"`
	JustEvolved    bool           `json:"justEvolved,omitempty"`    // can't evolve again this turn
	JustPlayed     bool           `json:"justPlayed,omitempty"`     // can't evolve the turn it was played
}

type StatusEffect struct {
	Type string `json:"type"` // "poisoned", "burned", "asleep", "paralyzed", "confused"
}

type ActionType string

const (
	ActionPlayPokemon   ActionType = "play_pokemon"
	ActionAttachEnergy  ActionType = "attach_energy"
	ActionPlayTrainer   ActionType = "play_trainer"
	ActionUseAbility    ActionType = "use_ability"
	ActionRetreat       ActionType = "retreat"
	ActionAttack        ActionType = "attack"
	ActionEndTurn       ActionType = "end_turn"
	ActionEvolvePokemon ActionType = "evolve_pokemon"
	ActionManualDraw    ActionType = "manual_draw"  // physical mode: player reports drawn card
	ActionDrawPrize     ActionType = "draw_prize"   // physical mode: player reports prize card drawn
)

type GameAction struct {
	Type        ActionType `json:"type"`
	CardUID     string     `json:"cardUid,omitempty"`
	TargetUID   string     `json:"targetUid,omitempty"`
	AttackIndex int        `json:"attackIndex,omitempty"`
	Position    string     `json:"position,omitempty"` // "active" or "bench"
	BenchIndex  int        `json:"benchIndex,omitempty"`
}

type ActionResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	State   *GameState  `json:"state,omitempty"`
	Events  []GameEvent `json:"events,omitempty"`
}

type GameEvent struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
