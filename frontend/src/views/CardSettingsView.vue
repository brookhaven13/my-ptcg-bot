<script setup lang="ts">
import SelectButton from 'primevue/selectbutton'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import { ref, watch } from 'vue'
import { useDeckStore } from '@/stores/deckStore'
import DeckEditor from '@/components/deck/DeckEditor.vue'
import DeckList from '@/components/deck/DeckList.vue'
import type { Deck } from '@/types/deck'

const deckStore = useDeckStore()
const deckListRef = ref<InstanceType<typeof DeckList> | null>(null)

const deckOwner = ref<'player' | 'ai'>('player')
const ownerOptions = [
  { label: '我的牌庫', value: 'player' },
  { label: 'AI 牌庫', value: 'ai' },
]

const showEditor = ref(false)
const editingDeck = ref<Deck | null>(null)
const submitting = ref(false)
const errorMsg = ref('')

watch(deckOwner, (owner) => {
  deckStore.fetchDecks(owner)
}, { immediate: true })

function openCreateDialog() {
  editingDeck.value = null
  errorMsg.value = ''
  showEditor.value = true
}

function openEditDialog(deck: Deck) {
  editingDeck.value = deck
  errorMsg.value = ''
  showEditor.value = true
}

async function handleSubmitDeck(name: string, rawList: string) {
  submitting.value = true
  errorMsg.value = ''
  try {
    if (editingDeck.value) {
      const updated = await deckStore.updateDeck(editingDeck.value.id, name, deckOwner.value, rawList)
      deckListRef.value?.refreshDeck(updated)
    } else {
      await deckStore.createDeck(name, deckOwner.value, rawList)
    }
    showEditor.value = false
    editingDeck.value = null
  } catch (e: any) {
    errorMsg.value = e?.message || '建立失敗，請稍後再試'
  } finally {
    submitting.value = false
  }
}

async function handleDeleteDeck(id: number) {
  await deckStore.deleteDeck(id, deckOwner.value)
}
</script>

<template>
  <div class="flex flex-col gap-4 max-w-3xl">
    <h1 class="text-2xl font-bold">卡牌設定</h1>

    <div class="flex items-center gap-4">
      <SelectButton v-model="deckOwner" :options="ownerOptions" optionLabel="label" optionValue="value" />
      <Button label="新增牌組" size="small" @click="openCreateDialog" />
    </div>

    <DeckList
      ref="deckListRef"
      :decks="deckOwner === 'player' ? deckStore.playerDecks : deckStore.aiDecks"
      @edit="openEditDialog"
      @delete="handleDeleteDeck"
    />

    <Dialog
      v-model:visible="showEditor"
      :header="editingDeck ? '編輯牌組' : '新增牌組'"
      modal
      :closable="!submitting"
      :closeOnEscape="!submitting"
      :style="{ width: '500px' }"
    >
      <p v-if="errorMsg" class="text-red-500 text-sm mb-3">{{ errorMsg }}</p>
      <DeckEditor
        :key="editingDeck?.id ?? 'new'"
        :initial-name="editingDeck?.name"
        :initial-raw-list="editingDeck?.rawList"
        :is-edit="!!editingDeck"
        :loading="submitting"
        @submit="handleSubmitDeck"
      />
    </Dialog>
  </div>
</template>
