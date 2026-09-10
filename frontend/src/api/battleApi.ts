import { api } from './client'
import type { GameState } from '@/types/battle'

export interface StartBattleRequest {
  playerDeckId: number
  aiDeckId: number
  mode: 'virtual' | 'physical'
}

export interface StartBattleResponse {
  gameId: string
  state: GameState
  events: GameEvent[]
}

export interface GameEvent {
  type: string
  message: string
  data?: unknown
}

export interface GameSummary {
  id: string
  playerDeckId: number
  aiDeckId: number
  mode: string
  status: string
  createdAt: string
  updatedAt: string
  playerDeckName: string
  aiDeckName: string
}

export const battleApi = {
  start: (req: StartBattleRequest) =>
    api.post<StartBattleResponse>('/battle/start', req),

  getState: (gameId: string) =>
    api.get<GameState>(`/battle/${gameId}`),

  listGames: () =>
    api.get<GameSummary[]>('/battle/games'),

  resume: (gameId: string) =>
    api.post<StartBattleResponse>(`/battle/resume/${gameId}`, {}),

  deleteGame: (gameId: string) =>
    api.delete(`/battle/${gameId}`),
}
