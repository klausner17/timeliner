<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import type { TrackColor } from '../tokens'
import TlButton from './TlButton.vue'
import { periodDate, periodRange, stackPeriods, type TrackPeriod } from './periodLayout'
import { useYearFormat } from './yearFormat'
import { MIN_SPAN, clampWindow, eventPos, layout, niceStep, posLabel, tickLabel, windowTicks, zoomWindow, type CompareBucket, type CompareEvent, type TrackItem } from './trackLayout'

export type { CompareBucket, CompareEvent, TrackPeriod }
/** Uma trilha traz eventos soltos, faixas com contagem (timelines grandes) ou os dois. */
export type CompareLane = {
  id: string
  name: string
  color: TrackColor
  events?: CompareEvent[]
  buckets?: CompareBucket[]
  /** Primeiro e último ano da timeline. Fora dele, o eixo some e a área fica apagada. */
  extent?: [number, number]
  /** Períodos da timeline (República Velha, Era Vargas...), em faixas acima do eixo. */
  periods?: TrackPeriod[]
}

/**
 * Duas ou mais timelines em trilhas paralelas no mesmo eixo de anos. Rótulos que não
 * cabem viram grupos ("23 eventos"); com `zoomable`, dá para aproximar e arrastar o eixo,
 * e clicar num grupo aproxima naquele trecho.
 */
const props = withDefaults(
  defineProps<{
    lanes: CompareLane[]
    tickStep?: number
    /** Ano destacado com uma faixa vertical (ex.: 1992). */
    highlightYear?: number
    zoomable?: boolean
    /** Limites do zoom e do arraste (ex.: primeiro e último ano das timelines). */
    min?: number
    max?: number
  }>(),
  { zoomable: false },
)

const selected = defineModel<string>('selected')
/** Janela visível em anos. Com v-model:start/end, o pai acompanha o zoom. */
const start = defineModel<number>('start', { required: true })
const end = defineModel<number>('end', { required: true })
const emit = defineEmits<{ zoom: [range: [number, number]] }>()

const ROW_H = 70 // cabe um rótulo de até duas linhas
const FIRST_ROW = 44
const MIN_ROWS = 2
const PERIOD_H = 24 // altura de uma linha de faixas de período
const PERIOD_GAP = 4
const NAME_H = 30 // com mais de uma trilha, o nome da timeline vai em cima de cada uma
const fmt = useYearFormat()

// Largura real do eixo, para saber quando dois rótulos se encostam.
const plot = ref<HTMLElement>()
const plotWidth = ref(900)
let observer: ResizeObserver | undefined
// Em tela estreita, a caixa encolhe e o eixo deixa de ter largura mínima.
const compact = ref(false)
let narrow: MediaQueryList | undefined
const onNarrow = () => (compact.value = !!narrow?.matches)
onMounted(() => {
  narrow = typeof window.matchMedia === 'function' ? window.matchMedia('(max-width: 860px)') : undefined
  onNarrow()
  narrow?.addEventListener?.('change', onNarrow)
})
onBeforeUnmount(() => narrow?.removeEventListener?.('change', onNarrow))
onMounted(() => {
  if (!plot.value || typeof ResizeObserver === 'undefined') return
  observer = new ResizeObserver(([entry]) => (plotWidth.value = entry.contentRect.width || plotWidth.value))
  observer.observe(plot.value)
})
onBeforeUnmount(() => observer?.disconnect())

// Sem coluna de nomes à esquerda: a trilha usa a largura toda. Só quando há mais de uma
// trilha o nome aparece, pequeno, em cima de cada uma; com uma só, o título da página basta.
const showNames = computed(() => props.lanes.length > 1)

const span = computed(() => Math.max(1 / 12, end.value - start.value))
const pct = (y: number) => ((y - start.value) / span.value) * 100
const clampPct = (y: number) => Math.min(100, Math.max(0, pct(y)))
const toPx = (y: number) => (clampPct(y) / 100) * plotWidth.value

const ticks = computed(() => {
  // Cerca de uma marcação a cada 90px: 8 no computador, 3 ou 4 no celular.
  const step = props.tickStep ?? niceStep(span.value, Math.min(8, Math.max(3, Math.floor(plotWidth.value / 90))))
  const f = fmt.value
  const all = f.ticks
    ? f.ticks(start.value, end.value, Math.max(1, step)).map((pos) => ({ pos, label: f.year(pos), month: false }))
    : windowTicks(start.value, end.value, step).map((pos) => ({ pos, label: tickLabel(pos), month: Math.round(pos * 12) % 12 !== 0 }))
  // Calendários com eras põem marcações no começo de cada era; as que encostam na anterior saem.
  const room = (label: string) => label.length * 7 + 12
  const kept: typeof all = []
  for (const t of all) {
    const prev = kept[kept.length - 1]
    if (!prev || toPx(t.pos) - toPx(prev.pos) >= (room(t.label) + room(prev.label)) / 2) kept.push(t)
  }
  return kept
})

function itemsOf(lane: CompareLane): TrackItem[] {
  const items: TrackItem[] = []
  for (const e of lane.events ?? []) {
    const pos = eventPos(e)
    if (pos >= start.value && pos <= end.value) items.push({ kind: 'event', key: e.id, event: e, year: e.year, px: toPx(pos) })
  }
  for (const b of lane.buckets ?? []) {
    if (b.to < start.value || b.from > end.value) continue
    const only = b.count === 1 ? b.sample?.[0] : undefined
    if (only) items.push({ kind: 'event', key: only.id, event: only, year: only.year, px: toPx(eventPos(only)) })
    else items.push({ kind: 'cluster', key: `c${b.from}-${b.to}`, from: b.from, to: b.to, count: b.count, sample: b.sample ?? [], px: toPx((b.from + b.to + 1) / 2) })
  }
  return items.sort((a, b) => a.px - b.px)
}

const laneViews = computed(() =>
  props.lanes.map((lane) => {
    const items = itemsOf(lane)
    const placed = layout(items, plotWidth.value, toPx)
    const rows = Math.max(MIN_ROWS, ...placed.map((p) => p.row + 1))
    const labels = placed.map((p) => {
      const top = FIRST_ROW + p.row * ROW_H
      return { ...p, x: (p.px / plotWidth.value) * 100, top, stem: top - 22 }
    })
    // No eixo, todo evento ganha seu ponto e toda faixa sua barra, mesmo quando o rótulo está num grupo.
    const dots = items.flatMap((i) => (i.kind === 'event' ? [{ key: i.key, id: i.event.id, x: (i.px / plotWidth.value) * 100 }] : []))
    const maxCount = Math.max(1, ...items.map((i) => (i.kind === 'cluster' ? i.count : 0)))
    const bands = items.flatMap((i) =>
      i.kind === 'cluster' ? [{ key: i.key, left: clampPct(i.from), width: Math.max(0.6, clampPct(i.to + 1) - clampPct(i.from)), strength: 0.2 + 0.8 * Math.sqrt(i.count / maxCount) }] : [],
    )
    const periods = periodsOf(lane)
    const nameH = showNames.value ? NAME_H : 0
    const offset = nameH + periods.height
    return { ...lane, labels, dots, bands, edges: edgesOf(lane), periods: periods.items, periodsHeight: periods.height, offset, height: `${offset + FIRST_ROW + rows * ROW_H}px` }
  }),
)

/**
 * As faixas de período da trilha. A altura conta todas as linhas, não só as visíveis,
 * para a trilha não pular de tamanho ao arrastar o eixo.
 */
function periodsOf(lane: CompareLane) {
  if (!lane.periods?.length) return { items: [], height: 0 }
  const placed = stackPeriods(lane.periods, fmt.value.openEnd(end.value))
  const rows = Math.max(...placed.map((p) => p.row + 1))
  const items = placed
    .filter((p) => p.to >= start.value && p.from <= end.value)
    .map((p) => {
      const left = clampPct(p.from)
      return { ...p, key: `${p.name}`, left, width: Math.max(0.4, clampPct(p.to) - left), top: PERIOD_GAP + p.row * PERIOD_H, range: periodRange(p, fmt.value) }
    })
  return { items, height: PERIOD_GAP * 2 + rows * PERIOD_H }
}

function edgesOf(lane: CompareLane) {
  const [first, last] = lane.extent ?? [start.value, end.value]
  const from = clampPct(first)
  const to = clampPct(last)
  return {
    axis: { left: from, width: Math.max(0, to - from) },
    before: from > 0 ? { year: first, x: from } : undefined,
    after: to < 100 ? { year: last, x: to } : undefined,
  }
}

const label = computed(() => `Comparação de ${fmt.value.year(Math.floor(start.value))} a ${fmt.value.year(Math.floor(end.value))}`)
/** "1930–1945", ou "-1813 a -876" quando o traço confundiria com o sinal de menos. */
const rangeText = (from: number, to: number) => {
  if (fmt.value.yearOnly) return from === to ? fmt.value.year(from) : `${fmt.value.year(from)} a ${fmt.value.year(to)}`
  if (from === to) return posLabel(from)
  const [a, b] = [posLabel(from), posLabel(to)]
  return from < 0 || to < 0 || a.includes(' ') || b.includes(' ') ? `${a} a ${b}` : `${a}–${b}`
}
/** Data do evento no rótulo: "1930", "out. 1930" ou "03/10/1930". */
const eventDate = (e: CompareEvent) => periodDate(e.year, e.month, e.day, fmt.value)

// ---------- Zoom ----------

const bounds = computed(() => ({ min: props.min, max: props.max }))
const canZoomOut = computed(() => props.min === undefined || props.max === undefined || start.value > props.min || end.value < props.max)

function setWindow([s, e]: [number, number]) {
  if (s === start.value && e === end.value) return
  start.value = s
  end.value = e
}

const zoomBy = (factor: number, anchor = (start.value + end.value) / 2) => setWindow(zoomWindow(start.value, end.value, factor, anchor, bounds.value))
const showAll = () => props.min !== undefined && props.max !== undefined && setWindow([props.min, props.max])

function openCluster(item: { from: number; to: number }) {
  emit('zoom', [item.from, item.to])
  if (!props.zoomable) return
  const pad = Math.max(1, Math.round((item.to - item.from + 1) * 0.1))
  setWindow(clampWindow(item.from - pad, item.to + 1 + pad, bounds.value))
}

const yearAt = (clientX: number) => {
  const r = plot.value?.getBoundingClientRect()
  return r && r.width ? start.value + ((clientX - r.left) / r.width) * span.value : (start.value + end.value) / 2
}

function onWheel(ev: WheelEvent) {
  if (!props.zoomable) return
  if (ev.ctrlKey || ev.metaKey) {
    // Pinça no trackpad chega como wheel com ctrlKey.
    ev.preventDefault()
    zoomBy(Math.exp(ev.deltaY * 0.003), yearAt(ev.clientX))
  } else if (Math.abs(ev.deltaX) > Math.abs(ev.deltaY)) {
    ev.preventDefault()
    const shift = (ev.deltaX / plotWidth.value) * span.value
    setWindow(clampWindow(start.value + shift, end.value + shift, bounds.value))
  }
}

// Arrastar com um dedo/mouse move o eixo; dois dedos fazem pinça.
const pointers = new Map<number, number>()
let gesture: { start: number; end: number; x: number; dist: number; anchor: number } | undefined
const dragging = ref(false)

function beginGesture() {
  const xs = [...pointers.values()]
  const x = xs.reduce((a, b) => a + b, 0) / xs.length
  gesture = { start: start.value, end: end.value, x, dist: xs.length > 1 ? Math.abs(xs[0] - xs[1]) : 0, anchor: yearAt(x) }
}

function onPointerDown(ev: PointerEvent) {
  if (!props.zoomable || (ev.target as HTMLElement).closest('button')) return
  pointers.set(ev.pointerId, ev.clientX)
  ;(ev.currentTarget as HTMLElement).setPointerCapture?.(ev.pointerId)
  dragging.value = true
  beginGesture()
}

function onPointerMove(ev: PointerEvent) {
  if (!gesture || !pointers.has(ev.pointerId)) return
  pointers.set(ev.pointerId, ev.clientX)
  const xs = [...pointers.values()]
  const g = gesture
  if (xs.length > 1 && g.dist > 0) {
    setWindow(zoomWindow(g.start, g.end, g.dist / Math.max(1, Math.abs(xs[0] - xs[1])), g.anchor, bounds.value))
  } else {
    const shift = -((xs[0] - g.x) / plotWidth.value) * (g.end - g.start)
    setWindow(clampWindow(g.start + shift, g.end + shift, bounds.value))
  }
}

function onPointerUp(ev: PointerEvent) {
  pointers.delete(ev.pointerId)
  if (pointers.size) beginGesture()
  else {
    gesture = undefined
    dragging.value = false
  }
}
</script>

<template>
  <section class="tl-compare" :class="{ 'tl-compare--compact': compact }" :aria-label="label">
    <div v-if="zoomable" class="tl-compare__toolbar">
      <span class="tl-compare__window" aria-live="polite">{{ rangeText(start, end) }}</span>
      <div class="tl-compare__zoom">
        <TlButton icon="minus" icon-only size="sm" label="Afastar" :disabled="!canZoomOut" @click="zoomBy(2)" />
        <TlButton icon="plus" icon-only size="sm" label="Aproximar" :disabled="end - start <= MIN_SPAN + 1e-9" @click="zoomBy(0.5)" />
        <TlButton v-if="min !== undefined && max !== undefined" icon="expand" size="sm" :disabled="!canZoomOut" @click="showAll">Ver tudo</TlButton>
      </div>
    </div>
    <div class="tl-compare__inner">
      <div class="tl-compare__ticks" aria-hidden="true">
        <span v-for="t in ticks" :key="t.pos" :class="{ 'tl-compare__tick--month': t.month }" :style="{ left: `${pct(t.pos)}%` }">{{ t.label }}</span>
      </div>
      <div
        ref="plot"
        class="tl-compare__plot"
        :class="{ 'tl-compare__plot--zoomable': zoomable, 'tl-compare__plot--dragging': dragging }"
        @wheel="onWheel"
        @pointerdown="onPointerDown"
        @pointermove="onPointerMove"
        @pointerup="onPointerUp"
        @pointercancel="onPointerUp"
      >
        <div v-for="t in ticks" :key="t.pos" class="tl-compare__grid" :style="{ left: `${pct(t.pos)}%` }" aria-hidden="true"></div>
        <div v-if="highlightYear !== undefined && highlightYear >= start && highlightYear <= end" class="tl-compare__highlight" :style="{ left: `${pct(highlightYear)}%` }" aria-hidden="true"></div>
        <div
          v-for="(lane, li) in laneViews"
          :key="lane.id"
          class="tl-compare__lane"
          :class="{ 'tl-compare__lane--sep': li > 0 }"
          :style="{ height: lane.height, '--track': `var(--tl-track-${lane.color})` }"
          role="group"
          :aria-label="lane.name"
        >
          <div v-if="showNames" class="tl-compare__lane-name" aria-hidden="true">
            <span class="tl-compare__swatch" :style="{ background: `var(--tl-track-${lane.color})` }"></span>
            <span class="tl-compare__lane-text">{{ lane.name }}</span>
          </div>
          <ul v-if="lane.periods.length" class="tl-compare__periods" :style="{ height: `${lane.periodsHeight}px` }" :aria-label="`Períodos de ${lane.name}`">
            <li
              v-for="p in lane.periods"
              :key="p.key"
              class="tl-compare__period"
              :style="{ left: `${p.left}%`, width: `${p.width}%`, top: `${p.top}px`, '--period': `var(--tl-period-${p.color})`, '--period-soft': `var(--tl-period-${p.color}-soft)` }"
              :title="`${p.name} · ${p.range}`"
            >
              <span class="tl-compare__period-name">{{ p.name }}</span>
              <span class="tl-visually-hidden">, {{ p.range }}</span>
            </li>
          </ul>
          <div class="tl-compare__track" :style="{ top: `${lane.offset}px` }">
            <template v-if="lane.edges.before">
              <div class="tl-compare__void" :style="{ left: 0, width: `${lane.edges.before.x}%` }" aria-hidden="true"></div>
              <div class="tl-compare__cap" :style="{ left: `${lane.edges.before.x}%` }" aria-hidden="true"></div>
              <span class="tl-compare__edge" :style="{ left: `${lane.edges.before.x}%` }">Início em {{ fmt.year(lane.edges.before.year) }}</span>
            </template>
            <template v-if="lane.edges.after">
              <div class="tl-compare__void" :style="{ left: `${lane.edges.after.x}%`, right: 0 }" aria-hidden="true"></div>
              <div class="tl-compare__cap" :style="{ left: `${lane.edges.after.x}%` }" aria-hidden="true"></div>
              <span class="tl-compare__edge" :style="{ right: `${100 - lane.edges.after.x}%` }">Fim em {{ fmt.year(lane.edges.after.year) }}</span>
            </template>
            <div class="tl-compare__axis" :style="{ left: `${lane.edges.axis.left}%`, width: `${lane.edges.axis.width}%` }" aria-hidden="true"></div>
            <div
              v-for="b in lane.bands"
              :key="b.key"
              class="tl-compare__band"
              :style="{ left: `${b.left}%`, width: `${b.width}%`, opacity: b.strength }"
              aria-hidden="true"
            ></div>
            <div
              v-for="d in lane.dots"
              :key="d.key"
              class="tl-compare__dot"
              :class="{ 'tl-compare__dot--on': selected === d.id }"
              :style="{ left: `${d.x}%` }"
              aria-hidden="true"
            ></div>
            <template v-for="e in lane.labels" :key="e.key">
              <div class="tl-compare__stem" :style="{ left: `${e.x}%`, height: `${e.stem}px` }" aria-hidden="true"></div>
              <button
                v-if="e.kind === 'event'"
                type="button"
                class="tl-compare__event"
                :class="{ 'tl-compare__event--on': selected === e.event.id, 'tl-compare__event--end': e.alignEnd }"
                :style="e.alignEnd ? { right: `${100 - e.x}%`, top: `${e.top}px` } : { left: `${e.x}%`, top: `${e.top}px` }"
                :aria-pressed="selected === e.event.id"
                @click="selected = e.event.id"
              >
                <span class="tl-compare__year">{{ eventDate(e.event) }}</span>
                <span class="tl-compare__title" :title="e.event.title">{{ e.event.title }}</span>
              </button>
              <button
                v-else
                type="button"
                class="tl-compare__event tl-compare__event--cluster"
                :class="{ 'tl-compare__event--end': e.alignEnd }"
                :style="e.alignEnd ? { right: `${100 - e.x}%`, top: `${e.top}px` } : { left: `${e.x}%`, top: `${e.top}px` }"
                :aria-label="`${e.count} eventos de ${e.from} a ${e.to}${zoomable ? '. Aproximar' : ''}`"
                :title="e.sample.map((s) => `${fmt.year(s.year)} · ${s.title}`).join('\n')"
                @click="openCluster(e)"
              >
                <span class="tl-compare__year">{{ rangeText(e.from, e.to) }}</span>
                <span class="tl-compare__title">{{ e.count.toLocaleString('pt-BR') }} eventos</span>
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
.tl-compare__toolbar {
  position: sticky;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-3);
  padding: var(--tl-space-4) var(--tl-space-7) 0;
}
.tl-compare__window {
  white-space: nowrap;
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-muted);
  font-variant-numeric: tabular-nums;
}
.tl-compare__zoom {
  display: flex;
  gap: var(--tl-space-1);
}
.tl-compare__inner {
  min-width: 920px;
  padding: var(--tl-space-6) var(--tl-space-7) var(--tl-space-5);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
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
.tl-compare__ticks .tl-compare__tick--month {
  color: var(--tl-color-text-subtle);
}
.tl-compare__swatch {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: var(--tl-radius-pill);
}
.tl-compare__plot {
  position: relative;
  overflow: hidden;
}
.tl-compare__plot--zoomable {
  cursor: grab;
  touch-action: pan-y;
}
.tl-compare__plot--dragging {
  cursor: grabbing;
  user-select: none;
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
.tl-compare__track {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
}
.tl-compare__periods {
  position: relative;
  margin: 0;
  padding: 0;
  list-style: none;
}
.tl-compare__period {
  position: absolute;
  height: 20px;
  padding: 0 8px;
  display: flex;
  align-items: center;
  overflow: hidden;
  border-radius: var(--tl-radius-sm);
  border-left: 3px solid var(--period);
  background: var(--period-soft);
  color: var(--period);
}
.tl-compare__period-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: var(--tl-text-xs);
  font-weight: var(--tl-weight-semibold);
}
.tl-compare__axis {
  position: absolute;
  top: 22px;
  height: 2px;
  background: var(--track);
  opacity: 0.35;
}
.tl-compare__void {
  position: absolute;
  top: 0;
  bottom: 0;
  background: repeating-linear-gradient(-45deg, var(--tl-color-bg) 0 6px, var(--tl-color-surface) 6px 12px);
}
.tl-compare__cap {
  position: absolute;
  top: 12px;
  width: 2px;
  height: 22px;
  margin-left: -1px;
  border-radius: var(--tl-radius-pill);
  background: var(--track);
}
.tl-compare__edge {
  position: absolute;
  top: 0;
  padding: 0 6px;
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
  white-space: nowrap;
}
.tl-compare__band {
  position: absolute;
  top: 19px;
  height: 8px;
  border-radius: var(--tl-radius-pill);
  background: var(--track);
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
.tl-compare__event--cluster {
  border-style: dashed;
  border-color: var(--track);
  background: color-mix(in srgb, var(--track) 6%, var(--tl-color-surface));
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
.tl-compare__lane-name {
  position: relative;
  z-index: 2;
  height: 30px;
  display: flex;
  align-items: center;
  gap: var(--tl-space-2);
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-md);
  pointer-events: none;
}
.tl-compare__lane-text {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  padding-right: var(--tl-space-1);
  background: var(--tl-color-surface);
}
.tl-compare--compact .tl-compare__toolbar {
  flex-wrap: wrap;
  padding: var(--tl-space-3) var(--tl-space-4) 0;
}
.tl-compare--compact .tl-compare__inner {
  min-width: 0;
  padding: var(--tl-space-4) var(--tl-space-4) var(--tl-space-3);
}
</style>
