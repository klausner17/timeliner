<script setup lang="ts">
import { computed } from 'vue'
import type { TrackColor } from '../tokens'

/** Minimapa: todos os eventos no eixo completo e a janela visível. */
const props = withDefaults(
  defineProps<{
    start: number
    end: number
    years: number[]
    /** Janela destacada, em anos. */
    window?: [number, number]
    ticks?: number[]
    color?: TrackColor
  }>(),
  { color: 'a' },
)

const span = computed(() => Math.max(1, props.end - props.start))
const pct = (y: number) => ((Math.min(Math.max(y, props.start), props.end) - props.start) / span.value) * 100
const tickList = computed(() => props.ticks ?? [props.start, props.end])
const label = computed(() => `Visão geral: ${props.years.length} eventos de ${props.start} a ${props.end}`)
</script>

<template>
  <div class="tl-overview" :style="{ '--track': `var(--tl-track-${color})` }">
    <div class="tl-overview__ticks" aria-hidden="true">
      <span v-for="t in tickList" :key="t" :style="{ left: `${pct(t)}%` }">{{ t }}</span>
    </div>
    <div class="tl-overview__bar" role="img" :aria-label="label">
      <div class="tl-overview__line"></div>
      <div v-for="(y, i) in years" :key="i" class="tl-overview__mark" :style="{ left: `${pct(y)}%` }"></div>
      <div
        v-if="window"
        class="tl-overview__window"
        :style="{ left: `${pct(window[0])}%`, width: `${pct(window[1]) - pct(window[0])}%` }"
      ></div>
    </div>
  </div>
</template>

<style scoped>
.tl-overview {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
  padding: var(--tl-space-4);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-lg);
  background: var(--tl-color-surface-muted);
}
.tl-overview__ticks {
  position: relative;
  height: 16px;
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
.tl-overview__ticks span {
  position: absolute;
  transform: translateX(-50%);
}
.tl-overview__ticks span:first-child {
  transform: none;
}
.tl-overview__ticks span:last-child {
  transform: translateX(-100%);
}
.tl-overview__bar {
  position: relative;
  height: 36px;
}
.tl-overview__line {
  position: absolute;
  left: 0;
  right: 0;
  top: 17px;
  height: 2px;
  background: var(--tl-color-border-strong);
}
.tl-overview__mark {
  position: absolute;
  top: 12px;
  width: 4px;
  height: 12px;
  margin-left: -2px;
  border-radius: 2px;
  background: var(--track);
}
.tl-overview__window {
  position: absolute;
  top: 0;
  bottom: 0;
  border-radius: var(--tl-radius-sm);
  background: color-mix(in srgb, var(--track) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--track) 45%, transparent);
}
</style>
