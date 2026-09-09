import type { CardDetail } from './card'

export interface Deck {
  id: number
  name: string
  owner: 'player' | 'ai'
  rawList: string
  cardCount: number
  cards?: DeckCard[]
  createdAt: string
  updatedAt: string
}

export interface DeckCard {
  cardId: string
  quantity: number
  card?: CardDetail
}

export interface CreateDeckRequest {
  name: string
  owner: 'player' | 'ai'
  rawList: string
}
