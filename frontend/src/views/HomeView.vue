<script setup lang="ts">
import { useRouter } from 'vue-router'
import Card from 'primevue/card'
import Button from 'primevue/button'

const router = useRouter()
</script>

<template>
  <div class="flex flex-col items-center gap-8 py-12">
    <h1 class="text-3xl font-bold">PTCG Bot</h1>
    <p class="text-lg">寶可夢卡牌 AI 對戰系統</p>

    <div class="flex gap-6 flex-wrap justify-center">
      <Card class="w-72">
        <template #title>對戰</template>
        <template #content>
          <p>與 AI 進行寶可夢卡牌對戰，支援虛擬對戰與實體牌對戰模式。</p>
        </template>
        <template #footer>
          <Button label="開始對戰" @click="router.push('/battle')" />
        </template>
      </Card>

      <Card class="w-72">
        <template #title>卡牌設定</template>
        <template #content>
          <p>管理你的牌庫與 AI 的牌庫，匯入牌組並查看卡片資料。</p>
        </template>
        <template #footer>
          <Button label="管理牌庫" @click="router.push('/cards')" />
        </template>
      </Card>
    </div>
  </div>
</template>
