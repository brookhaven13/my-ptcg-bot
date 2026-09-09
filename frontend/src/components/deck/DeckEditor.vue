<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'

const props = defineProps<{
  initialName?: string
  initialRawList?: string
  isEdit?: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  submit: [name: string, rawList: string]
}>()

const deckName = ref('')
const pokemonList = ref('')
const energyList = ref('')
const supporterList = ref('')
const itemList = ref('')
const stadiumList = ref('')

onMounted(() => {
  if (props.initialName) deckName.value = props.initialName
  if (props.initialRawList) parseRawListIntoSections(props.initialRawList)
})

function parseRawListIntoSections(raw: string) {
  const sectionMap: Record<string, string[]> = {
    '寶可夢卡': [],
    '能量卡': [],
    '支援者卡': [],
    '物品卡': [],
    '競技場卡': [],
  }
  let current = '寶可夢卡'

  for (const line of raw.split('\n')) {
    const trimmed = line.trim()
    if (!trimmed) continue
    const match = trimmed.match(/^\[(.+)]$/)
    if (match && match[1] in sectionMap) {
      current = match[1]
      continue
    }
    sectionMap[current].push(trimmed)
  }

  pokemonList.value = sectionMap['寶可夢卡'].join('\n')
  energyList.value = sectionMap['能量卡'].join('\n')
  supporterList.value = sectionMap['支援者卡'].join('\n')
  itemList.value = sectionMap['物品卡'].join('\n')
  stadiumList.value = sectionMap['競技場卡'].join('\n')
}

const hasContent = computed(() => {
  return deckName.value.trim() && (
    pokemonList.value.trim() ||
    energyList.value.trim() ||
    supporterList.value.trim() ||
    itemList.value.trim() ||
    stadiumList.value.trim()
  )
})

function buildRawList(): string {
  const sections: string[] = []
  if (pokemonList.value.trim()) {
    sections.push(`[寶可夢卡]\n${pokemonList.value.trim()}`)
  }
  if (energyList.value.trim()) {
    sections.push(`[能量卡]\n${energyList.value.trim()}`)
  }
  if (supporterList.value.trim()) {
    sections.push(`[支援者卡]\n${supporterList.value.trim()}`)
  }
  if (itemList.value.trim()) {
    sections.push(`[物品卡]\n${itemList.value.trim()}`)
  }
  if (stadiumList.value.trim()) {
    sections.push(`[競技場卡]\n${stadiumList.value.trim()}`)
  }
  return sections.join('\n\n')
}

function handleSubmit() {
  if (!hasContent.value || props.loading) return
  emit('submit', deckName.value.trim(), buildRawList())
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <InputText v-model="deckName" placeholder="牌組名稱" :disabled="loading" />

    <div class="flex flex-col gap-1">
      <label class="text-sm font-semibold">寶可夢卡</label>
      <Textarea
        v-model="pokemonList"
        rows="6"
        placeholder="名稱&#9;擴充標記&#9;編號/總數&#9;數量&#10;多龍梅西亞&#9;MC&#9;546/742&#9;4&#10;多龍奇&#9;MC&#9;547/742&#9;4"
      />
    </div>

    <div class="flex flex-col gap-1">
      <label class="text-sm font-semibold">能量卡</label>
      <Textarea
        v-model="energyList"
        rows="3"
        placeholder="基本超能量&#9;MC&#9;001/742&#9;3"
      />
    </div>

    <div class="flex flex-col gap-1">
      <label class="text-sm font-semibold">支援者卡</label>
      <Textarea
        v-model="supporterList"
        rows="3"
        placeholder="莉莉艾的決意&#9;MC&#9;622/742&#9;4"
      />
    </div>

    <div class="flex flex-col gap-1">
      <label class="text-sm font-semibold">物品卡</label>
      <Textarea
        v-model="itemList"
        rows="3"
        placeholder="高級球&#9;MC&#9;582/742&#9;4"
      />
    </div>

    <div class="flex flex-col gap-1">
      <label class="text-sm font-semibold">競技場卡</label>
      <Textarea
        v-model="stadiumList"
        rows="2"
        placeholder="阻礙之塔&#9;MC&#9;688/742&#9;1"
      />
    </div>

    <Button
      :label="isEdit ? '更新牌組' : '建立牌組'"
      :loading="loading"
      :disabled="!hasContent || loading"
      @click="handleSubmit"
    />
  </div>
</template>
