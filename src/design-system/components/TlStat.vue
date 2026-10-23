<script setup lang="ts">
defineProps<{ label: string; value: string | number }>()
</script>

<template>
  <div class="tl-stat">
    <dt class="tl-stat__label">{{ label }}</dt>
    <dd class="tl-stat__value">{{ value }}</dd>
  </div>
</template>

<style scoped>
.tl-stat {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
  padding: var(--tl-space-4);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-lg);
  background: var(--tl-color-surface);
}
.tl-stat__label {
  /* O rótulo vem antes no HTML (dt/dd), mas aparece embaixo do número. */
  order: 1;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tl-stat__value {
  margin: 0;
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-3xl);
  line-height: 1.1;
}
</style>
