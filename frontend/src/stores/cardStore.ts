import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { CardDetail } from '@/types/card'
import { api } from '@/api/client'

export const useCardStore = defineStore('card', () => {
  const cards = ref<Map<string, CardDetail>>(new Map())

  function getCard(id: string): CardDetail | undefined {
    return cards.value.get(id)
  }

  async function fetchCard(id: string): Promise<CardDetail> {
    const cached = cards.value.get(id)
    if (cached) return cached

    const card = await api.get<CardDetail>(`/cards/${id}`)
    cards.value.set(id, card)
    return card
  }

  async function lookupCards(entries: { setCode: string; number: string }[]): Promise<Record<string, CardDetail>> {
    const result = await api.post<Record<string, CardDetail>>('/cards/lookup', { cards: entries })
    for (const [id, card] of Object.entries(result)) {
      cards.value.set(id, card)
    }
    return result
  }

  return { cards, getCard, fetchCard, lookupCards }
})
