package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
)

func (s *Store) GetCard(id string) (*model.Card, error) {
	var card model.Card
	err := s.Pool.QueryRow(
		context.Background(),
		"SELECT id, local_id, set_id, name, category, data_json, image_url, created_at FROM cards WHERE id = $1",
		id,
	).Scan(&card.ID, &card.LocalID, &card.SetID, &card.Name, &card.Category, &card.DataJSON, &card.ImageURL, &card.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query card: %w", err)
	}
	return &card, nil
}

func (s *Store) SaveCard(card *model.Card) error {
	_, err := s.Pool.Exec(
		context.Background(),
		`INSERT INTO cards (id, local_id, set_id, name, category, data_json, image_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (id) DO UPDATE SET
		   name = EXCLUDED.name,
		   data_json = EXCLUDED.data_json,
		   image_url = EXCLUDED.image_url`,
		card.ID, card.LocalID, card.SetID, card.Name, card.Category, card.DataJSON, card.ImageURL,
	)
	if err != nil {
		return fmt.Errorf("save card: %w", err)
	}
	return nil
}
