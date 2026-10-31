<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TrackColor } from '../tokens'
import { periodRange, stackPeriods, type TrackPeriod } from './periodLayout'

export type OverviewBucket = { from: number; to: number; count: number }
/** Uma timeline no minimapa: anos soltos ou faixas já contadas (timelines grandes), e os períodos dela. */
export type OverviewTrack = { color: TrackColor; years?: number[]; buckets?: OverviewBucket[]; label?: string; periods?: TrackPeriod[] }

/**
 * Minimapa: um histograma de quantos eventos há em cada parte do eixo completo e a
 * janela visível. Com `interactive`, a janela arrasta, muda de tamanho pelas bordas e
 * responde às setas do teclado (v-model:window).
 */
const props = withDefaults(
  defineProps<{
    start: number
    end: number
    years?: number[]
    buckets?: OverviewBucket[]
    /** Períodos, em faixas finas embaixo do histograma. */
    periods?: TrackPeriod[]
    /** Várias timelines empilhadas (tela de comparação). Substitui years/buckets. */
    tracks?: OverviewTrack[]
    ticks?: number[]
    color?: TrackColor
    interactive?: boolean
  }>(),
  { color: 'a', interactive: false },
)

/** Janela destacada, em anos. */
const windowModel = defineModel<[number, number]>('window')

const BINS = 100
const span = computed(() => Math.max(1, props.end - props.start))
const clamp = (y: number) => Math.min(Math.max(y, props.start), props.end)
const pct = (y: number) => ((clamp(y) - props.start) / span.value) * 100
const tickList = computed(() => props.ticks ?? [props.start, props.end])

const trackList = computed<OverviewTrack[]>(() => props.tracks ?? [{ color: props.color, years: props.years, buckets: props.buckets, periods: props.periods }])

const PERIOD_ROW = 6
const THIS_YEAR = new Date().getFullYear()

const histograms = computed(() =>
  trackList.value.map((t) => {
    const bins = new Array<number>(BINS).fill(0)
    const add = (year: number, n: number) => {
      const i = Math.min(BINS - 1, Math.floor(((clamp(year) - props.start) / span.value) * BINS))
      bins[i] += n
    }
    for (const y of t.years ?? []) add(y, 1)
    for (const b of t.buckets ?? []) add((b.from + b.to) / 2, b.count)
    const max = Math.max(1, ...bins)
    const total = bins.reduce((a, b) => a + b, 0)
    // Raiz quadrada: uma década cheia não esconde as faixas com poucos eventos.
    const bars = bins.flatMap((n, i) => (n ? [{ i, h: Math.max(12, Math.sqrt(n / max) * 100) }] : []))
    const placed = stackPeriods(t.periods ?? [], Math.max(THIS_YEAR, props.end)).filter((p) => p.to >= props.start && p.from <= props.end)
    const periods = placed.map((p) => ({ ...p, left: pct(p.from), width: Math.max(0.5, pct(p.to) - pct(p.from)), top: p.row * PERIOD_ROW, title: `${p.name} · ${periodRange(p)}` }))
    const periodsHeight = periods.length ? Math.max(...periods.map((p) => p.row + 1)) * PERIOD_ROW : 0
    return { ...t, bars, total, periods, periodsHeight }
  }),
)

const total = computed(() => histograms.value.reduce((n, h) => n + h.total, 0))
const label = computed(() => `Visão geral: ${total.value.toLocaleString('pt-BR')} eventos de ${props.start} a ${props.end}`)

// ---------- Janela ----------

const bar = ref<HTMLElement>()
let drag: { mode: 'move' | 'from' | 'to'; x: number; win: [number, number] } | undefined

function setWindow(lo: number, hi: number) {
  const w = Math.max(1, Math.round(hi - lo))
  let s = Math.round(lo)
  s = Math.min(Math.max(s, props.start), props.end - Math.min(w, span.value))
  windowModel.value = [s, s + Math.min(w, span.value)]
}

const yearsPerPx = () => span.value / (bar.value?.getBoundingClientRect().width || 1)
const yearAt = (clientX: number) => {
  const r = bar.value?.getBoundingClientRect()
  return r && r.width ? props.start + ((clientX - r.left) / r.width) * span.value : props.start
}

function onPointerDown(ev: PointerEvent) {
  if (!props.interactive) return
  const target = ev.target as HTMLElement
  const edge = target.dataset.edge as 'from' | 'to' | undefined
  let win = windowModel.value ?? [props.start, props.end]
  if (!edge && !target.closest('.tl-overview__window')) {
    // Clique fora da janela: centraliza nela o ponto clicado e já começa a arrastar.
    const w = win[1] - win[0]
    setWindow(yearAt(ev.clientX) - w / 2, yearAt(ev.clientX) + w / 2)
    win = windowModel.value ?? win
  }
  drag = { mode: edge ?? 'move', x: ev.clientX, win: [...win] }
  ;(ev.currentTarget as HTMLElement).setPointerCapture?.(ev.pointerId)
  ev.preventDefault()
}

function onPointerMove(ev: PointerEvent) {
  if (!drag) return
  const dy = (ev.clientX - drag.x) * yearsPerPx()
  const [lo, hi] = drag.win
  if (drag.mode === 'move') setWindow(lo + dy, hi + dy)
  else if (drag.mode === 'from') windowModel.value = [Math.round(Math.min(Math.max(props.start, lo + dy), hi - 2)), hi]
  else windowModel.value = [lo, Math.round(Math.max(Math.min(props.end, hi + dy), lo + 2))]
}

const onPointerUp = () => (drag = undefined)

function onKey(ev: KeyboardEvent) {
  const win = windowModel.value
  if (!props.interactive || !win) return
  const w = win[1] - win[0]
  const step = Math.max(1, Math.round(w * (ev.shiftKey ? 0.5 : 0.1)))
  const mid = (win[0] + win[1]) / 2
  const actions: Record<string, () => void> = {
    ArrowLeft: () => setWindow(win[0] - step, win[1] - step),
    ArrowRight: () => setWindow(win[0] + step, win[1] + step),
    Home: () => setWindow(props.start, props.start + w),
    End: () => setWindow(props.end - w, props.end),
    '+': () => setWindow(mid - w / 4, mid + w / 4),
    '-': () => setWindow(mid - w, mid + w),
  }
  const act = actions[ev.key]
  if (act) {
    ev.preventDefault()
    act()
  }
}
</script>

<template>
  <div class="tl-overview" :class="{ 'tl-overview--interactive': interactive }" :style="{ '--track': `var(--tl-track-${color})` }">
    <div class="tl-overview__ticks" aria-hidden="true">
      <span v-for="t in tickList" :key="t" :style="{ left: `${pct(t)}%` }">{{ t }}</span>
    </div>
    <div
      ref="bar"
      class="tl-overview__bar"
      :class="{ 'tl-overview__bar--stacked': trackList.length > 1 }"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
    >
      <div class="tl-overview__tracks" role="img" :aria-label="label">
        <template v-for="(h, hi) in histograms" :key="hi">
          <div class="tl-overview__track" :style="{ '--track': `var(--tl-track-${h.color})` }" :title="h.label">
            <div class="tl-overview__line"></div>
            <div v-for="b in h.bars" :key="b.i" class="tl-overview__mark" :style="{ left: `${b.i}%`, height: `${b.h}%` }"></div>
          </div>
          <div v-if="h.periods.length" class="tl-overview__periods" :style="{ height: `${h.periodsHeight}px` }">
            <div
              v-for="p in h.periods"
              :key="p.name"
              class="tl-overview__period"
              :style="{ left: `${p.left}%`, width: `${p.width}%`, top: `${p.top}px`, background: `var(--tl-period-${p.color})` }"
              :title="p.title"
            ></div>
          </div>
        </template>
      </div>
      <div
        v-if="windowModel"
        class="tl-overview__window"
        :style="{ left: `${pct(windowModel[0])}%`, width: `${pct(windowModel[1]) - pct(windowModel[0])}%` }"
        :role="interactive ? 'slider' : undefined"
        :tabindex="interactive ? 0 : undefined"
        :aria-label="interactive ? 'Trecho visível' : undefined"
        :aria-valuemin="interactive ? start : undefined"
        :aria-valuemax="interactive ? end : undefined"
        :aria-valuenow="interactive ? windowModel[0] : undefined"
        :aria-valuetext="interactive ? `${windowModel[0]} a ${windowModel[1]}` : undefined"
        @keydown="onKey"
      >
        <template v-if="interactive">
          <span class="tl-overview__edge tl-overview__edge--from" data-edge="from" aria-hidden="true"></span>
          <span class="tl-overview__edge tl-overview__edge--to" data-edge="to" aria-hidden="true"></span>
        </template>
      </div>
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
}
.tl-overview--interactive .tl-overview__bar {
  cursor: pointer;
  touch-action: pan-y;
}
.tl-overview__tracks {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.tl-overview__track {
  position: relative;
  flex: 1;
  min-height: 14px;
}
.tl-overview__bar:not(.tl-overview__bar--stacked) .tl-overview__track {
  flex: none;
  height: 36px;
  padding: 4px 0;
}
.tl-overview__periods {
  position: relative;
  margin-top: -1px;
}
.tl-overview__period {
  position: absolute;
  height: 4px;
  border-radius: var(--tl-radius-pill);
  opacity: 0.85;
}
.tl-overview__line {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 2px;
  background: var(--tl-color-border-strong);
}
.tl-overview__mark {
  position: absolute;
  bottom: 0;
  width: 1%;
  min-width: 2px;
  border-radius: 2px 2px 0 0;
  background: var(--track);
}
.tl-overview__window {
  position: absolute;
  top: -2px;
  bottom: -2px;
  border-radius: var(--tl-radius-sm);
  background: color-mix(in srgb, var(--track) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--track) 45%, transparent);
}
.tl-overview--interactive .tl-overview__window {
  cursor: grab;
}
.tl-overview__window:focus-visible {
  outline: 2px solid var(--track);
  outline-offset: 2px;
}
.tl-overview__edge {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 10px;
  cursor: ew-resize;
}
.tl-overview__edge--from {
  left: -5px;
}
.tl-overview__edge--to {
  right: -5px;
}
</style>
