package model

import "time"

type Deck struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Owner     string     `json:"owner"`
	RawList   string     `json:"rawList"`
	CardCount int        `json:"cardCount"`
	Cards     []DeckCard `json:"cards,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type DeckCard struct {
	CardID   string      `json:"cardId"`
	Quantity int         `json:"quantity"`
	Card     *CardDetail `json:"card,omitempty"`
}
