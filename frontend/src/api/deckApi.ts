import { api } from './client'
import type { Deck, CreateDeckRequest } from '@/types/deck'

export const deckApi = {
  list: (owner: 'player' | 'ai') =>
    api.get<Deck[]>(`/decks?owner=${owner}`),

  get: (id: number) =>
    api.get<Deck>(`/decks/${id}`),

  create: (req: CreateDeckRequest) =>
    api.post<Deck>('/decks', req),

  update: (id: number, req: CreateDeckRequest) =>
    api.put<Deck>(`/decks/${id}`, req),

  delete: (id: number) =>
    api.delete(`/decks/${id}`),
}

export const cardApi = {
  patchImage: (cardId: string, imageUrl: string) =>
    api.patch<{ id: string; imageUrl: string }>('/cards/image', { id: cardId, imageUrl }),
}
