<script setup lang="ts">
import Button from 'primevue/button'
import Card from 'primevue/card'
import { ref } from 'vue'
import { deckApi } from '@/api/deckApi'
import DeckCardGrid from './DeckCardGrid.vue'
import type { Deck } from '@/types/deck'

defineProps<{
  decks: Deck[]
}>()

const emit = defineEmits<{
  edit: [deck: Deck]
  delete: [id: number]
}>()

const expandedId = ref<number | null>(null)
const loadedDecks = ref<Map<number, Deck>>(new Map())
const loadingId = ref<number | null>(null)

async function toggleExpand(deck: Deck) {
  if (expandedId.value === deck.id) {
    expandedId.value = null
    return
  }

  expandedId.value = deck.id

  if (!loadedDecks.value.has(deck.id)) {
    loadingId.value = deck.id
    try {
      const detail = await deckApi.get(deck.id)
      loadedDecks.value.set(deck.id, detail)
    } finally {
      loadingId.value = null
    }
  }
}

function handleEdit(deck: Deck, event: Event) {
  event.stopPropagation()
  const detail = loadedDecks.value.get(deck.id)
  emit('edit', detail ?? deck)
}

function handleDelete(id: number, event: Event) {
  event.stopPropagation()
  expandedId.value = null
  loadedDecks.value.delete(id)
  emit('delete', id)
}

function refreshDeck(deck: Deck) {
  loadedDecks.value.delete(deck.id)
  if (expandedId.value === deck.id) {
    toggleExpand(deck)
  }
}

defineExpose({ refreshDeck })
</script>

<template>
  <div class="flex flex-col gap-3">
    <p v-if="decks.length === 0" class="text-gray-500">尚無牌組</p>

    <Card v-for="deck in decks" :key="deck.id">
      <template #title>
        <div class="flex items-center justify-between cursor-pointer" @click="toggleExpand(deck)">
          <span>{{ deck.name }}</span>
          <div class="flex items-center gap-2">
            <i
              class="pi text-sm text-gray-400 transition-transform duration-200"
              :class="expandedId === deck.id ? 'pi-chevron-up' : 'pi-chevron-down'"
            />
          </div>
        </div>
      </template>

      <template #content v-if="expandedId === deck.id">
        <div v-if="loadingId === deck.id" class="text-center py-4 text-gray-400">
          <i class="pi pi-spinner pi-spin mr-2" />載入中...
        </div>
        <template v-else-if="loadedDecks.get(deck.id)?.cards?.length">
          <div class="py-4">

            <DeckCardGrid :cards="loadedDecks.get(deck.id)!.cards!" />
          </div>
        </template>
        <div v-else class="text-center py-4 text-gray-400">
          此牌組沒有卡片
        </div>

        <div class="flex items-center justify-between gap-4 border-t border-gray-200">
          <span class="text-sm font-normal text-gray-500">{{ deck.cardCount ?? 0 }} 張</span>
          <div class="flex gap-4 pt-3 ">
            <Button
              label="編輯"
              size="small"
              severity="secondary"
              @click="handleEdit(deck, $event)"
            />
            <Button
              label="刪除"
              size="small"
              severity="danger"
              text
              @click="handleDelete(deck.id, $event)"
            />
          </div>
        </div>
      </template>
    </Card>
  </div>
</template>
