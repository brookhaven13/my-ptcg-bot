package model

import "time"

type Card struct {
	ID        string    `json:"id"`
	LocalID   string    `json:"localId"`
	SetID     string    `json:"setId"`
	Name      string    `json:"name"`
	Category  string    `json:"category"`
	DataJSON  string    `json:"-"`
	ImageURL  string    `json:"imageUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type CardDetail struct {
	ID           string       `json:"id"`
	LocalID      string       `json:"localId"`
	Name         string       `json:"name"`
	Category     string       `json:"category"`
	HP           int          `json:"hp,omitempty"`
	Types        []string     `json:"types,omitempty"`
	Stage        string       `json:"stage,omitempty"`
	EvolveFrom   string       `json:"evolveFrom,omitempty"`
	Attacks      []Attack     `json:"attacks,omitempty"`
	Weaknesses   []TypeValue  `json:"weaknesses,omitempty"`
	Resistances  []TypeValue  `json:"resistances,omitempty"`
	Retreat      int          `json:"retreat,omitempty"`
	Abilities    []Ability    `json:"abilities,omitempty"`
	ImageURL     string       `json:"imageUrl"`
	Set          SetBrief     `json:"set"`
	Rarity       string       `json:"rarity,omitempty"`
}

type Attack struct {
	Name   string   `json:"name"`
	Cost   []string `json:"cost"`
	Damage any      `json:"damage,omitempty"`
	Effect string   `json:"effect,omitempty"`
}

type TypeValue struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Ability struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type SetBrief struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
