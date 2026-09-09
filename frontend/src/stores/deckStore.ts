import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Deck } from '@/types/deck'
import { deckApi } from '@/api/deckApi'

export const useDeckStore = defineStore('deck', () => {
  const playerDecks = ref<Deck[]>([])
  const aiDecks = ref<Deck[]>([])
  const loading = ref(false)

  async function fetchDecks(owner: 'player' | 'ai') {
    loading.value = true
    try {
      const decks = await deckApi.list(owner)
      if (owner === 'player') {
        playerDecks.value = decks
      } else {
        aiDecks.value = decks
      }
    } finally {
      loading.value = false
    }
  }

  async function createDeck(name: string, owner: 'player' | 'ai', rawList: string) {
    const deck = await deckApi.create({ name, owner, rawList })
    if (owner === 'player') {
      playerDecks.value.unshift(deck)
    } else {
      aiDecks.value.unshift(deck)
    }
    return deck
  }

  async function updateDeck(id: number, name: string, owner: 'player' | 'ai', rawList: string) {
    const updated = await deckApi.update(id, { name, owner, rawList })
    const list = owner === 'player' ? playerDecks : aiDecks
    const idx = list.value.findIndex(d => d.id === id)
    if (idx !== -1) {
      list.value[idx] = updated
    }
    return updated
  }

  async function deleteDeck(id: number, owner: 'player' | 'ai') {
    await deckApi.delete(id)
    if (owner === 'player') {
      playerDecks.value = playerDecks.value.filter(d => d.id !== id)
    } else {
      aiDecks.value = aiDecks.value.filter(d => d.id !== id)
    }
  }

  return { playerDecks, aiDecks, loading, fetchDecks, createDeck, updateDeck, deleteDeck }
})
