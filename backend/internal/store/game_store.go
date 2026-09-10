package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type GameSummary struct {
	ID           string    `json:"id"`
	PlayerDeckID int       `json:"playerDeckId"`
	AIDeckID     int       `json:"aiDeckId"`
	Mode         string    `json:"mode"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	// Joined from state_json or decks
	PlayerDeckName string `json:"playerDeckName,omitempty"`
	AIDeckName     string `json:"aiDeckName,omitempty"`
	TurnNumber     int    `json:"turnNumber,omitempty"`
	Phase          string `json:"phase,omitempty"`
}

type SavedGame struct {
	ID           string    `json:"id"`
	PlayerDeckID int       `json:"playerDeckId"`
	AIDeckID     int       `json:"aiDeckId"`
	Mode         string    `json:"mode"`
	StateJSON    string    `json:"stateJson"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (s *Store) SaveGame(id string, playerDeckID, aiDeckID int, mode, stateJSON, status string) error {
	_, err := s.Pool.Exec(
		context.Background(),
		`INSERT INTO games (id, player_deck_id, ai_deck_id, mode, state_json, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE SET
		   state_json = EXCLUDED.state_json,
		   status = EXCLUDED.status,
		   updated_at = NOW()`,
		id, playerDeckID, aiDeckID, mode, stateJSON, status,
	)
	if err != nil {
		return fmt.Errorf("save game: %w", err)
	}
	return nil
}

func (s *Store) GetGame(id string) (*SavedGame, error) {
	var g SavedGame
	err := s.Pool.QueryRow(
		context.Background(),
		`SELECT id, player_deck_id, ai_deck_id, mode, state_json, status, created_at, updated_at
		 FROM games WHERE id = $1`,
		id,
	).Scan(&g.ID, &g.PlayerDeckID, &g.AIDeckID, &g.Mode, &g.StateJSON, &g.Status, &g.CreatedAt, &g.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get game: %w", err)
	}
	return &g, nil
}

func (s *Store) ListActiveGames() ([]GameSummary, error) {
	rows, err := s.Pool.Query(
		context.Background(),
		`SELECT g.id, g.player_deck_id, g.ai_deck_id, g.mode, g.status, g.created_at, g.updated_at,
		        COALESCE(pd.name, ''), COALESCE(ad.name, '')
		 FROM games g
		 LEFT JOIN decks pd ON pd.id = g.player_deck_id
		 LEFT JOIN decks ad ON ad.id = g.ai_deck_id
		 WHERE g.status = 'in_progress'
		 ORDER BY g.updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list active games: %w", err)
	}
	defer rows.Close()

	var games []GameSummary
	for rows.Next() {
		var g GameSummary
		if err := rows.Scan(&g.ID, &g.PlayerDeckID, &g.AIDeckID, &g.Mode, &g.Status,
			&g.CreatedAt, &g.UpdatedAt, &g.PlayerDeckName, &g.AIDeckName); err != nil {
			return nil, fmt.Errorf("scan game: %w", err)
		}
		games = append(games, g)
	}
	return games, rows.Err()
}

func (s *Store) DeleteGame(id string) error {
	_, err := s.Pool.Exec(context.Background(), "DELETE FROM games WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete game: %w", err)
	}
	return nil
}
