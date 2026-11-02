<script setup lang="ts">
/*
 * Modo apresentação: a timeline evento a evento, em tela cheia, para aulas e palestras.
 * Setas, espaço e PageUp/PageDown (passadores de slide) navegam; no celular, deslize.
 * O evento atual fica na URL (?evento=), então recarregar ou mandar o link volta nele.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TlButton, periodColorOf, periodFrom, periodTo, provideYearFormat, type PeriodColor } from '@/design-system'
import { errorMessage } from '@/data/api'
import { getTimeline, handleLabel, loadEvents, rangeLabel, yearRange } from '@/data/store'
import type { TimelineDetail, TimelineEvent } from '@/data/types'
import { dateLabel } from '@/lib/format'
import { yearFormatOf } from '@/lib/calendar'
import { reckoningOf } from '@/lib/scale'
import { axisFraction, buildSlides, groupByPeriod, eventText, type Slide } from '@/lib/presentation'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const router = useRouter()

const slug = computed(() => String(route.params.slug))
const timeline = ref<TimelineDetail>()
const status = ref<'loading' | 'ok' | 'missing' | 'error'>('loading')
const statusError = ref('')

const reck = computed(() => reckoningOf(timeline.value))
const years = computed(() => yearFormatOf(reck.value))
provideYearFormat(years)

const range = computed(() => (timeline.value ? yearRange(timeline.value) : undefined))
const axisEnd = computed(() => range.value?.[1] ?? new Date().getFullYear())
const periods = computed(() => (timeline.value?.eras ?? []).map((e, i) => ({ ...e, color: periodColorOf(e, i) as PeriodColor })))

/** Período escolhido na timeline (?periodo=): a apresentação fica só nele. */
const onlyPeriod = computed(() => periods.value.find((p) => p.name === route.query.periodo))
const filter = computed(() => {
  const p = onlyPeriod.value
  return p ? { from: p.start, to: p.end ?? axisEnd.value } : {}
})

// ---------- Eventos, página a página ----------

const PAGE = 200
const events = shallowRef<TimelineEvent[]>([])
const nextCursor = ref<string | null>(null)
const loading = ref(false)
const loadError = ref('')

async function loadMore(): Promise<void> {
  const t = timeline.value
  if (!t || loading.value || (events.value.length && !nextCursor.value)) return
  loading.value = true
  try {
    const page = await loadEvents(t.slug, { ...filter.value, cursor: nextCursor.value, limit: PAGE })
    events.value = [...events.value, ...page.items]
    nextCursor.value = page.nextCursor
    loadError.value = ''
  } catch (e) {
    loadError.value = errorMessage(e, 'Não deu para carregar os eventos.')
  } finally {
    loading.value = false
  }
}

const groups = computed(() => groupByPeriod(events.value, periods.value, years.value, axisEnd.value))
const complete = computed(() => !!timeline.value && !loading.value && !nextCursor.value && !loadError.value)
const slides = computed<Slide[]>(() => buildSlides(groups.value, complete.value))
const index = ref(0)
const slide = computed(() => slides.value[Math.min(index.value, slides.value.length - 1)])
const eventTotal = computed(() => (onlyPeriod.value ? undefined : timeline.value?.eventCount))

/** Só depois de achar o evento do link a URL passa a seguir o slide atual. */
const ready = ref(false)
const startAt = route.query.evento

onMounted(async () => {
  try {
    const t = await getTimeline(slug.value)
    timeline.value = t
    status.value = t ? 'ok' : 'missing'
    if (!t) return
    await loadMore()
    // Volta no evento do link, carregando as páginas até ele.
    const wanted = startAt
    if (typeof wanted === 'string') {
      let at = -1
      while ((at = slides.value.findIndex((s) => s.kind === 'event' && s.event.id === wanted)) < 0 && nextCursor.value && !loadError.value) await loadMore()
      if (at >= 0) index.value = at
    }
    ready.value = true
  } catch (e) {
    statusError.value = errorMessage(e, 'Não deu para carregar a timeline.')
    status.value = 'error'
  }
})

// Carrega a próxima página antes de chegar no fim do que já veio.
watch([index, () => slides.value.length], ([i, n]) => {
  if (n - i < 8 && nextCursor.value) loadMore()
})

// O evento atual vai para a URL, sem empilhar no histórico.
watch(slide, (s) => {
  if (!ready.value || !s) return
  const evento = s.kind === 'event' ? s.event.id : undefined
  if (route.query.evento === evento) return
  router.replace({ query: { ...route.query, evento } })
})

// ---------- Navegação ----------

const direction = ref<'next' | 'prev'>('next')
const canPrev = computed(() => index.value > 0)
const canNext = computed(() => index.value < slides.value.length - 1)

function go(to: number) {
  const last = slides.value.length - 1
  const i = Math.max(0, Math.min(last, to))
  if (i === index.value) return
  direction.value = i > index.value ? 'next' : 'prev'
  index.value = i
}
const next = () => go(index.value + 1)
const prev = () => go(index.value - 1)

const timelineLink = computed(() => ({ name: 'timeline', params: { slug: slug.value } }))
function leave() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => undefined)
  router.push(timelineLink.value)
}

function onKeydown(e: KeyboardEvent) {
  if (e.defaultPrevented || e.altKey || e.ctrlKey || e.metaKey) return
  const el = e.target as HTMLElement | null
  if (el?.closest('input, textarea, select, [contenteditable="true"]')) return
  // Espaço e Enter num botão ou link fazem o clique dele.
  const onControl = !!el?.closest('button, a')
  switch (e.key) {
    case 'ArrowRight':
    case 'ArrowDown':
    case 'PageDown':
      next()
      break
    case ' ':
    case 'Enter':
      if (onControl) return
      if (e.shiftKey) prev()
      else next()
      break
    case 'ArrowLeft':
    case 'ArrowUp':
    case 'PageUp':
    case 'Backspace':
      prev()
      break
    case 'Home':
      go(0)
      break
    case 'End':
      go(slides.value.length - 1)
      break
    case 'f':
    case 'F':
      toggleFullscreen()
      break
    case 'Escape':
      // Em tela cheia, o Esc já é do navegador (sai da tela cheia).
      if (!document.fullscreenElement) leave()
      break
    default:
      return
  }
  e.preventDefault()
}

// Deslizar no celular: para a esquerda avança, para a direita volta.
let touch: { x: number; y: number; t: number } | undefined
function onTouchStart(e: TouchEvent) {
  const p = e.touches[0]
  touch = e.touches.length === 1 && p ? { x: p.clientX, y: p.clientY, t: Date.now() } : undefined
}
function onTouchEnd(e: TouchEvent) {
  const p = e.changedTouches[0]
  if (!touch || !p) return
  const dx = p.clientX - touch.x
  const dy = p.clientY - touch.y
  const quick = Date.now() - touch.t < 800
  touch = undefined
  if (quick && Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy) * 1.5) {
    if (dx < 0) next()
    else prev()
  }
}

// ---------- Tela cheia ----------

const stage = ref<HTMLElement>()
const isFull = ref(false)
const canFullscreen = typeof document !== 'undefined' && !!document.documentElement.requestFullscreen
function toggleFullscreen() {
  if (document.fullscreenElement) document.exitFullscreen().catch(() => undefined)
  else document.documentElement.requestFullscreen?.()?.catch(() => undefined)
}
const onFullscreenChange = () => (isFull.value = !!document.fullscreenElement)

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
  document.addEventListener('fullscreenchange', onFullscreenChange)
  // Quem clicou em "Apresentar" ainda está no gesto do clique: dá para abrir em tela cheia.
  document.documentElement.requestFullscreen?.()?.catch(() => undefined)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeydown)
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  if (document.fullscreenElement) document.exitFullscreen().catch(() => undefined)
})

// O texto longo de um evento começa do topo a cada slide.
watch(index, () => nextTick(() => stage.value?.querySelector('.slide__scroll')?.scrollTo?.({ top: 0 })))

// ---------- O que aparece no slide ----------

const current = computed(() => (slide.value?.kind === 'event' ? slide.value : undefined))
const text = computed(() => (current.value ? eventText(current.value.event) : undefined))
const currentGroup = computed(() => (slide.value?.kind === 'event' || slide.value?.kind === 'period' ? slide.value.group : undefined))
const accent = computed(() => {
  const c = currentGroup.value?.color
  return c ? `var(--tl-period-${c})` : `var(--tl-track-${timeline.value?.color ?? 'a'})`
})

/** O eixo de baixo: as faixas dos períodos e onde o evento atual está. */
const axis = computed(() => {
  const r = range.value
  if (!r) return undefined
  const [start, end] = r[1] > r[0] ? r : [r[0] - 1, r[0] + 1]
  const openEnd = years.value.openEnd(end)
  const pct = (v: number) => `${Math.min(100, Math.max(0, ((v - start) / (end - start)) * 100))}%`
  const bands = periods.value.map((p) => {
    const from = periodFrom(p)
    const to = Math.min(periodTo(p, openEnd), end)
    return { name: p.name, color: p.color, left: pct(from), width: `calc(${pct(to)} - ${pct(from)})` }
  })
  const s = slide.value
  let marker: number | undefined
  if (s?.kind === 'event') marker = axisFraction(s.event, start, end)
  else if (s?.kind === 'period') {
    const p = periods.value.find((x) => x.name === s.group.name)
    if (p) marker = (periodFrom(p) - start) / (end - start)
  } else if (s?.kind === 'end') marker = 1
  else marker = 0
  return { bands, marker: marker === undefined ? undefined : `${Math.min(1, Math.max(0, marker)) * 100}%`, startLabel: years.value.year(Math.round(start)), endLabel: years.value.year(Math.round(end)) }
})

const counter = computed(() => {
  const s = slide.value
  if (s?.kind !== 'event') return ''
  const total = eventTotal.value ?? (complete.value ? s.number + slides.value.slice(index.value + 1).filter((x) => x.kind === 'event').length : undefined)
  return total ? `${s.number} de ${total.toLocaleString('pt-BR')}` : `${s.number}`
})
const periodEventCount = computed(() => (slide.value?.kind === 'period' ? slide.value.group.events.length : 0))
</script>

<template>
  <main v-if="status === 'loading'" class="present present--loading" aria-busy="true"></main>
  <main v-else-if="status === 'error'" class="present present--loading">
    <p role="alert">{{ statusError }}</p>
    <TlButton :to="timelineLink">Voltar para a timeline</TlButton>
  </main>
  <NotFoundView v-else-if="!timeline" what="timeline" />
  <main
    v-else
    ref="stage"
    class="present"
    :style="{ '--accent': accent }"
    aria-roledescription="apresentação"
    :aria-label="`Apresentação: ${timeline.title}`"
    @touchstart.passive="onTouchStart"
    @touchend.passive="onTouchEnd"
  >
    <header class="bar">
      <RouterLink :to="timelineLink" class="bar__title" :title="`Voltar para ${timeline.title}`">{{ timeline.title }}</RouterLink>
      <span v-if="currentGroup?.color" class="bar__period">
        <span class="bar__swatch" aria-hidden="true"></span>{{ currentGroup.name }}
      </span>
      <span class="bar__spacer"></span>
      <span v-if="counter" class="bar__counter">{{ counter }}</span>
      <TlButton
        v-if="canFullscreen"
        size="sm"
        variant="ghost"
        icon="expand"
        icon-only
        :label="isFull ? 'Sair da tela cheia (F)' : 'Tela cheia (F)'"
        :title="isFull ? 'Sair da tela cheia (F)' : 'Tela cheia (F)'"
        @click="toggleFullscreen"
      />
      <TlButton size="sm" variant="ghost" icon="close" icon-only label="Sair da apresentação" title="Sair da apresentação (Esc)" @click="leave" />
    </header>

    <div class="stage" aria-live="polite">
      <Transition :name="`slide-${direction}`" mode="out-in">
        <section v-if="slide?.kind === 'intro'" :key="slide.key" class="slide slide--intro">
          <span class="slide__eyebrow">{{ onlyPeriod ? onlyPeriod.name : rangeLabel(timeline) }}</span>
          <h1 class="slide__title slide__title--xl">{{ timeline.title }}</h1>
          <p v-if="timeline.description" class="slide__lead">{{ timeline.description }}</p>
          <p class="slide__meta">
            {{ (eventTotal ?? events.length).toLocaleString('pt-BR') }} eventos · por {{ handleLabel(timeline.owner.handle) }}
          </p>
          <p v-if="!timeline.eventCount" class="slide__meta">Esta timeline ainda não tem eventos.</p>
          <TlButton v-else variant="dark" size="lg" icon="play" class="slide__start" @click="next">Começar</TlButton>
        </section>

        <section v-else-if="slide?.kind === 'period'" :key="slide.key" class="slide slide--period">
          <span class="slide__eyebrow">Período</span>
          <h2 class="slide__title slide__title--xl">{{ slide.group.name }}</h2>
          <p v-if="slide.group.range" class="slide__lead">{{ slide.group.range }}</p>
          <p class="slide__meta">
            <template v-if="slide.group.parent">Parte de {{ slide.group.parent }} · </template>
            {{ periodEventCount }}{{ nextCursor && slide.group === groups[groups.length - 1] ? '+' : '' }}
            {{ periodEventCount === 1 ? 'evento' : 'eventos' }}
          </p>
        </section>

        <article v-else-if="slide?.kind === 'event'" :key="slide.key" class="slide slide--event">
          <div class="slide__scroll">
            <span class="slide__date">{{ dateLabel(slide.event, reck) }}</span>
            <h2 class="slide__title">{{ slide.event.title }}</h2>
            <p v-if="text?.lead" class="slide__lead" :class="{ 'slide__lead--small': text.lead.length > 280 }">{{ text.lead }}</p>
            <p v-for="(para, i) in text?.body" :key="i" class="slide__long">{{ para }}</p>
            <p v-if="text?.links.length" class="slide__links">
              <a v-for="l in text.links" :key="l.url" :href="l.url" target="_blank" rel="noopener">{{ l.label }}</a>
            </p>
          </div>
        </article>

        <section v-else-if="slide?.kind === 'end'" :key="slide.key" class="slide slide--intro">
          <span class="slide__eyebrow">Fim</span>
          <h2 class="slide__title slide__title--xl">{{ timeline.title }}</h2>
          <p v-if="timeline.cyclic" class="slide__lead">↻ Depois daqui, o ciclo recomeça do início.</p>
          <div class="slide__actions">
            <TlButton variant="dark" icon="back" @click="go(0)">Recomeçar</TlButton>
            <TlButton :to="timelineLink">Voltar para a timeline</TlButton>
          </div>
        </section>
      </Transition>
      <p v-if="loadError" class="stage__error" role="alert">
        {{ loadError }} <button type="button" class="stage__retry" @click="loadMore">Tentar de novo</button>
      </p>
    </div>

    <footer class="foot">
      <TlButton icon="back" icon-only size="lg" label="Anterior" title="Anterior (←)" :disabled="!canPrev" @click="prev" />
      <div v-if="axis" class="axis" aria-hidden="true">
        <div class="axis__rail">
          <span v-for="b in axis.bands" :key="b.name" class="axis__band" :style="{ left: b.left, width: b.width, background: `var(--tl-period-${b.color})` }"></span>
          <span v-if="axis.marker" class="axis__marker" :style="{ left: axis.marker }"></span>
        </div>
        <div class="axis__labels">
          <span>{{ axis.startLabel }}</span>
          <span class="axis__hint">← → para navegar · F tela cheia</span>
          <span>{{ axis.endLabel }}</span>
        </div>
      </div>
      <TlButton
        :icon="loading && !canNext ? undefined : 'forward'"
        icon-only
        size="lg"
        variant="dark"
        :label="loading && !canNext ? 'Carregando' : 'Próximo'"
        title="Próximo (→)"
        :disabled="!canNext"
        @click="next"
      />
    </footer>
  </main>
</template>

<style scoped>
.present {
  --accent: var(--tl-track-a);
  position: fixed;
  inset: 0;
  z-index: 100;
  display: grid;
  grid-template-rows: auto 1fr auto;
  background: var(--tl-color-bg);
  color: var(--tl-color-text);
  overflow: hidden;
}
.present--loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--tl-space-4);
}

.bar {
  display: flex;
  align-items: center;
  gap: var(--tl-space-3);
  padding: var(--tl-space-3) var(--tl-space-5);
  min-width: 0;
}
.bar__title {
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text);
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.bar__period {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 10px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
  white-space: nowrap;
}
.bar__swatch {
  width: 8px;
  height: 8px;
  border-radius: var(--tl-radius-pill);
  background: var(--accent);
}
.bar__spacer {
  flex: 1;
}
.bar__counter {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.stage {
  position: relative;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--tl-space-4) clamp(16px, 6vw, 96px);
  touch-action: pan-y;
}
.stage__error {
  position: absolute;
  bottom: var(--tl-space-3);
  font-size: var(--tl-text-sm);
  color: var(--tl-color-warning-strong);
}
.stage__retry {
  border: 0;
  background: none;
  color: inherit;
  text-decoration: underline;
  cursor: pointer;
  font: inherit;
}

.slide {
  width: 100%;
  max-width: 1200px;
  max-height: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.slide--intro,
.slide--period {
  align-items: center;
  text-align: center;
}
.slide--period .slide__title {
  color: var(--accent);
}
.slide--event {
  flex-direction: row;
  align-items: center;
  gap: clamp(24px, 4vw, 64px);
}
.slide__scroll {
  flex: 1 1 0;
  min-width: 0;
  max-height: 100%;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
  padding-left: var(--tl-space-5);
  border-left: 6px solid var(--accent);
}
.slide__eyebrow,
.slide__date {
  font-size: clamp(14px, 1.6vw, 20px);
  font-weight: var(--tl-weight-semibold);
  letter-spacing: var(--tl-tracking-eyebrow);
  text-transform: uppercase;
  color: var(--accent);
}
.slide__date {
  text-transform: none;
  letter-spacing: 0;
  font-family: var(--tl-font-display);
  font-size: clamp(20px, 2.6vw, 34px);
}
.slide__title {
  font-family: var(--tl-font-display);
  font-size: clamp(30px, 4.6vw, 64px);
  line-height: var(--tl-leading-tight);
  letter-spacing: var(--tl-tracking-tight);
  text-wrap: balance;
}
.slide__title--xl {
  font-size: clamp(36px, 6.4vw, 88px);
}
.slide__lead {
  font-size: clamp(18px, 2vw, 26px);
  line-height: 1.45;
  color: var(--tl-color-text-strong);
  max-width: 60ch;
  text-wrap: pretty;
}
.slide__lead--small {
  font-size: clamp(17px, 1.6vw, 21px);
}
.slide__long {
  font-size: clamp(16px, 1.5vw, 20px);
  line-height: 1.6;
  color: var(--tl-color-text-body);
  max-width: 68ch;
  white-space: pre-line;
}
.slide__meta {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.slide__links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2) var(--tl-space-4);
  font-size: var(--tl-text-md);
}
.slide__start {
  margin-top: var(--tl-space-3);
}
.slide__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--tl-space-2);
  margin-top: var(--tl-space-3);
}

.foot {
  display: flex;
  align-items: center;
  gap: var(--tl-space-4);
  padding: var(--tl-space-3) var(--tl-space-5) var(--tl-space-5);
}
.axis {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.axis__rail {
  position: relative;
  height: 8px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-rail);
  overflow: visible;
}
.axis__band {
  position: absolute;
  top: 0;
  bottom: 0;
  opacity: 0.45;
  border-radius: var(--tl-radius-pill);
}
.axis__marker {
  position: absolute;
  top: 50%;
  width: 18px;
  height: 18px;
  margin: -9px 0 0 -9px;
  border-radius: var(--tl-radius-pill);
  background: var(--accent);
  border: 3px solid var(--tl-color-surface);
  box-shadow: 0 1px 4px rgb(0 0 0 / 0.25);
  transition: left 0.35s ease;
}
.axis__labels {
  display: flex;
  justify-content: space-between;
  gap: var(--tl-space-3);
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
  font-variant-numeric: tabular-nums;
}
.axis__hint {
  color: var(--tl-color-text-subtle);
}

.slide-next-enter-active,
.slide-next-leave-active,
.slide-prev-enter-active,
.slide-prev-leave-active {
  transition:
    opacity 0.18s ease,
    transform 0.18s ease;
}
.slide-next-enter-from,
.slide-prev-leave-to {
  opacity: 0;
  transform: translateX(32px);
}
.slide-next-leave-to,
.slide-prev-enter-from {
  opacity: 0;
  transform: translateX(-32px);
}
@media (prefers-reduced-motion: reduce) {
  .slide-next-enter-active,
  .slide-next-leave-active,
  .slide-prev-enter-active,
  .slide-prev-leave-active,
  .axis__marker {
    transition: none;
  }
}

@media (max-width: 760px) {
  .bar {
    padding: var(--tl-space-2) var(--tl-space-3);
  }
  .bar__period {
    display: none;
  }
  .stage {
    align-items: start;
    padding: var(--tl-space-3) var(--tl-space-4);
  }
  .slide--event {
    height: 100%;
    align-items: stretch;
  }
  .slide--intro,
  .slide--period {
    margin: auto 0;
  }
  .slide__scroll {
    flex: 1 1 auto;
    padding-left: var(--tl-space-3);
    border-left-width: 4px;
  }
      .foot {
    padding: var(--tl-space-2) var(--tl-space-3) var(--tl-space-4);
    gap: var(--tl-space-3);
  }
  .axis__hint {
    display: none;
  }
}
</style>
