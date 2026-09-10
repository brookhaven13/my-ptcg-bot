import { defineStore } from 'pinia'
import { ref } from 'vue'
import { battleApi } from '@/api/battleApi'
import type { GameState, BattleMode } from '@/types/battle'
import type { GameEvent, GameSummary } from '@/api/battleApi'

export const useBattleStore = defineStore('battle', () => {
  const gameId = ref<string | null>(null)
  const state = ref<GameState | null>(null)
  const events = ref<GameEvent[]>([])
  const loading = ref(false)
  const error = ref('')
  const savedGames = ref<GameSummary[]>([])

  async function startBattle(playerDeckId: number, aiDeckId: number, mode: BattleMode) {
    loading.value = true
    error.value = ''
    try {
      const res = await battleApi.start({ playerDeckId, aiDeckId, mode })
      gameId.value = res.gameId
      state.value = res.state
      events.value = res.events
      return res
    } catch (e: any) {
      error.value = e?.message || '開始對戰失敗'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function resumeBattle(id: string) {
    loading.value = true
    error.value = ''
    try {
      const res = await battleApi.resume(id)
      gameId.value = res.gameId
      state.value = res.state
      events.value = res.events ?? []
      return res
    } catch (e: any) {
      error.value = e?.message || '恢復對戰失敗'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchSavedGames() {
    try {
      savedGames.value = await battleApi.listGames()
    } catch {
      savedGames.value = []
    }
  }

  async function deleteGame(id: string) {
    await battleApi.deleteGame(id)
    savedGames.value = savedGames.value.filter((g) => g.id !== id)
  }

  function updateState(newState: GameState, newEvents?: GameEvent[]) {
    state.value = newState
    if (newEvents) {
      events.value = [...events.value, ...newEvents]
    }
  }

  function addEvents(newEvents: GameEvent[]) {
    events.value = [...events.value, ...newEvents]
  }

  function reset() {
    gameId.value = null
    state.value = null
    events.value = []
    error.value = ''
  }

  return {
    gameId,
    state,
    events,
    loading,
    error,
    savedGames,
    startBattle,
    resumeBattle,
    fetchSavedGames,
    deleteGame,
    updateState,
    addEvents,
    reset,
  }
})
