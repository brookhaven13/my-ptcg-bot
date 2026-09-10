<script setup lang="ts">
import { ref, computed } from 'vue'
import Button from 'primevue/button'
import Card from 'primevue/card'
import SelectButton from 'primevue/selectbutton'
import Popover from 'primevue/popover'
import { useDeckStore } from '@/stores/deckStore'
import { useBattleStore } from '@/stores/battleStore'
import { useWebSocket } from '@/composables/useWebSocket'
import CardImage from '@/components/common/CardImage.vue'
import type { BattleMode, CardInstance, BoardPokemon } from '@/types/battle'
import type { CardDetail } from '@/types/card'

const deckStore = useDeckStore()
const battleStore = useBattleStore()
const { connected, connect, send, onMessage, disconnect } = useWebSocket()

// --- Phase: lobby ---
const screen = ref<'lobby' | 'setup' | 'battle'>('lobby')
const selectedPlayerDeck = ref<number | null>(null)
const selectedAIDeck = ref<number | null>(null)
const selectedMode = ref<BattleMode>('virtual')
const starting = ref(false)
const validActions = ref<string[]>([])

deckStore.fetchDecks('player')
deckStore.fetchDecks('ai')
battleStore.fetchSavedGames()

const modeOptions = [
  { label: '虛擬對戰', value: 'virtual' },
  { label: '實體牌對戰', value: 'physical' },
]

async function startGame() {
  if (!selectedPlayerDeck.value || !selectedAIDeck.value) return
  starting.value = true
  try {
    const res = await battleStore.startBattle(
      selectedPlayerDeck.value,
      selectedAIDeck.value,
      selectedMode.value,
    )
    connect(res.gameId)
    onMessage(handleWSMessage)
    screen.value = 'setup'
  } catch {
    // error handled by store
  } finally {
    starting.value = false
  }
}

const resuming = ref(false)

async function resumeGame(gameId: string) {
  resuming.value = true
  try {
    const res = await battleStore.resumeBattle(gameId)
    connect(res.gameId)
    onMessage(handleWSMessage)
    const phase = res.state?.phase
    if (phase === 'SETUP') {
      screen.value = 'setup'
    } else {
      screen.value = 'battle'
      setTimeout(() => send({ type: 'sync' }), 300)
    }
  } catch {
    // error handled by store
  } finally {
    resuming.value = false
  }
}

async function deleteSavedGame(gameId: string) {
  await battleStore.deleteGame(gameId)
}

function goToLobby() {
  disconnect()
  battleStore.reset()
  battleStore.fetchSavedGames()
  screen.value = 'lobby'
}

// --- Phase: setup ---
const selectedActiveUID = ref<string | null>(null)
const selectedBenchUIDs = ref<string[]>([])

function isPokemon(cat?: string) {
  return cat === 'Pokemon' || cat === 'Pokémon'
}

function isBasicStage(stage?: string, evolveFrom?: string) {
  return stage === 'Basic' || stage === '基礎' || (!stage && !evolveFrom)
}

const handBasicPokemon = computed(() => {
  if (!battleStore.state) return []
  return battleStore.state.player.hand.filter(
    (c) => isPokemon(c.card?.category) && isBasicStage(c.card?.stage, c.card?.evolveFrom),
  )
})

function toggleBenchSelect(uid: string) {
  if (uid === selectedActiveUID.value) return
  const idx = selectedBenchUIDs.value.indexOf(uid)
  if (idx >= 0) {
    selectedBenchUIDs.value.splice(idx, 1)
  } else if (selectedBenchUIDs.value.length < 5) {
    selectedBenchUIDs.value.push(uid)
  }
}

function confirmSetup() {
  if (!selectedActiveUID.value) return
  send({
    type: 'place_pokemon',
    activeUid: selectedActiveUID.value,
    benchUids: selectedBenchUIDs.value,
  })
}

function startBattle() {
  send({ type: 'start_battle' })
  screen.value = 'battle'
}

// --- Phase: battle ---
const selectedHandCard = ref<string | null>(null)
const selectedTarget = ref<string | null>(null)
const actionMode = ref<string | null>(null)

function selectHandCard(uid: string) {
  selectedHandCard.value = selectedHandCard.value === uid ? null : uid
  actionMode.value = null
  selectedTarget.value = null
}

function performAction(type: string, extra: Record<string, unknown> = {}) {
  send({ type: 'action', action: type, ...extra })
  selectedHandCard.value = null
  selectedTarget.value = null
  actionMode.value = null
  draggingCard.value = null
}

function playPokemon() {
  if (!selectedHandCard.value) return
  performAction('play_pokemon', { cardUid: selectedHandCard.value })
}

function attachEnergy(targetUid: string) {
  if (!selectedHandCard.value) return
  performAction('attach_energy', { cardUid: selectedHandCard.value, targetUid })
}

function evolvePokemon(targetUid: string) {
  if (!selectedHandCard.value) return
  performAction('evolve_pokemon', { cardUid: selectedHandCard.value, targetUid })
}

function playTrainer() {
  if (!selectedHandCard.value) return
  performAction('play_trainer', { cardUid: selectedHandCard.value })
}

function doRetreat(benchUid: string) {
  performAction('retreat', { targetUid: benchUid })
}

function doAttack(index: number) {
  performAction('attack', { attackIndex: index })
}

function endTurn() {
  performAction('end_turn')
}

function manualDraw(cardUid: string) {
  performAction('manual_draw', { cardUid })
}

// --- Drag & Drop ---
const draggingCard = ref<CardInstance | null>(null)
const dragOverZone = ref<string | null>(null)

function onDragStart(e: DragEvent, card: CardInstance) {
  if (!isPlayerTurn.value || gamePhase.value !== 'MAIN_PHASE') return
  draggingCard.value = card
  e.dataTransfer!.effectAllowed = 'move'
  e.dataTransfer!.setData('text/plain', card.uid)
  // Make the dragged element semi-transparent
  const target = e.target as HTMLElement
  setTimeout(() => target.classList.add('opacity-40'), 0)
}

function onDragEnd(e: DragEvent) {
  draggingCard.value = null
  dragOverZone.value = null
  const target = e.target as HTMLElement
  target.classList.remove('opacity-40')
}

function onDragEnter(zone: string) {
  dragOverZone.value = zone
}

function onDragLeave(e: DragEvent, zone: string) {
  const related = e.relatedTarget as HTMLElement | null
  const current = e.currentTarget as HTMLElement
  if (related && current.contains(related)) return
  if (dragOverZone.value === zone) {
    dragOverZone.value = null
  }
}

function onDropPokemonTarget(e: DragEvent, targetUid: string) {
  e.preventDefault()
  dragOverZone.value = null
  const card = draggingCard.value
  if (!card) return

  if (card.card?.category === 'Energy' && canDo('attach_energy')) {
    performAction('attach_energy', { cardUid: card.uid, targetUid })
  } else if (isPokemon(card.card?.category) && card.card?.evolveFrom && canDo('evolve_pokemon')) {
    performAction('evolve_pokemon', { cardUid: card.uid, targetUid })
  }
}

function onDropField(e: DragEvent) {
  e.preventDefault()
  dragOverZone.value = null
  const card = draggingCard.value
  if (!card) return

  const cat = card.card?.category ?? ''
  if (isPokemon(cat) && isBasicStage(card.card?.stage, card.card?.evolveFrom) && canDo('play_pokemon')) {
    performAction('play_pokemon', { cardUid: card.uid })
  } else if (['Supporter', 'Item', 'Stadium'].includes(cat)) {
    performAction('play_trainer', { cardUid: card.uid })
  }
}

const isDraggingBasic = computed(() => {
  const c = draggingCard.value
  return c && isPokemon(c.card?.category) && isBasicStage(c.card?.stage, c.card?.evolveFrom)
})

const isDraggingEnergy = computed(() => {
  return draggingCard.value?.card?.category === 'Energy'
})

const isDraggingEvolution = computed(() => {
  const c = draggingCard.value
  return c && isPokemon(c.card?.category) && !!c.card?.evolveFrom
})

const isDraggingTrainer = computed(() => {
  const c = draggingCard.value
  return c && ['Supporter', 'Item', 'Stadium'].includes(c.card?.category ?? '')
})

const showFieldDropHint = computed(() => isDraggingBasic.value || isDraggingTrainer.value)
const showPokemonDropHint = computed(() => isDraggingEnergy.value || isDraggingEvolution.value)

// --- Card Preview Popover ---
const cardPopover = ref()
const previewCard = ref<CardDetail | null>(null)

function showPreview(e: Event, card?: CardDetail) {
  if (!card) return
  previewCard.value = card
  cardPopover.value?.toggle(e)
}

function onFieldCardClick(e: Event, bp: BoardPokemon) {
  if (actionMode.value === 'attach_energy') {
    attachEnergy(bp.pokemon.uid)
  } else if (actionMode.value === 'evolve') {
    evolvePokemon(bp.pokemon.uid)
  } else if (actionMode.value === 'retreat') {
    doRetreat(bp.pokemon.uid)
  } else {
    showPreview(e, bp.pokemon.card)
  }
}

function onActiveCardClick(e: Event) {
  if (!playerState.value?.active) return
  if (actionMode.value === 'attach_energy') {
    attachEnergy(playerState.value.active.pokemon.uid)
  } else if (actionMode.value === 'evolve') {
    evolvePokemon(playerState.value.active.pokemon.uid)
  } else {
    showPreview(e, playerState.value.active.pokemon.card)
  }
}

// --- WS handler ---
function handleWSMessage(msg: { type: string; payload?: any }) {
  if (msg.type === 'state_update' || msg.type === 'ai_action') {
    if (msg.payload?.state) {
      battleStore.updateState(msg.payload.state, msg.payload.events)
    }
    if (msg.payload?.validActions) {
      validActions.value = msg.payload.validActions
    }
  }
  if (msg.type === 'error') {
    console.error('[battle]', msg.payload?.message)
  }
}

// --- Helpers ---
const playerState = computed(() => battleStore.state?.player)
const aiState = computed(() => battleStore.state?.ai)
const isPlayerTurn = computed(() => battleStore.state?.activePlayer === 'player')
const gamePhase = computed(() => battleStore.state?.phase)
const isPhysical = computed(() => battleStore.state?.mode === 'physical')

function cardLabel(c: CardInstance) {
  return c.card?.name ?? c.cardId
}

function pokemonHP(bp: BoardPokemon) {
  const hp = bp.pokemon.card?.hp ?? 0
  const damage = bp.damageCounters * 10
  return `${hp - damage}/${hp}`
}

function canDo(action: string) {
  return validActions.value.includes(action)
}

const selectedCardInfo = computed(() => {
  if (!selectedHandCard.value || !playerState.value) return null
  return playerState.value.hand.find((c) => c.uid === selectedHandCard.value) ?? null
})

const eventLog = computed(() => {
  return battleStore.events.slice(-20).reverse()
})
</script>

<template>
  <div class="max-w-5xl mx-auto">
    <!-- Card Preview Popover -->
    <Popover ref="cardPopover">
      <div v-if="previewCard" class="p-1">
        <CardImage :src="previewCard.imageUrl ?? ''" :alt="previewCard.name ?? ''" size="lg" />
      </div>
    </Popover>

    <!-- Lobby -->
    <template v-if="screen === 'lobby'">
      <h1 class="text-2xl font-bold mb-4">開始對戰</h1>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
        <Card>
          <template #title>選擇玩家牌組</template>
          <template #content>
            <div class="flex flex-col gap-2">
              <Button
                v-for="deck in deckStore.playerDecks"
                :key="deck.id"
                :label="`${deck.name} (${deck.cardCount}張)`"
                :severity="selectedPlayerDeck === deck.id ? 'primary' : 'secondary'"
                size="small"
                @click="selectedPlayerDeck = deck.id"
              />
              <p v-if="deckStore.playerDecks.length === 0" class="text-gray-400">
                尚無玩家牌組，請先到卡牌設定建立
              </p>
            </div>
          </template>
        </Card>

        <Card>
          <template #title>選擇 AI 牌組</template>
          <template #content>
            <div class="flex flex-col gap-2">
              <Button
                v-for="deck in deckStore.aiDecks"
                :key="deck.id"
                :label="`${deck.name} (${deck.cardCount}張)`"
                :severity="selectedAIDeck === deck.id ? 'primary' : 'secondary'"
                size="small"
                @click="selectedAIDeck = deck.id"
              />
              <p v-if="deckStore.aiDecks.length === 0" class="text-gray-400">
                尚無 AI 牌組，請先到卡牌設定建立
              </p>
            </div>
          </template>
        </Card>
      </div>

      <div class="flex items-center gap-4 mb-4">
        <span class="text-sm font-semibold">對戰模式：</span>
        <SelectButton v-model="selectedMode" :options="modeOptions" optionLabel="label" optionValue="value" />
      </div>

      <Button
        label="開始對戰"
        :disabled="!selectedPlayerDeck || !selectedAIDeck"
        :loading="starting"
        @click="startGame"
      />

      <p v-if="battleStore.error" class="text-red-500 mt-2 text-sm">{{ battleStore.error }}</p>

      <!-- Saved Games -->
      <div v-if="battleStore.savedGames.length > 0" class="mt-6">
        <h2 class="text-lg font-bold mb-3">繼續對戰</h2>
        <div class="flex flex-col gap-2">
          <div
            v-for="game in battleStore.savedGames"
            :key="game.id"
            class="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
          >
            <div class="text-sm">
              <span class="font-semibold">{{ game.playerDeckName }}</span>
              <span class="text-gray-400 mx-1">vs</span>
              <span class="font-semibold">{{ game.aiDeckName }}</span>
              <span class="text-gray-400 ml-2">{{ game.mode === 'physical' ? '實體牌' : '虛擬' }}</span>
              <span class="text-gray-400 ml-2 text-xs">
                {{ new Date(game.updatedAt).toLocaleString('zh-TW') }}
              </span>
            </div>
            <div class="flex gap-2">
              <Button
                label="繼續"
                size="small"
                severity="success"
                :loading="resuming"
                @click="resumeGame(game.id)"
              />
              <Button
                label="刪除"
                size="small"
                severity="danger"
                text
                @click="deleteSavedGame(game.id)"
              />
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Setup -->
    <template v-else-if="screen === 'setup'">
      <h1 class="text-2xl font-bold mb-4">放置寶可夢</h1>
      <p class="text-sm text-gray-500 mb-4">選擇一隻基本寶可夢作為戰鬥寶可夢，可選最多 5 隻作為後備</p>

      <div class="grid grid-cols-4 sm:grid-cols-6 gap-3 mb-4">
        <div
          v-for="c in handBasicPokemon"
          :key="c.uid"
          class="cursor-pointer relative"
          @click="
            selectedActiveUID
              ? (selectedActiveUID === c.uid ? (selectedActiveUID = null) : toggleBenchSelect(c.uid))
              : (selectedActiveUID = c.uid)
          "
        >
          <CardImage :src="c.card?.imageUrl ?? ''" :alt="cardLabel(c)" size="sm" />
          <span
            v-if="selectedActiveUID === c.uid"
            class="absolute top-0 left-0 bg-blue-500 text-white text-xs px-1 rounded"
          >
            戰鬥
          </span>
          <span
            v-else-if="selectedBenchUIDs.includes(c.uid)"
            class="absolute top-0 left-0 bg-green-500 text-white text-xs px-1 rounded"
          >
            後備
          </span>
        </div>
      </div>

      <div class="flex gap-3">
        <Button
          label="確認放置"
          :disabled="!selectedActiveUID"
          @click="confirmSetup"
        />
        <Button
          label="開始對戰"
          severity="success"
          :disabled="!playerState?.active"
          @click="startBattle"
        />
      </div>
    </template>

    <!-- Battle -->
    <template v-else-if="screen === 'battle'">
      <div class="flex items-center justify-between mb-3">
        <h1 class="text-xl font-bold">
          第 {{ battleStore.state?.turnNumber }} 回合
          <span class="text-sm font-normal text-gray-500 ml-2">
            {{ isPlayerTurn ? '你的回合' : 'AI 的回合' }}
          </span>
        </h1>
        <div class="flex items-center gap-2">
          <span
            class="text-xs px-2 py-1 rounded"
            :class="connected ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'"
          >
            {{ connected ? '已連線' : '未連線' }}
          </span>
          <Button label="返回大廳" size="small" severity="secondary" text @click="goToLobby" />
        </div>
      </div>

      <!-- Game Over -->
      <div
        v-if="gamePhase === 'GAME_OVER'"
        class="p-4 mb-4 rounded-lg text-center text-lg font-bold"
        :class="battleStore.state?.winner === 'player' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'"
      >
        {{ battleStore.state?.winner === 'player' ? '你贏了！' : 'AI 獲勝！' }}
        <span class="text-sm font-normal block">{{ battleStore.state?.winReason }}</span>
      </div>

      <!-- AI Field -->
      <div class="mb-4 p-3 bg-gray-50 rounded-lg">
        <div class="text-xs text-gray-400 mb-2">AI 場地</div>
        <div class="flex gap-3 items-start">
          <!-- AI Active -->
          <div v-if="aiState?.active" class="text-center cursor-pointer" @click="showPreview($event, aiState.active.pokemon.card)">
            <div class="text-xs text-gray-500 mb-1">戰鬥</div>
            <CardImage :src="aiState.active.pokemon.card?.imageUrl ?? ''" :alt="cardLabel(aiState.active.pokemon)" size="sm" />
            <div class="text-xs mt-1">HP {{ pokemonHP(aiState.active) }}</div>
            <div v-if="aiState.active.attachedEnergy.length" class="text-xs text-gray-400">
              能量 ×{{ aiState.active.attachedEnergy.length }}
            </div>
          </div>
          <!-- AI Bench -->
          <div v-for="bp in aiState?.bench" :key="bp.pokemon.uid" class="text-center cursor-pointer" @click="showPreview($event, bp.pokemon.card)">
            <div class="text-xs text-gray-500 mb-1">後備</div>
            <CardImage :src="bp.pokemon.card?.imageUrl ?? ''" :alt="cardLabel(bp.pokemon)" size="sm" />
            <div class="text-xs mt-1">HP {{ pokemonHP(bp) }}</div>
          </div>
          <!-- AI Info -->
          <div class="ml-auto text-right text-xs text-gray-400">
            <div>牌庫 {{ aiState?.deck.length ?? 0 }}</div>
            <div>手牌 {{ aiState?.hand.length ?? 0 }}</div>
            <div>獎品 {{ aiState?.prizes.length ?? 0 }}</div>
            <div>棄牌 {{ aiState?.discard.length ?? 0 }}</div>
          </div>
        </div>
      </div>

      <!-- Player Field (drop zone) -->
      <div
        class="mb-4 p-3 rounded-lg transition-colors"
        :class="[
          dragOverZone === 'field' && showFieldDropHint ? 'bg-blue-100 ring-2 ring-blue-400 ring-dashed' : 'bg-blue-50',
          showFieldDropHint ? 'ring-1 ring-blue-200 ring-dashed' : ''
        ]"
        @dragover.prevent
        @dragenter.prevent="onDragEnter('field')"
        @dragleave="onDragLeave($event, 'field')"
        @drop="onDropField"
      >
        <div class="flex items-center justify-between mb-2">
          <div class="text-xs text-blue-400">你的場地</div>
          <div v-if="draggingCard" class="text-xs text-blue-500 animate-pulse">
            {{ isDraggingBasic ? '放開以放置到後備區' : isDraggingTrainer ? '放開以使用' : isDraggingEnergy || isDraggingEvolution ? '拖到目標寶可夢上' : '' }}
          </div>
        </div>
        <div class="flex gap-3 items-start">
          <!-- Player Active -->
          <div v-if="playerState?.active" class="text-center">
            <div class="text-xs text-blue-500 mb-1">戰鬥</div>
            <div
              class="cursor-pointer rounded-lg transition-all"
              :class="{
                'ring-2 ring-yellow-400': actionMode && actionMode !== 'retreat',
                'ring-2 ring-green-400 ring-dashed scale-105': dragOverZone === 'active' && showPokemonDropHint,
                'ring-1 ring-green-300 ring-dashed': showPokemonDropHint && dragOverZone !== 'active',
              }"
              @click="onActiveCardClick($event)"
              @dragover.prevent
              @dragenter.prevent.stop="onDragEnter('active')"
              @dragleave.stop="onDragLeave($event, 'active')"
              @drop.stop="onDropPokemonTarget($event, playerState!.active!.pokemon.uid)"
            >
              <CardImage :src="playerState.active.pokemon.card?.imageUrl ?? ''" :alt="cardLabel(playerState.active.pokemon)" size="sm" />
            </div>
            <div class="text-xs mt-1">HP {{ pokemonHP(playerState.active) }}</div>
            <div v-if="playerState.active.attachedEnergy.length" class="text-xs text-gray-400">
              能量 ×{{ playerState.active.attachedEnergy.length }}
            </div>
            <div v-if="playerState.active.status.length" class="text-xs text-red-500">
              {{ playerState.active.status.map(s => s.type).join(', ') }}
            </div>
          </div>
          <!-- Player Bench -->
          <div v-for="(bp, idx) in playerState?.bench" :key="bp.pokemon.uid" class="text-center">
            <div class="text-xs text-blue-500 mb-1">後備</div>
            <div
              class="cursor-pointer rounded-lg transition-all"
              :class="{
                'ring-2 ring-yellow-400': actionMode === 'retreat' || actionMode === 'attach_energy' || actionMode === 'evolve',
                'ring-2 ring-green-400 ring-dashed scale-105': dragOverZone === `bench-${idx}` && showPokemonDropHint,
                'ring-1 ring-green-300 ring-dashed': showPokemonDropHint && dragOverZone !== `bench-${idx}`,
              }"
              @click="onFieldCardClick($event, bp)"
              @dragover.prevent
              @dragenter.prevent.stop="onDragEnter(`bench-${idx}`)"
              @dragleave.stop="onDragLeave($event, `bench-${idx}`)"
              @drop.stop="onDropPokemonTarget($event, bp.pokemon.uid)"
            >
              <CardImage :src="bp.pokemon.card?.imageUrl ?? ''" :alt="cardLabel(bp.pokemon)" size="sm" />
            </div>
            <div class="text-xs mt-1">HP {{ pokemonHP(bp) }}</div>
          </div>
          <!-- Player Info -->
          <div class="ml-auto text-right text-xs text-gray-400">
            <div>牌庫 {{ playerState?.deck.length ?? 0 }}</div>
            <div>獎品 {{ isPhysical ? (playerState?.prizesRemaining ?? 0) : (playerState?.prizes.length ?? 0) }}</div>
            <div>棄牌 {{ playerState?.discard.length ?? 0 }}</div>
          </div>
        </div>
      </div>

      <!-- Player Hand -->
      <div class="mb-4">
        <div class="text-sm font-semibold mb-2">手牌 ({{ playerState?.hand.length ?? 0 }})</div>
        <div class="flex gap-2 overflow-x-auto pb-2">
          <div
            v-for="c in playerState?.hand"
            :key="c.uid"
            class="shrink-0 cursor-pointer transition-transform select-none"
            :class="{ 'ring-2 ring-blue-500 rounded-lg -translate-y-2': selectedHandCard === c.uid }"
            :draggable="isPlayerTurn && gamePhase === 'MAIN_PHASE'"
            @click="selectHandCard(c.uid)"
            @dragstart="onDragStart($event, c)"
            @dragend="onDragEnd"
          >
            <CardImage :src="c.card?.imageUrl ?? ''" :alt="cardLabel(c)" size="sm" />
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div v-if="isPlayerTurn && gamePhase === 'MAIN_PHASE'" class="flex flex-wrap gap-2 mb-4">
        <Button
          v-if="isPokemon(selectedCardInfo?.card?.category) && isBasicStage(selectedCardInfo?.card?.stage, selectedCardInfo?.card?.evolveFrom) && canDo('play_pokemon')"
          label="放到後備區"
          size="small"
          @click="playPokemon"
        />
        <Button
          v-if="selectedCardInfo?.card?.category === 'Energy' && canDo('attach_energy')"
          label="貼能量"
          size="small"
          severity="warn"
          @click="actionMode = 'attach_energy'"
        />
        <Button
          v-if="isPokemon(selectedCardInfo?.card?.category) && selectedCardInfo?.card?.evolveFrom && canDo('evolve_pokemon')"
          label="進化"
          size="small"
          severity="info"
          @click="actionMode = 'evolve'"
        />
        <Button
          v-if="selectedCardInfo?.card && ['Supporter', 'Item', 'Stadium'].includes(selectedCardInfo.card.category)"
          label="使用"
          size="small"
          severity="secondary"
          @click="playTrainer"
        />
        <Button
          v-if="canDo('retreat')"
          label="撤退"
          size="small"
          severity="secondary"
          @click="actionMode = 'retreat'"
        />
        <template v-if="canDo('attack') && playerState?.active?.pokemon.card?.attacks">
          <Button
            v-for="(atk, i) in playerState.active.pokemon.card.attacks"
            :key="i"
            :label="`攻擊: ${atk.name} (${atk.damage ?? 0})`"
            size="small"
            severity="danger"
            @click="doAttack(i)"
          />
        </template>
        <Button label="結束回合" size="small" severity="secondary" text @click="endTurn" />
      </div>

      <div v-if="actionMode" class="text-sm text-yellow-600 mb-2">
        {{ actionMode === 'attach_energy' ? '點擊或拖曳到場上的寶可夢來貼能量' : actionMode === 'evolve' ? '點擊要進化的寶可夢' : '點擊後備區的寶可夢來交換' }}
        <Button label="取消" size="small" text @click="actionMode = null" class="ml-2" />
      </div>

      <!-- Drag hint -->
      <div v-if="draggingCard && !actionMode" class="text-xs text-blue-400 mb-2 animate-pulse">
        拖曳到場地放開即可使用卡片
      </div>

      <!-- Physical mode card pool -->
      <div v-if="isPhysical && playerState?.cardPool?.length" class="mb-4">
        <div class="text-sm font-semibold mb-2">卡牌池 ({{ playerState.cardPool.length }})</div>
        <div class="flex gap-2 overflow-x-auto pb-2">
          <div
            v-for="c in playerState.cardPool"
            :key="c.uid"
            class="shrink-0 cursor-pointer"
            @click="manualDraw(c.uid)"
          >
            <CardImage :src="c.card?.imageUrl ?? ''" :alt="cardLabel(c)" size="sm" />
          </div>
        </div>
      </div>

      <!-- Event Log -->
      <div class="border-t pt-3">
        <div class="text-sm font-semibold mb-2">對戰紀錄</div>
        <div class="max-h-40 overflow-y-auto text-xs text-gray-600 space-y-1">
          <div v-for="(evt, i) in eventLog" :key="i">
            {{ evt.message }}
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
