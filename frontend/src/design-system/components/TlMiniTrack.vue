<script setup lang="ts">
import type { TrackColor } from '../tokens'

/** Faixa decorativa com pontos, usada no topo dos cartões de timeline. */
defineProps<{ color: TrackColor; points: number[] }>()
</script>

<template>
  <div class="tl-mini-track" :style="{ '--track': `var(--tl-track-${color})`, '--track-soft': `var(--tl-track-${color}-soft)` }" aria-hidden="true">
    <div class="tl-mini-track__line"></div>
    <div v-for="(p, i) in points" :key="i" class="tl-mini-track__dot" :style="{ left: `calc(12px + (100% - 36px) * ${p / 100})` }"></div>
  </div>
</template>

<style scoped>
.tl-mini-track {
  position: relative;
  height: 40px;
  border-radius: var(--tl-radius-md);
  background: var(--track-soft);
}
.tl-mini-track__line {
  position: absolute;
  left: 12px;
  right: 12px;
  top: 19px;
  height: 2px;
  background: var(--track);
  opacity: 0.5;
}
.tl-mini-track__dot {
  position: absolute;
  top: 14px;
  width: 12px;
  height: 12px;
  border-radius: var(--tl-radius-pill);
  background: var(--track);
}
</style>
