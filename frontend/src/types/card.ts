export interface CardDetail {
  id: string
  localId: string
  name: string
  category: 'Pokemon' | 'Trainer' | 'Energy'
  hp?: number
  types?: string[]
  stage?: string
  evolveFrom?: string
  attacks?: Attack[]
  weaknesses?: TypeValue[]
  resistances?: TypeValue[]
  retreat?: number
  abilities?: Ability[]
  imageUrl: string
  set: SetBrief
  rarity?: string
}

export interface Attack {
  name: string
  cost: string[]
  damage?: number | string
  effect?: string
}

export interface TypeValue {
  type: string
  value: string
}

export interface Ability {
  type: string
  name: string
  effect: string
}

export interface SetBrief {
  id: string
  name: string
}
