package store

import (
	"context"
	"fmt"

	"github.com/harleylin/ptcg-bot/backend/internal/model"
)

func (s *Store) ListDecks(owner string) ([]model.Deck, error) {
	rows, err := s.Pool.Query(
		context.Background(),
		`SELECT d.id, d.name, d.owner, d.raw_list,
		        COALESCE(SUM(dc.quantity), 0) AS card_count,
		        d.created_at, d.updated_at
		 FROM decks d
		 LEFT JOIN deck_cards dc ON dc.deck_id = d.id
		 WHERE d.owner = $1
		 GROUP BY d.id
		 ORDER BY d.updated_at DESC`,
		owner,
	)
	if err != nil {
		return nil, fmt.Errorf("list decks: %w", err)
	}
	defer rows.Close()

	var decks []model.Deck
	for rows.Next() {
		var d model.Deck
		if err := rows.Scan(&d.ID, &d.Name, &d.Owner, &d.RawList, &d.CardCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan deck: %w", err)
		}
		decks = append(decks, d)
	}
	return decks, rows.Err()
}

func (s *Store) GetDeck(id int) (*model.Deck, error) {
	var d model.Deck
	err := s.Pool.QueryRow(
		context.Background(),
		"SELECT id, name, owner, raw_list, created_at, updated_at FROM decks WHERE id = $1",
		id,
	).Scan(&d.ID, &d.Name, &d.Owner, &d.RawList, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get deck: %w", err)
	}

	rows, err := s.Pool.Query(
		context.Background(),
		"SELECT card_id, quantity FROM deck_cards WHERE deck_id = $1",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get deck cards: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dc model.DeckCard
		if err := rows.Scan(&dc.CardID, &dc.Quantity); err != nil {
			return nil, fmt.Errorf("scan deck card: %w", err)
		}
		d.Cards = append(d.Cards, dc)
	}

	return &d, rows.Err()
}

func (s *Store) CreateDeck(d *model.Deck) error {
	tx, err := s.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	err = tx.QueryRow(
		context.Background(),
		"INSERT INTO decks (name, owner, raw_list) VALUES ($1, $2, $3) RETURNING id",
		d.Name, d.Owner, d.RawList,
	).Scan(&d.ID)
	if err != nil {
		return fmt.Errorf("insert deck: %w", err)
	}

	for _, dc := range d.Cards {
		_, err := tx.Exec(
			context.Background(),
			"INSERT INTO deck_cards (deck_id, card_id, quantity) VALUES ($1, $2, $3)",
			d.ID, dc.CardID, dc.Quantity,
		)
		if err != nil {
			return fmt.Errorf("insert deck card: %w", err)
		}
	}

	return tx.Commit(context.Background())
}

func (s *Store) UpdateDeck(d *model.Deck) error {
	tx, err := s.Pool.Begin(context.Background())
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(context.Background())

	_, err = tx.Exec(
		context.Background(),
		"UPDATE decks SET name = $1, raw_list = $2, updated_at = NOW() WHERE id = $3",
		d.Name, d.RawList, d.ID,
	)
	if err != nil {
		return fmt.Errorf("update deck: %w", err)
	}

	_, err = tx.Exec(context.Background(), "DELETE FROM deck_cards WHERE deck_id = $1", d.ID)
	if err != nil {
		return fmt.Errorf("delete old deck cards: %w", err)
	}

	for _, dc := range d.Cards {
		_, err := tx.Exec(
			context.Background(),
			"INSERT INTO deck_cards (deck_id, card_id, quantity) VALUES ($1, $2, $3)",
			d.ID, dc.CardID, dc.Quantity,
		)
		if err != nil {
			return fmt.Errorf("insert deck card: %w", err)
		}
	}

	return tx.Commit(context.Background())
}

func (s *Store) DeleteDeck(id int) error {
	_, err := s.Pool.Exec(context.Background(), "DELETE FROM decks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete deck: %w", err)
	}
	return nil
}
