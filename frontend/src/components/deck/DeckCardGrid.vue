<script setup lang="ts">
import { ref } from 'vue'
import CardImage from '@/components/common/CardImage.vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import { cardApi } from '@/api/deckApi'
import type { DeckCard } from '@/types/deck'

defineProps<{
  cards: DeckCard[]
}>()

const emit = defineEmits<{
  imageUpdated: [cardId: string, imageUrl: string]
}>()

const showImageDialog = ref(false)
const editingCard = ref<DeckCard | null>(null)
const imageUrlInput = ref('')
const saving = ref(false)

function openImageDialog(dc: DeckCard) {
  editingCard.value = dc
  imageUrlInput.value = dc.card?.imageUrl ?? ''
  showImageDialog.value = true
}

async function saveImageUrl() {
  if (!editingCard.value) return
  saving.value = true
  try {
    await cardApi.patchImage(editingCard.value.cardId, imageUrlInput.value)
    if (editingCard.value.card) {
      editingCard.value.card.imageUrl = imageUrlInput.value
    }
    emit('imageUpdated', editingCard.value.cardId, imageUrlInput.value)
    showImageDialog.value = false
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="grid grid-cols-4 sm:grid-cols-6 md:grid-cols-8 gap-2">
    <div v-for="dc in cards" :key="dc.cardId" class="relative group">
      <CardImage :src="dc.card?.imageUrl ?? ''" :alt="dc.card?.name ?? dc.cardId" size="sm" />
      <span
        class="absolute bottom-1 left-1 bg-slate-600 text-white text-base rounded-full w-8 h-8 flex items-center justify-center"
      >
        {{ dc.quantity }}
      </span>
      <button
        class="absolute bottom-1 right-4 bg-slate-800/60 text-white rounded-full w-8 h-8 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
        title="更換圖片"
        @click="openImageDialog(dc)"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 16 16"
          fill="currentColor"
          class="w-4 h-4"
        >
          <path
            d="M11.013 1.427a1.75 1.75 0 0 1 2.474 0l1.086 1.086a1.75 1.75 0 0 1 0 2.474l-8.61 8.61c-.21.21-.47.364-.756.445l-3.251.93a.75.75 0 0 1-.927-.928l.929-3.25c.081-.286.235-.547.445-.758l8.61-8.61Zm1.414 1.06a.25.25 0 0 0-.354 0L10.811 3.75l1.439 1.44 1.263-1.263a.25.25 0 0 0 0-.354l-1.086-1.086ZM11.189 6.25l-1.44-1.44-6.37 6.37a.25.25 0 0 0-.063.107l-.558 1.953 1.953-.558a.25.25 0 0 0 .108-.064l6.37-6.368Z"
          />
        </svg>
      </button>
    </div>
  </div>

  <Dialog v-model:visible="showImageDialog" header="自訂卡片圖片" modal :style="{ width: '450px' }">
    <div class="flex flex-col gap-3">
      <p class="text-sm text-gray-500">
        {{ editingCard?.card?.name ?? editingCard?.cardId }}
      </p>
      <InputText v-model="imageUrlInput" placeholder="貼上圖片 URL" class="w-full" />
      <div v-if="imageUrlInput" class="flex justify-center">
        <img
          :src="imageUrlInput"
          class="max-h-48 rounded-lg"
          @error="($event.target as HTMLImageElement).style.display = 'none'"
        />
      </div>
      <Button label="儲存" :loading="saving" :disabled="saving" @click="saveImageUrl" />
    </div>
  </Dialog>
</template>
