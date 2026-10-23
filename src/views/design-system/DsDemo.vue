<script setup lang="ts">
defineProps<{ label?: string; stack?: boolean }>()
</script>

<template>
  <div class="ds-demo">
    <div v-if="label" class="ds-demo__label">{{ label }}</div>
    <div class="ds-demo__body" :class="{ 'ds-demo__body--stack': stack }"><slot /></div>
  </div>
</template>

<style scoped>
.ds-demo {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
  padding: var(--tl-space-5);
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-xl);
  min-width: 0;
}
.ds-demo__label {
  font-size: var(--tl-text-xs);
  font-weight: var(--tl-weight-semibold);
  letter-spacing: var(--tl-tracking-eyebrow);
  text-transform: uppercase;
  color: var(--tl-color-text-muted);
}
.ds-demo__body {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-3);
}
.ds-demo__body--stack {
  flex-direction: column;
  align-items: stretch;
}
</style>
