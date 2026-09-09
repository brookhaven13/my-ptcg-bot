package tcgdex

type TCGdexCard struct {
	ID          string        `json:"id"`
	LocalID     string        `json:"localId"`
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	HP          int           `json:"hp,omitempty"`
	Types       []string      `json:"types,omitempty"`
	Stage       string        `json:"stage,omitempty"`
	EvolveFrom  string        `json:"evolveFrom,omitempty"`
	Attacks     []TCGdexAttack    `json:"attacks,omitempty"`
	Weaknesses  []TCGdexTypeVal   `json:"weaknesses,omitempty"`
	Resistances []TCGdexTypeVal   `json:"resistances,omitempty"`
	Retreat     int           `json:"retreat,omitempty"`
	Abilities   []TCGdexAbility   `json:"abilities,omitempty"`
	Image       string        `json:"image,omitempty"`
	Set         TCGdexSetBrief    `json:"set"`
	Rarity      string        `json:"rarity,omitempty"`
	Illustrator string        `json:"illustrator,omitempty"`
	Description string        `json:"description,omitempty"`
}

type TCGdexAttack struct {
	Name   string   `json:"name"`
	Cost   []string `json:"cost"`
	Damage any      `json:"damage,omitempty"`
	Effect string   `json:"effect,omitempty"`
}

type TCGdexTypeVal struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type TCGdexAbility struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Effect string `json:"effect"`
}

type TCGdexSetBrief struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Logo      string         `json:"logo,omitempty"`
	Symbol    string         `json:"symbol,omitempty"`
	CardCount map[string]int `json:"cardCount,omitempty"`
}
