<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  TlButton,
  TlCard,
  TlCheckbox,
  TlCompareTracks,
  TlInput,
  TlOverviewBar,
  TlSegmented,
  TlSelect,
  TlTrackLabel,
  axisTicks,
  trackColors,
  type CompareEvent,
  type CompareLane,
  type OverviewTrack,
  type TrackColor,
  type TrackPeriod,
} from '@/design-system'
import { errorMessage } from '@/data/api'
import { currentUser, listSavedComparisons, listTimelines, loadComparison, loadSummary, saveComparison, yearRange, type Comparison } from '@/data/store'
import type { SavedComparison, Timeline, TimelineEvent } from '@/data/types'
import { extent, type EventWindow } from '@/data/windowing'
import { copyLink, dateLabel } from '@/lib/format'
import { useLatest } from '@/lib/useLatest'

const route = useRoute()
const router = useRouter()

const KEYS = ['a', 'b', 'c', 'd'] as const

// Timelines que aparecem nos seletores: as públicas e as do usuário.
const { data: choiceData, error: choiceError } = useLatest(
  () => {
    const me = currentUser.value?.handle
    return Promise.all([listTimelines({ pageSize: 100 }), me ? listTimelines({ owner: 'me', pageSize: 100 }) : undefined]).then(([pub, own]) => ({ pub: pub.items, own: own?.items ?? [] }))
  },
  undefined as { pub: Timeline[]; own: Timeline[] } | undefined,
)
const choices = computed(() => {
  if (!choiceData.value) return []
  const { pub, own } = choiceData.value
  return [...pub, ...own.filter((t) => t.eventCount && !pub.some((p) => p.slug === t.slug))]
})
const findTimeline = (slug: string) => choices.value.find((t) => t.slug === slug)
const options = computed(() => choices.value.map((t) => ({ value: t.slug, label: t.title })))

const slugs = computed<string[]>(() => {
  const picked = KEYS.map((k) => route.query[k]).filter((v): v is string => typeof v === 'string' && !!findTimeline(v))
  const out = [...picked]
  // Sempre pelo menos duas: completa com as primeiras que ainda não estão na comparação.
  for (const t of choices.value) {
    if (out.length >= 2) break
    if (!out.includes(t.slug)) out.push(t.slug)
  }
  return out
})
const selected = computed(() => slugs.value.map((s) => findTimeline(s)).filter((t): t is Timeline => !!t))

function setSlugs(next: string[], extra: Record<string, string | undefined> = {}) {
  const q: Record<string, string | undefined> = { de: route.query.de as string | undefined, ate: route.query.ate as string | undefined, ...extra }
  KEYS.forEach((k, i) => (q[k] = next[i]))
  router.replace({ query: q })
}

function pick(i: number, slug: string) {
  const next = [...slugs.value]
  const other = next.indexOf(slug)
  if (other !== -1 && other !== i) next[other] = next[i] // escolher uma que já está troca as duas de lugar
  next[i] = slug
  setSlugs(next)
}
const swap = () => setSlugs([slugs.value[1], slugs.value[0], ...slugs.value.slice(2)])
function addLane() {
  const free = choices.value.find((t) => !slugs.value.includes(t.slug))
  if (free) setSlugs([...slugs.value, free.slug])
}
const removeLane = (i: number) => setSlugs(slugs.value.filter((_, j) => j !== i))

// ---------- Período ----------

const defaultRange = computed<[number, number]>(() => {
  const ranges = selected.value.map(yearRange).filter((r): r is [number, number] => !!r)
  if (!ranges.length) return [1900, 2020]
  const lo = Math.max(...ranges.map((r) => r[0]))
  const hi = Math.min(...ranges.map((r) => r[1]))
  // Com anos em comum, mostra só a sobreposição; sem, mostra tudo.
  const [s, e] = lo < hi ? [lo, hi] : [Math.min(...ranges.map((r) => r[0])), Math.max(...ranges.map((r) => r[1]))]
  return [Math.floor(s / 5) * 5, Math.ceil(e / 5) * 5]
})
const routeStart = computed(() => Number(route.query.de) || defaultRange.value[0])
const routeEnd = computed(() => Math.max(routeStart.value + 1 / 12, Number(route.query.ate) || defaultRange.value[1]))
/** Ano na URL: inteiro, ou com fração quando o zoom está em meses. */
const urlPos = (x: number) => (Number.isInteger(x) ? String(x) : x.toFixed(4))

// A janela segue a URL, e o zoom nas trilhas ou no minimapa volta para a URL logo depois.
const view = ref<[number, number]>([routeStart.value, routeEnd.value])
watch([routeStart, routeEnd], ([s, e]) => {
  if (s !== view.value[0] || e !== view.value[1]) view.value = [s, e]
})
let urlTimer: ReturnType<typeof setTimeout> | undefined
watch(view, ([s, e]) => {
  clearTimeout(urlTimer)
  if (s === routeStart.value && e === routeEnd.value) return
  urlTimer = setTimeout(() => setSlugs(slugs.value, { de: urlPos(s), ate: urlPos(e) }), 400)
})
onBeforeUnmount(() => clearTimeout(urlTimer))
const start = computed({ get: () => view.value[0], set: (v) => (view.value = [v, view.value[1]]) })
const end = computed({ get: () => view.value[1], set: (v) => (view.value = [view.value[0], v]) })

/** Limites do zoom: do primeiro ao último ano de todas as timelines escolhidas. */
const bounds = computed<[number, number]>(() => {
  const ranges = selected.value.map(yearRange).filter((r): r is [number, number] => !!r)
  const lo = extent(ranges.map((r) => r[0]))
  const hi = extent(ranges.map((r) => r[1]))
  // Arredonda como defaultRange e inclui um período digitado fora das timelines.
  if (!lo || !hi) return [routeStart.value, routeEnd.value]
  return [Math.min(Math.floor(lo[0] / 5) * 5, routeStart.value), Math.max(Math.ceil(hi[1] / 5) * 5, routeEnd.value)]
})

// Os campos mostram anos inteiros; com zoom em meses, os anos que cobrem a janela.
const startInput = ref(String(Math.floor(start.value)))
const endInput = ref(String(Math.ceil(end.value)))
watch(view, ([s, e]) => {
  startInput.value = String(Math.floor(s))
  endInput.value = String(Math.ceil(e))
})
function applyRange() {
  const s = Number(startInput.value)
  const e = Number(endInput.value)
  if (!Number.isInteger(s) || !Number.isInteger(e) || e <= s) {
    startInput.value = String(Math.floor(start.value))
    endInput.value = String(Math.ceil(end.value))
    return
  }
  if (s === Math.floor(start.value) && e === Math.ceil(end.value)) return
  view.value = [s, e]
  setSlugs(slugs.value, { de: String(s), ate: String(e) })
}
const resetRange = () => setSlugs(slugs.value, { de: undefined, ate: undefined })

// ---------- Trilhas e lista ----------

const mode = ref('trilhas')
const colorOf = (i: number): TrackColor => trackColors[i % trackColors.length]

// Períodos de cada timeline (República Velha, Era Vargas...) em faixas acima de cada trilha.
const showPeriods = ref(true)
const hasPeriods = computed(() => selected.value.some((t) => t.eras.length))
const periodsOf = (t: Timeline | undefined): TrackPeriod[] | undefined => (showPeriods.value && t?.eras.length ? t.eras : undefined)

// Cada trilha vem como na API: os eventos do trecho, ou faixas quando são muitos.
const { data: comparison } = useLatest(
  () => (selected.value.length >= 2 ? loadComparison(slugs.value, Math.floor(view.value[0]), Math.ceil(view.value[1]), { buckets: 60 }) : undefined),
  undefined as Comparison | undefined,
)
const windows = computed(() => comparison.value?.lanes ?? [])
const bucketed = computed(() => windows.value.some((w) => w.mode === 'faixas'))

const toCompare = (e: TimelineEvent): CompareEvent => ({ id: e.id, year: e.year, month: e.month, day: e.day, title: e.title })
const lanes = computed<CompareLane[]>(() =>
  selected.value.map((t, i) => {
    const w = windows.value[i]
    return {
      id: t.slug,
      name: t.title,
      color: colorOf(i),
      extent: yearRange(t),
      periods: periodsOf(t),
      events: w?.events.map(toCompare) ?? [],
      buckets: w?.buckets.map((b) => ({ ...b, sample: b.sample.map(toCompare) })) ?? [],
    }
  }),
)

/** Os eventos que vieram: todos, ou os exemplos de cada faixa. */
const shown = (w: EventWindow | undefined) => (w ? [...w.events, ...w.buckets.flatMap((b) => b.sample)].sort((a, b) => a.year - b.year) : [])
const allEvents = computed(() => selected.value.flatMap((t, i) => shown(windows.value[i]).map((e) => ({ ...e, timeline: t, color: colorOf(i) }))))
const selectedId = ref<string>()
const current = computed(() => allEvents.value.find((e) => e.id === selectedId.value) ?? allEvents.value[0])

const ROWS_STEP = 20
const rowsShown = ref(ROWS_STEP)
watch([view, slugs], () => (rowsShown.value = ROWS_STEP))
const years = computed(() => [...new Set(allEvents.value.map((e) => e.year))].sort((a, b) => a - b))
const rows = computed(() => {
  const byLane = selected.value.map((_, i) => shown(windows.value[i]))
  return years.value.map((y) => {
    const cells = byLane.map((evs) => evs.filter((e) => e.year === y))
    return { year: y, cells, shared: cells.filter((c) => c.length).length > 1 }
  })
})
const visibleRows = computed(() => rows.value.slice(0, rowsShown.value))

/** Anos em que duas ou mais timelines têm eventos (vem pronto, com teto, da comparação). */
const coincidences = computed(() => comparison.value?.overlaps ?? [])
const highlight = computed(() => {
  const y = current.value?.year
  return coincidences.value.find((o) => o.year === y) ?? coincidences.value[0]
})
const highlightText = computed(() => {
  const h = highlight.value
  if (!h) return ''
  const parts = allEvents.value.filter((e) => e.year === h.year && h.laneIds.includes(e.timeline.slug)).map((e) => e.title)
  const names = h.laneIds.map((id) => findTimeline(id)?.title ?? id)
  const and = (xs: string[]) => (xs.length > 1 ? `${xs.slice(0, -1).join(', ')} e ${xs[xs.length - 1]}` : xs[0])
  return parts.length > 1 ? `Em ${h.year}, ${and(parts)} aparecem no mesmo ano.` : `Em ${h.year}, ${and(names)} têm eventos no mesmo ano.`
})
const coincidenceCount = computed(() => {
  const n = comparison.value?.overlapsTotal ?? 0
  return n > 1 ? `${n.toLocaleString('pt-BR')} anos com eventos em mais de uma timeline neste período.` : ''
})

// Minimapa com as timelines empilhadas sobre o eixo completo.
const { data: overviewData } = useLatest(
  () => Promise.all(selected.value.map((t) => loadSummary(t.slug, { from: bounds.value[0], to: bounds.value[1], buckets: 100, maxEvents: 1 }))),
  [] as EventWindow[],
)
const overviewTracks = computed<OverviewTrack[]>(() =>
  overviewData.value.map((w, i) => ({
    color: colorOf(i),
    label: selected.value[i]?.title,
    periods: periodsOf(selected.value[i]),
    buckets: w.mode === 'faixas' ? w.buckets : w.events.map((e) => ({ from: e.year, to: e.year, count: 1 })),
  })),
)
const overviewTicks = computed(() => axisTicks(bounds.value[0], bounds.value[1]))
const overviewWindow = computed<[number, number] | undefined>({ get: () => view.value, set: (v) => v && (view.value = v) })

// ---------- Salvar e compartilhar ----------

const savedList = ref<SavedComparison[]>([])
watch(
  () => currentUser.value?.handle,
  async (me) => {
    savedList.value = me ? await listSavedComparisons().catch(() => []) : []
  },
  { immediate: true },
)
const sameSlugs = (a: string[], b: string[]) => a.length === b.length && a.every((s, i) => s === b[i])
const saved = computed(() => savedList.value.some((c) => sameSlugs(c.timelines, slugs.value) && c.start === Math.floor(start.value) && c.end === Math.ceil(end.value)))
const saving = ref(false)
const shareMsg = ref('')
async function save() {
  if (!currentUser.value) {
    router.push({ path: '/entrar', query: { next: route.fullPath } })
    return
  }
  saving.value = true
  try {
    savedList.value = [await saveComparison(slugs.value, Math.floor(start.value), Math.ceil(end.value)), ...savedList.value]
  } catch (e) {
    shareMsg.value = errorMessage(e, 'Não deu para salvar a comparação.')
  } finally {
    saving.value = false
  }
}
async function share() {
  const url = new URL(router.resolve({ query: { ...route.query, a: slugs.value[0], b: slugs.value[1], de: urlPos(start.value), ate: urlPos(end.value) } }).href, window.location.origin)
  shareMsg.value = (await copyLink(url.href)) ? 'Link copiado' : 'Não deu para copiar o link'
  setTimeout(() => (shareMsg.value = ''), 2500)
}
</script>

<template>
  <main class="tl-container page">
    <div class="page__head">
      <h1 class="page__title">Comparar timelines</h1>
      <p class="page__lead">Coloque duas histórias no mesmo eixo de tempo e veja o que aconteceu junto.</p>
    </div>

    <p v-if="choiceError" class="page__lead" role="alert">{{ choiceError }}</p>
    <div v-else-if="!choiceData" aria-busy="true"></div>
    <TlCard v-else-if="choices.length < 2" tone="muted" padding="lg">Ainda não há timelines suficientes para comparar. Volte quando houver pelo menos duas publicadas.</TlCard>
    <template v-else>

    <div class="pickers">
      <template v-for="(t, i) in selected" :key="i">
        <div class="picker" :style="{ '--track': `var(--tl-track-${colorOf(i)})` }">
          <label :for="`tl-${i}`" class="picker__label">Timeline {{ KEYS[i].toUpperCase() }}</label>
          <div class="picker__box">
            <span class="picker__dot" aria-hidden="true"></span>
            <TlSelect :id="`tl-${i}`" :model-value="t.slug" :options="options" class="picker__select" @update:model-value="(v) => v && pick(i, v)" />
            <TlButton v-if="i > 1" icon="close" icon-only size="sm" variant="ghost" :label="`Remover timeline ${KEYS[i].toUpperCase()}`" @click="removeLane(i)" />
          </div>
        </div>
        <TlButton v-if="i === 0" icon="swap" icon-only size="lg" label="Trocar A e B" class="pickers__swap" @click="swap" />
      </template>
      <TlButton v-if="selected.length < 4 && choices.length > selected.length" variant="dashed" shape="rounded" size="lg" @click="addLane">+ Adicionar timeline</TlButton>
    </div>

    <div class="controls">
      <form class="range" @submit.prevent="applyRange">
        <label for="de" class="range__label">Período</label>
        <TlInput id="de" v-model="startInput" inputmode="numeric" aria-label="Ano inicial" class="range__input" @blur="applyRange" />
        <span aria-hidden="true" class="range__label">até</span>
        <TlInput v-model="endInput" inputmode="numeric" aria-label="Ano final" class="range__input" @blur="applyRange" />
        <TlButton v-if="route.query.de || route.query.ate" variant="ghost" size="sm" @click="resetRange">Período sugerido</TlButton>
      </form>
      <TlCheckbox v-if="hasPeriods" v-model="showPeriods" label="Mostrar períodos" />
      <TlSegmented
        v-model="mode"
        label="Modo de comparação"
        :options="[
          { value: 'trilhas', label: 'Trilhas' },
          { value: 'lado', label: 'Lado a lado' },
        ]"
      />
    </div>

    <TlOverviewBar v-model:window="overviewWindow" :start="bounds[0]" :end="bounds[1]" :tracks="overviewTracks" :ticks="overviewTicks" interactive />

    <TlCompareTracks
      v-if="mode === 'trilhas'"
      v-model:start="start"
      v-model:end="end"
      :selected="current?.id"
      :lanes="lanes"
      :highlight-year="highlight?.year"
      :min="bounds[0]"
      :max="bounds[1]"
      zoomable
      @update:selected="(v) => (selectedId = v)"
    />

    <TlCard v-else padding="sm" class="list" :class="{ 'list--many': selected.length > 2 }">
      <div class="list__row list__row--head" aria-hidden="true">
        <template v-if="selected.length === 2">
          <TlTrackLabel color="a" size="sm" class="list__a">{{ selected[0].title }}</TlTrackLabel>
          <span></span>
          <TlTrackLabel color="b" size="sm">{{ selected[1].title }}</TlTrackLabel>
        </template>
        <template v-else>
          <span></span>
          <TlTrackLabel v-for="(t, i) in selected" :key="t.slug" :color="colorOf(i)" size="sm">{{ t.title }}</TlTrackLabel>
        </template>
      </div>
      <p v-if="bucketed" class="list__note">Há muitos eventos neste período: aparecem só os primeiros de cada trecho. Diminua o período para ver todos.</p>
      <div v-for="r in visibleRows" :key="r.year" class="list__row">
        <template v-if="selected.length === 2">
          <div class="list__cell list__a">{{ r.cells[0].map((e) => e.title).join(' · ') }}</div>
          <div class="list__year" :class="{ 'list__year--shared': r.shared }">{{ r.year }}</div>
          <div class="list__cell">{{ r.cells[1].map((e) => e.title).join(' · ') }}</div>
        </template>
        <template v-else>
          <div class="list__year" :class="{ 'list__year--shared': r.shared }">{{ r.year }}</div>
          <div v-for="(c, i) in r.cells" :key="i" class="list__cell">{{ c.map((e) => e.title).join(' · ') }}</div>
        </template>
      </div>
      <p v-if="!rows.length" class="list__empty">Nenhum evento entre {{ start }} e {{ end }}.</p>
      <div v-if="rows.length > rowsShown" class="list__more">
        <TlButton size="sm" @click="rowsShown += ROWS_STEP">Ver mais ({{ (rows.length - rowsShown).toLocaleString('pt-BR') }} anos)</TlButton>
      </div>
    </TlCard>

    <div class="bottom">
      <TlCard v-if="current" class="detail" padding="lg">
        <TlTrackLabel :color="current.color" size="sm" class="detail__meta">{{ current.timeline.title }} · {{ dateLabel(current) }}</TlTrackLabel>
        <h2 class="detail__title">{{ current.title }}</h2>
        <p class="detail__desc">{{ current.description }}</p>
        <RouterLink :to="`/t/${current.timeline.slug}`" class="detail__link">Abrir na timeline</RouterLink>
      </TlCard>
      <TlCard tone="inverse" padding="lg" class="match">
        <h2 class="match__label">Coincidência destacada</h2>
        <p class="match__text">{{ highlightText || 'Nenhum ano com eventos nas duas timelines neste período.' }}</p>
        <p v-if="coincidenceCount" class="match__count">{{ coincidenceCount }}</p>
        <div class="match__actions">
          <TlButton variant="inverse" :disabled="saved || saving" @click="save">{{ saved ? 'Comparação salva' : 'Salvar comparação' }}</TlButton>
          <TlButton variant="ghost" class="match__ghost" @click="share">Compartilhar link</TlButton>
          <span class="match__toast" role="status">{{ shareMsg }}</span>
        </div>
      </TlCard>
    </div>
    </template>
  </main>
</template>

<style scoped>
.page {
  padding-top: var(--tl-space-7);
  padding-bottom: calc(var(--tl-space-8) * 2);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-6);
}
.page__head {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.page__title {
  font-size: var(--tl-text-4xl);
  letter-spacing: var(--tl-tracking-tight);
}
.page__lead {
  font-size: 16px;
  color: var(--tl-color-text-muted);
}
.pickers {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  gap: var(--tl-space-3);
}
.picker {
  flex: 1 1 300px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.picker__label {
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-muted);
}
.picker__box {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: var(--tl-control-lg);
  padding: 0 6px 0 14px;
  background: var(--tl-color-surface);
  border: 1.5px solid var(--track);
  border-radius: 12px;
}
.picker__dot {
  flex: none;
  width: 12px;
  height: 12px;
  border-radius: var(--tl-radius-pill);
  background: var(--track);
}
.picker__select {
  flex: 1;
  min-width: 0;
  border: 0;
  background-color: transparent;
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  padding-left: 0;
}
.controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-3);
}
.range {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.range__label {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.range__input {
  width: 84px;
}
.list {
  display: flex;
  flex-direction: column;
  padding-left: var(--tl-space-6);
  padding-right: var(--tl-space-6);
  overflow-x: auto;
}
.list__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 72px minmax(0, 1fr);
  gap: var(--tl-space-4);
  align-items: center;
  padding: var(--tl-space-3) 0;
  border-bottom: 1px solid var(--tl-color-border-subtle);
}
.list--many .list__row {
  grid-template-columns: 72px repeat(v-bind('selected.length'), minmax(160px, 1fr));
}
.list__row--head {
  font-weight: var(--tl-weight-semibold);
}
.list__a {
  text-align: right;
  justify-content: end;
}
.list__cell {
  font-size: var(--tl-text-base);
  font-weight: var(--tl-weight-medium);
}
.list__year {
  text-align: center;
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-semibold);
  font-size: 16px;
  padding: 6px 0;
  border-radius: var(--tl-radius-pill);
}
.list__year--shared {
  background: var(--tl-color-surface-sunken);
}
.list__note {
  padding: var(--tl-space-3) var(--tl-space-4);
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.list__more {
  display: flex;
  justify-content: center;
  padding: var(--tl-space-3);
}
.match__count {
  font-size: var(--tl-text-sm);
  opacity: 0.75;
}
.list__empty {
  padding: var(--tl-space-4) 0;
  color: var(--tl-color-text-muted);
}
.bottom {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-5);
  align-items: stretch;
}
.detail {
  flex: 2 1 460px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail__meta {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.detail__title {
  font-size: var(--tl-text-2xl);
}
.detail__desc {
  color: var(--tl-color-text-body);
}
.detail__link {
  width: fit-content;
  padding: 6px 0;
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-md);
}
.match {
  flex: 1 1 300px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.match__label {
  font-family: var(--tl-font-body);
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-inverse-muted);
  letter-spacing: 0;
}
.match__text {
  font-family: var(--tl-font-display);
  font-size: 22px;
  line-height: 1.25;
}
.match__actions {
  margin-top: auto;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
}
.match__ghost {
  color: var(--tl-color-text-inverse);
  border-color: var(--tl-color-text-muted);
}
.match__ghost:hover {
  color: var(--tl-color-text-inverse);
}
.match__toast {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-inverse-muted);
}
</style>
