<script setup lang="ts">
import { computed } from 'vue'
import type { TrackColor } from '../tokens'

const props = withDefaults(
  defineProps<{
    name: string
    /** Texto exibido; por padrão a primeira letra do nome, sem @. */
    initial?: string
    size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
    tint?: TrackColor | 'neutral'
    /** Avatar decorativo ao lado do nome escrito: esconde de leitores de tela. */
    decorative?: boolean
  }>(),
  { size: 'md', tint: 'neutral' },
)

const text = computed(() => props.initial ?? props.name.replace(/^@/, '').charAt(0).toUpperCase())
</script>

<template>
  <span
    class="tl-avatar"
    :class="[`tl-avatar--${size}`, `tl-avatar--${tint}`]"
    :title="decorative ? undefined : name"
    :role="decorative ? undefined : 'img'"
    :aria-label="decorative ? undefined : name"
    :aria-hidden="decorative ? 'true' : undefined"
    >{{ text }}</span
  >
</template>

<style scoped>
.tl-avatar {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--tl-radius-pill);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text);
  line-height: 1;
}
.tl-avatar--xs {
  width: 24px;
  height: 24px;
  font-size: var(--tl-text-xs);
}
.tl-avatar--sm {
  width: 36px;
  height: 36px;
  font-size: var(--tl-text-md);
}
.tl-avatar--md {
  width: 44px;
  height: 44px;
  font-size: var(--tl-text-base);
}
.tl-avatar--lg {
  width: 64px;
  height: 64px;
  font-size: var(--tl-text-2xl);
  font-family: var(--tl-font-display);
}
.tl-avatar--xl {
  width: 104px;
  height: 104px;
  font-size: var(--tl-text-5xl);
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-bold);
}
.tl-avatar--neutral {
  background: var(--tl-color-surface-sunken);
  color: var(--tl-color-text-strong);
}
.tl-avatar--a {
  background: var(--tl-track-a-soft);
  color: var(--tl-track-a);
}
.tl-avatar--b {
  background: var(--tl-track-b-soft);
  color: var(--tl-color-warning-strong);
}
.tl-avatar--c {
  background: var(--tl-track-c-soft);
  color: var(--tl-color-success-strong);
}
.tl-avatar--d {
  background: var(--tl-track-d-soft);
  color: var(--tl-track-d);
}
</style>
