<script setup lang="ts">
withDefaults(defineProps<{ size?: number; wordmark?: boolean }>(), { size: 30, wordmark: true })
</script>

<template>
  <span class="tl-logo">
    <svg :width="size" :height="size" viewBox="0 0 30 30" aria-hidden="true">
      <line x1="2" y1="15" x2="28" y2="15" stroke="var(--tl-color-text)" stroke-width="2" />
      <circle cx="7" cy="15" r="4" fill="var(--tl-track-a)" />
      <circle cx="15" cy="15" r="4" fill="var(--tl-color-surface)" stroke="var(--tl-color-text)" stroke-width="2" />
      <circle cx="23" cy="15" r="4" fill="var(--tl-track-b)" />
    </svg>
    <span v-if="wordmark">Timeliner</span>
  </span>
</template>

<style scoped>
.tl-logo {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-bold);
  font-size: var(--tl-text-xl);
  color: var(--tl-color-text);
}
</style>
