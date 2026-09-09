<script setup lang="ts">
import { RouterLink, RouterView } from 'vue-router'
</script>

<template>
  <div class="min-h-screen">
    <nav class="flex items-center gap-4 p-4 border-b">
      <RouterLink to="/" class="font-bold text-lg">PTCG Bot</RouterLink>
      <RouterLink to="/cards">卡牌設定</RouterLink>
      <RouterLink to="/battle">對戰</RouterLink>
    </nav>

    <main class="p-4">
      <RouterView />
    </main>
  </div>
</template>
