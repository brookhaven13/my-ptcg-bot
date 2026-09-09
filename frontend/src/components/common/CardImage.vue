<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{
  src: string
  alt: string
  size?: 'sm' | 'md' | 'lg'
}>()

const hasError = ref(false)

watch(() => props.src, () => {
  hasError.value = false
})

const sizeClass = {
  sm: 'w-20',
  md: 'w-32',
  lg: 'w-48',
}
</script>

<template>
  <div :class="sizeClass[props.size ?? 'md']">
    <img
      v-if="!hasError"
      :src="props.src"
      :alt="props.alt"
      class="w-full rounded-lg shadow"
      loading="lazy"
      @error="hasError = true"
    />
    <div
      v-else
      class="w-full aspect-[2.5/3.5] rounded-lg bg-gray-200 flex items-center justify-center text-xs text-gray-500"
    >
      {{ props.alt }}
    </div>
  </div>
</template>
