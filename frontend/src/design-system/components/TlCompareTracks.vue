<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { TrackColor } from '../tokens'

export type CompareEvent = { id: string; year: number; title: string }
export type CompareLane = { id: string; name: string; color: TrackColor; events: CompareEvent[] }

/** Duas ou mais timelines em trilhas paralelas no mesmo eixo de anos. */
const props = withDefaults(
  defineProps<{
    lanes: CompareLane[]
    start: number
    end: number
    tickStep?: number
    /** Ano destacado com uma faixa vertical (ex.: 1992). */
    highlightYear?: number
  }>(),
  { tickStep: 5 },
)

const selected = defineModel<string>('selected')

const ROW_H = 70 // cabe um rótulo de até duas linhas
const FIRST_ROW = 44
const LABEL_W = 176 // max-width do rótulo + folga
const MIN_ROWS = 2

// Largura real do eixo, para saber quando dois rótulos se encostam.
const plot = ref<HTMLElement>()
const plotWidth = ref(900)
let observer: ResizeObserver | undefined
onMounted(() => {
  if (!plot.value || typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(([entry]) => (plotWidth.value = entry.contentRect.width || plotWidth.value))
  observer.observe(plot.value)
})
onBeforeUnmount(() => observer?.disconnect())

const pct = (y: number) => ((y - props.start) / Math.max(1, props.end - props.start)) * 100

const ticks = computed(() => {
  const out: number[] = []
  const first = Math.ceil(props.start / props.tickStep) * props.tickStep
  for (let y = first; y <= props.end; y += props.tickStep) out.push(y)
  return out
})

const laneViews = computed(() =>
  props.lanes.map((lane) => {
    // Cada rótulo vai para a primeira linha onde não encosta no anterior.
    const rowEnds: number[] = []
    const events = lane.events
      .filter((e) => e.year >= props.start && e.year <= props.end)
      .sort((a, b) => a.year - b.year)
      .map((e) => {
        const x = pct(e.year)
        const px = (x / 100) * plotWidth.value
        const alignEnd = px + LABEL_W > plotWidth.value
        const from = alignEnd ? px - LABEL_W : px
        let row = rowEnds.findIndex((end) => from >= end)
        if (row === -1) row = rowEnds.length
        rowEnds[row] = from + LABEL_W
        const top = FIRST_ROW + row * ROW_H
        return { ...e, x, top, stem: top - 22, alignEnd }
      })
    const rows = Math.max(MIN_ROWS, rowEnds.length)
    return { ...lane, events, height: `${FIRST_ROW + rows * ROW_H}px` }
  }),
)
</script>

<template>
  <section class="tl-compare" :aria-label="`Comparação de ${start} a ${end}`">
    <div class="tl-compare__inner">
      <div class="tl-compare__row">
        <div></div>
        <div class="tl-compare__ticks" aria-hidden="true">
          <span v-for="t in ticks" :key="t" :style="{ left: `${pct(t)}%` }">{{ t }}</span>
        </div>
      </div>
      <div class="tl-compare__row">
        <div>
          <div v-for="(lane, li) in laneViews" :key="lane.id" class="tl-compare__name" :class="{ 'tl-compare__name--sep': li > 0 }" :style="{ height: lane.height }">
            <span class="tl-compare__swatch" :style="{ background: `var(--tl-track-${lane.color})` }" aria-hidden="true"></span>
            {{ lane.name }}
          </div>
        </div>
        <div ref="plot" class="tl-compare__plot">
          <div v-for="t in ticks" :key="t" class="tl-compare__grid" :style="{ left: `${pct(t)}%` }" aria-hidden="true"></div>
          <div v-if="highlightYear !== undefined" class="tl-compare__highlight" :style="{ left: `${pct(highlightYear)}%` }" aria-hidden="true"></div>
          <div
            v-for="(lane, li) in laneViews"
            :key="lane.id"
            class="tl-compare__lane"
            :class="{ 'tl-compare__lane--sep': li > 0 }"
            :style="{ height: lane.height, '--track': `var(--tl-track-${lane.color})` }"
            role="group"
            :aria-label="lane.name"
          >
            <div class="tl-compare__axis" aria-hidden="true"></div>
            <template v-for="e in lane.events" :key="e.id">
              <div class="tl-compare__stem" :style="{ left: `${e.x}%`, height: `${e.stem}px` }" aria-hidden="true"></div>
              <div class="tl-compare__dot" :class="{ 'tl-compare__dot--on': selected === e.id }" :style="{ left: `${e.x}%` }" aria-hidden="true"></div>
              <button
                type="button"
                class="tl-compare__event"
                :class="{ 'tl-compare__event--on': selected === e.id, 'tl-compare__event--end': e.alignEnd }"
                :style="e.alignEnd ? { right: `${100 - e.x}%`, top: `${e.top}px` } : { left: `${e.x}%`, top: `${e.top}px` }"
                :aria-pressed="selected === e.id"
                @click="selected = e.id"
              >
                <span class="tl-compare__year">{{ e.year }}</span>
                <span class="tl-compare__title" :title="e.title">{{ e.title }}</span>
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.tl-compare {
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: 18px;
  overflow-x: auto;
}
.tl-compare__inner {
  min-width: 920px;
  padding: var(--tl-space-6) var(--tl-space-7) var(--tl-space-5);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
}
.tl-compare__row {
  display: grid;
  grid-template-columns: 150px minmax(0, 1fr);
  gap: var(--tl-space-4);
}
.tl-compare__ticks {
  position: relative;
  height: 24px;
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
.tl-compare__ticks span {
  position: absolute;
  transform: translateX(-50%);
}
.tl-compare__name {
  display: flex;
  align-items: start;
  gap: var(--tl-space-2);
  padding-top: 14px;
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-md);
}
.tl-compare__name--sep {
  border-top: 1px solid var(--tl-color-border-subtle);
}
.tl-compare__swatch {
  flex: none;
  margin-top: 5px;
  width: 10px;
  height: 10px;
  border-radius: var(--tl-radius-pill);
}
.tl-compare__plot {
  position: relative;
}
.tl-compare__grid {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: #f0f1f4;
}
.tl-compare__highlight {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 28px;
  margin-left: -14px;
  border-radius: var(--tl-radius-sm);
  background: rgb(21 23 28 / 0.06);
}
.tl-compare__lane {
  position: relative;
}
.tl-compare__lane--sep {
  border-top: 1px solid var(--tl-color-border-subtle);
}
.tl-compare__axis {
  position: absolute;
  left: 0;
  right: 0;
  top: 22px;
  height: 2px;
  background: var(--track);
  opacity: 0.35;
}
.tl-compare__stem {
  position: absolute;
  top: 22px;
  width: 1px;
  background: var(--track);
  opacity: 0.4;
}
.tl-compare__dot {
  position: absolute;
  top: 16px;
  width: 14px;
  height: 14px;
  margin-left: -7px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-surface);
  border: 2px solid var(--track);
}
.tl-compare__dot--on {
  background: var(--track);
}
.tl-compare__event {
  position: absolute;
  z-index: 1; /* acima das hastes de outros eventos */
  max-width: 170px;
  min-height: 44px;
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  text-align: left;
  border-radius: var(--tl-radius-md);
  border: 1px solid var(--tl-color-border);
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  cursor: pointer;
}
.tl-compare__event--end {
  text-align: right;
}
.tl-compare__event:hover {
  border-color: var(--track);
}
.tl-compare__event--on {
  border-color: var(--track);
  background: color-mix(in srgb, var(--track) 8%, var(--tl-color-surface));
}
.tl-compare__year {
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
.tl-compare__title {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
  line-height: var(--tl-leading-snug);
}
</style>
