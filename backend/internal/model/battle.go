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
	ID           string      `json:"id"`
	Mode         BattleMode  `json:"mode"`
	Phase        GamePhase   `json:"phase"`
	TurnNumber   int         `json:"turnNumber"`
	ActivePlayer string      `json:"activePlayer"`
	Player       PlayerState `json:"player"`
	AI           PlayerState `json:"ai"`
	Winner       string      `json:"winner,omitempty"`
	WinReason    string      `json:"winReason,omitempty"`
}

type PlayerState struct {
	Deck    []CardInstance  `json:"deck"`
	Hand    []CardInstance  `json:"hand"`
	Active  *BoardPokemon   `json:"active"`
	Bench   []*BoardPokemon `json:"bench"`
	Prizes  []CardInstance  `json:"prizes"`
	Discard []CardInstance  `json:"discard"`
}

type CardInstance struct {
	UID    string      `json:"uid"`
	CardID string      `json:"cardId"`
	Card   *CardDetail `json:"card,omitempty"`
}

type BoardPokemon struct {
	Pokemon        CardInstance   `json:"pokemon"`
	AttachedEnergy []CardInstance `json:"attachedEnergy"`
	DamageCounters int            `json:"damageCounters"`
	Status         []StatusEffect `json:"status"`
	EvolvedFrom    *BoardPokemon  `json:"evolvedFrom,omitempty"`
}

type StatusEffect struct {
	Type string `json:"type"`
}
