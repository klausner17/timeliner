<script setup lang="ts">
import { computed } from 'vue'
import { icons, type IconName } from './icons'

const props = withDefaults(defineProps<{ name: IconName; size?: number; label?: string }>(), {
  size: 16,
})

const icon = computed(() => icons[props.name])
</script>

<template>
  <svg
    class="tl-icon"
    :width="size"
    :height="size"
    :viewBox="icon.viewBox"
    :fill="icon.fill ? 'currentColor' : 'none'"
    :stroke="icon.fill ? 'none' : 'currentColor'"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    :role="label ? 'img' : undefined"
    :aria-label="label"
    :aria-hidden="label ? undefined : 'true'"
  >
    <path v-for="(d, i) in icon.paths" :key="i" :d="d" />
  </svg>
</template>

<style scoped>
.tl-icon {
  flex: none;
}
</style>
