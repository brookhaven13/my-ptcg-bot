package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{Pool: pool}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS cards (
			id          TEXT PRIMARY KEY,
			local_id    TEXT NOT NULL,
			set_id      TEXT NOT NULL,
			name        TEXT NOT NULL,
			category    TEXT NOT NULL,
			data_json   TEXT NOT NULL,
			image_url   TEXT NOT NULL,
			created_at  TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS decks (
			id          SERIAL PRIMARY KEY,
			name        TEXT NOT NULL,
			owner       TEXT NOT NULL,
			raw_list    TEXT NOT NULL,
			created_at  TIMESTAMPTZ DEFAULT NOW(),
			updated_at  TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS deck_cards (
			id          SERIAL PRIMARY KEY,
			deck_id     INTEGER NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
			card_id     TEXT NOT NULL REFERENCES cards(id),
			quantity    INTEGER NOT NULL,
			UNIQUE(deck_id, card_id)
		)`,
		`CREATE TABLE IF NOT EXISTS games (
			id              TEXT PRIMARY KEY,
			player_deck_id  INTEGER REFERENCES decks(id),
			ai_deck_id      INTEGER REFERENCES decks(id),
			mode            TEXT NOT NULL,
			state_json      TEXT NOT NULL,
			status          TEXT NOT NULL,
			created_at      TIMESTAMPTZ DEFAULT NOW(),
			updated_at      TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, m := range migrations {
		if _, err := s.Pool.Exec(context.Background(), m); err != nil {
			return fmt.Errorf("execute migration: %w", err)
		}
	}
	return nil
}

func (s *Store) Close() {
	s.Pool.Close()
}
