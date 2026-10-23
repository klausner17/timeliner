<script setup lang="ts">
export type BadgeTone = 'success' | 'info' | 'warning' | 'neutral'

withDefaults(defineProps<{ tone?: BadgeTone; size?: 'sm' | 'md' }>(), { tone: 'neutral', size: 'sm' })
</script>

<template>
  <span class="tl-badge" :class="[`tl-badge--${tone}`, `tl-badge--${size}`]"><slot /></span>
</template>

<style scoped>
.tl-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 10px;
  border-radius: var(--tl-radius-pill);
  font-size: var(--tl-text-xs);
  font-weight: var(--tl-weight-semibold);
  line-height: 1.3;
  white-space: nowrap;
}
.tl-badge--md {
  padding: 6px 12px;
  font-size: var(--tl-text-sm);
}
.tl-badge--success {
  background: var(--tl-color-success-soft);
  color: var(--tl-color-success-strong);
}
.tl-badge--info {
  background: var(--tl-color-primary-soft);
  color: var(--tl-color-primary-hover);
}
.tl-badge--warning {
  background: var(--tl-color-warning-soft);
  color: var(--tl-color-warning-strong);
}
.tl-badge--neutral {
  background: var(--tl-color-neutral-soft);
  color: var(--tl-color-text-body);
}
</style>
