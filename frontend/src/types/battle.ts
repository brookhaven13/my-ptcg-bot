import type { CardDetail } from './card'

export type GamePhase = 'SETUP' | 'DRAW' | 'MAIN_PHASE' | 'ATTACK' | 'BETWEEN_TURNS' | 'GAME_OVER'
export type BattleMode = 'virtual' | 'physical'

export interface GameState {
  id: string
  mode: BattleMode
  phase: GamePhase
  turnNumber: number
  activePlayer: 'player' | 'ai'
  player: PlayerState
  ai: PlayerState
  winner?: string
  winReason?: string
}

export interface PlayerState {
  deck: CardInstance[]
  hand: CardInstance[]
  active: BoardPokemon | null
  bench: BoardPokemon[]
  prizes: CardInstance[]
  discard: CardInstance[]
}

export interface CardInstance {
  uid: string
  cardId: string
  card?: CardDetail
}

export interface BoardPokemon {
  pokemon: CardInstance
  attachedEnergy: CardInstance[]
  damageCounters: number
  status: StatusEffect[]
  evolvedFrom?: BoardPokemon
}

export interface StatusEffect {
  type: string
}

export interface WSMessage {
  type: 'action' | 'state_update' | 'ai_action' | 'error' | 'game_over' | 'waiting'
  [key: string]: unknown
}
