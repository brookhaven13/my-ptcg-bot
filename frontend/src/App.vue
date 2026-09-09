<script setup lang="ts">
import { RouterLink, RouterView, useRoute } from 'vue-router'
import Toolbar from 'primevue/toolbar'
import Button from 'primevue/button'

const route = useRoute()

const isActive = (path: string) => route.path === path
</script>

<template>
  <div class="min-h-screen">
    <Toolbar class="mb-4">
      <template #start>
        <RouterLink to="/" class="font-bold text-lg mr-8">PTCG Bot</RouterLink>
      </template>
      <template #center>
        <div class="flex items-center gap-4">
          <Button
            as="router-link"
            :to="'/'"
            :severity="isActive('/') ? 'contrast' : 'secondary'"
            text
          >
            首頁
          </Button>
          <Button
            as="router-link"
            :to="'/cards'"
            :severity="isActive('/cards') ? 'contrast' : 'secondary'"
            text
          >
            卡牌設定
          </Button>
          <Button
            as="router-link"
            :to="'/battle'"
            :severity="isActive('/battle') ? 'contrast' : 'secondary'"
            text
          >
            對戰
          </Button>
        </div>
      </template>
    </Toolbar>

    <main class="p-4">
      <RouterView />
    </main>
  </div>
</template>
