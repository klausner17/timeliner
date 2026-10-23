<script setup lang="ts">
import type { TrackColor } from '../tokens'

withDefaults(defineProps<{ color: TrackColor; size?: 'sm' | 'md' }>(), { size: 'md' })
</script>

<template>
  <span class="tl-track-label" :class="`tl-track-label--${size}`">
    <span class="tl-track-label__dot" :style="{ background: `var(--tl-track-${color})` }" aria-hidden="true"></span>
    <slot />
  </span>
</template>

<style scoped>
.tl-track-label {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-weight: var(--tl-weight-semibold);
  min-width: 0;
}
.tl-track-label--sm {
  gap: var(--tl-space-2);
  font-size: var(--tl-text-sm);
}
.tl-track-label__dot {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: var(--tl-radius-pill);
}
</style>
