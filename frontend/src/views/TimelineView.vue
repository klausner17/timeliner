<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  TlAvatar,
  TlBadge,
  TlButton,
  TlCard,
  TlChip,
  TlCompareTracks,
  TlEventItem,
  TlOverviewBar,
  TlSegmented,
  axisTicks,
  clampWindow,
  datePos,
  innermostPeriod,
  parentPeriod,
  periodColorOf,
  periodFrom,
  periodRange,
  periodTo,
  type CompareEvent,
  type CompareLane,
  type PeriodColor,
} from '@/design-system'
import { errorMessage } from '@/data/api'
import {
  canEdit,
  followTimeline,
  getTimeline,
  handleLabel,
  listTimelines,
  loadEvents,
  loadSummary,
  rangeLabel,
  relatedTo,
  session,
  yearRange,
} from '@/data/store'
import type { Era, TimelineDetail, TimelineEvent, TimelinePage } from '@/data/types'
import type { EventWindow } from '@/data/windowing'
import { copyLink, dateLabel, detailedDateLabel, timeAgo, yearLabel } from '@/lib/format'
import { useLatest } from '@/lib/useLatest'
import NotFoundView from './NotFoundView.vue'
import VideoExport from './timeline/VideoExport.vue'

const route = useRoute()
const router = useRouter()

// A API já esconde o que o usuário não pode ver (privada de outra pessoa = 404).
const timeline = ref<TimelineDetail>()
const status = ref<'loading' | 'ok' | 'missing' | 'error'>('loading')
const statusError = ref('')
let timelineSeq = 0
watch(
  () => route.params.slug,
  async (slug) => {
    const id = ++timelineSeq
    status.value = 'loading'
    try {
      const t = await getTimeline(String(slug))
      if (id !== timelineSeq) return
      timeline.value = t
      status.value = t ? 'ok' : 'missing'
    } catch (e) {
      if (id !== timelineSeq) return
      statusError.value = errorMessage(e, 'Não deu para carregar a timeline.')
      status.value = 'error'
    }
  },
  { immediate: true },
)

const ALL = 'Tudo'
const PAGE = 100
const era = ref(ALL)
const mode = ref('vertical')
const selected = ref<string>()
const shareMsg = ref('')
watch(
  () => route.params.slug,
  () => {
    era.value = ALL
    selected.value = undefined
  },
)

const range = computed(() => (timeline.value ? yearRange(timeline.value) : undefined))
const THIS_YEAR = new Date().getFullYear()

/** Os períodos com a cor que cada um mostra (a escolhida ou a da posição na lista). */
const periods = computed(() => (timeline.value?.eras ?? []).map((e, i) => ({ ...e, color: periodColorOf(e, i) as PeriodColor })))
const showPeriods = ref(true)
const eraRange = computed<[number, number] | undefined>(() => {
  const e = timeline.value?.eras.find((x) => x.name === era.value)
  return e && range.value ? [e.start, e.end ?? range.value[1]] : undefined
})

// ---------- Vertical: páginas por cursor, a próxima carrega ao chegar no fim ----------

// Carrega a próxima página quando o fim da lista aparece na tela.
const sentinel = ref<HTMLElement>()
let io: IntersectionObserver | undefined
watch(sentinel, (el) => {
  io?.disconnect()
  if (!el || typeof IntersectionObserver === 'undefined') return
  io = new IntersectionObserver((entries) => entries.some((e) => e.isIntersecting) && loadPage(false), { rootMargin: '600px' })
  io.observe(el)
})
onBeforeUnmount(() => io?.disconnect())

const loaded = shallowRef<TimelineEvent[]>([])
const nextCursor = ref<string | null>(null)
const loading = ref(false)
const loadError = ref('')
let listSeq = 0

async function loadPage(reset: boolean) {
  const t = timeline.value
  if (!t || (!reset && (!nextCursor.value || loading.value))) return
  const id = reset ? ++listSeq : listSeq
  loading.value = true
  try {
    const [from, to] = eraRange.value ?? []
    const page = await loadEvents(t.slug, { from, to, cursor: reset ? null : nextCursor.value, limit: PAGE })
    if (id !== listSeq) return
    loaded.value = reset ? page.items : [...loaded.value, ...page.items]
    nextCursor.value = page.nextCursor
    loadError.value = ''
    // Se o fim da lista continua visível (tela alta), observar de novo dispara mais uma carga.
    if (sentinel.value && io) {
      io.unobserve(sentinel.value)
      io.observe(sentinel.value)
    }
  } catch (e) {
    if (id === listSeq) loadError.value = e instanceof Error ? e.message : 'Não deu para carregar os eventos.'
  } finally {
    if (id === listSeq) loading.value = false
  }
}
watch([() => timeline.value?.slug, eraRange, () => timeline.value?.eventCount], () => loadPage(true), { immediate: true })


type Group = { name: string; range?: string; color?: PeriodColor; parent?: string; events: TimelineEvent[] }

/**
 * Agrupa os eventos por período, na ordem do tempo. O evento marcado com um período vai
 * para ele; o sem marca vai para o período mais curto que contém o ano (1940 cai no
 * Estado Novo, não na Era Vargas). Eventos seguidos do mesmo período formam uma seção;
 * os que não cabem em nenhum ficam em "Fora dos períodos", no lugar deles no tempo.
 */
const groups = computed<Group[]>(() => {
  const events = loaded.value
  const list = periods.value
  if (!list.length) return [{ name: '', events }]
  const openEnd = Math.max(THIS_YEAR, range.value?.[1] ?? THIS_YEAR)
  const byName = new Map(list.map((p) => [p.name, p]))
  const out: Group[] = []
  for (const e of events) {
    const p = (e.era ? byName.get(e.era) : undefined) ?? innermostPeriod(list, datePos(e.year, e.month, e.day), openEnd)
    const name = p?.name ?? 'Fora dos períodos'
    const last = out[out.length - 1]
    if (last?.name === name) {
      last.events.push(e)
      continue
    }
    out.push(
      p
        ? { name, range: periodRange(p), color: p.color, parent: parentPeriod<Era>(list, p, openEnd)?.name, events: [e] }
        : { name, events: [e] },
    )
  }
  return out
})

// ---------- Minimapa ----------

const overviewTicks = computed(() => (range.value ? axisTicks(range.value[0], range.value[1]) : []))
// Histograma do eixo completo: 100 faixas, sem trazer os eventos.
const { data: overview } = useLatest(
  () => (timeline.value && range.value ? loadSummary(timeline.value.slug, { from: range.value[0], to: range.value[1], buckets: 100, maxEvents: 1 }) : undefined),
  undefined as EventWindow | undefined,
)
const overviewBuckets = computed(() => {
  const w = overview.value
  return w ? (w.mode === 'faixas' ? w.buckets : w.events.map((e) => ({ from: e.year, to: e.year, count: 1 }))) : []
})

// ---------- Horizontal: janela com zoom; muitos eventos viram faixas ----------

/** Limites do zoom: o eixo completo com uma folga de 3%. */
const bounds = computed<[number, number]>(() => {
  const r = range.value
  if (!r) return [0, 1]
  const pad = Math.max(1, Math.round((r[1] - r[0]) * 0.03))
  return [r[0] - pad, r[1] + pad]
})
const view = ref<[number, number]>([0, 1])
// Ao escolher um período, a trilha mostra a data exata dele, com uma folga dos lados.
const eraWindow = computed<[number, number] | undefined>(() => {
  const p = periods.value.find((x) => x.name === era.value)
  if (!p || !range.value) return undefined
  const from = periodFrom(p)
  const to = periodTo(p, Math.max(THIS_YEAR, range.value[1]))
  const pad = Math.max(1 / 12, (to - from) * 0.05)
  return clampWindow(from - pad, to + pad, { min: bounds.value[0], max: bounds.value[1] })
})
watch([bounds, eraWindow], () => (view.value = eraWindow.value ?? bounds.value), { immediate: true })
const viewStart = computed({ get: () => view.value[0], set: (v) => (view.value = [v, view.value[1]]) })
const viewEnd = computed({ get: () => view.value[1], set: (v) => (view.value = [view.value[0], v]) })
const overviewWindow = computed<[number, number] | undefined>({
  get: () => (mode.value === 'horizontal' ? view.value : eraRange.value),
  set: (v) => v && (view.value = v),
})

const { data: windowData } = useLatest(
  () => (timeline.value && mode.value === 'horizontal' ? loadSummary(timeline.value.slug, { from: Math.floor(view.value[0]), to: Math.ceil(view.value[1]), buckets: 60 }) : undefined),
  undefined as EventWindow | undefined,
)
const toCompare = (e: TimelineEvent): CompareEvent => ({ id: e.id, year: e.year, month: e.month, day: e.day, title: e.title })
const lanes = computed<CompareLane[]>(() => {
  const t = timeline.value
  const w = windowData.value
  if (!t || !w) return []
  return [
    {
      id: t.slug,
      name: t.title,
      color: t.color,
      extent: range.value,
      periods: showPeriods.value ? periods.value : undefined,
      events: w.events.map(toCompare),
      buckets: w.buckets.map((b) => ({ ...b, sample: b.sample.map(toCompare) })),
    },
  ]
})
const windowEvents = computed(() => {
  const w = windowData.value
  return w ? [...w.events, ...w.buckets.flatMap((b) => b.sample)] : []
})
const selectedEvent = computed(() => windowEvents.value.find((e) => e.id === selected.value) ?? windowEvents.value[0])
const grouped = computed(() => (windowData.value?.mode === 'faixas' ? windowData.value : undefined))

const pending = computed(() => timeline.value?.pendingSuggestions ?? 0)
const owner = computed(() => timeline.value?.owner)
const mine = computed(() => !!timeline.value && canEdit(timeline.value))
const following = computed(() => !!timeline.value?.following)

// "Compare com": entre as públicas mais populares, as que têm mais anos em comum.
const { data: popular } = useLatest(() => (timeline.value ? listTimelines({ pageSize: 50 }) : undefined), undefined as TimelinePage | undefined)
const related = computed(() => (timeline.value && popular.value ? relatedTo(timeline.value, popular.value.items) : []))

const followBusy = ref(false)
async function toggleFollow() {
  const t = timeline.value
  if (!t || followBusy.value) return
  if (!session.user) {
    router.push({ name: 'login', query: { next: route.fullPath } })
    return
  }
  followBusy.value = true
  try {
    await followTimeline(t.slug, !t.following)
    t.followers += t.following ? -1 : 1
    t.following = !t.following
  } catch (e) {
    shareMsg.value = errorMessage(e)
    setTimeout(() => (shareMsg.value = ''), 2500)
  } finally {
    followBusy.value = false
  }
}

const relatedColors = ['b', 'c', 'd'] as const

// ---------- Tela cheia: usa a API do navegador e, sem ela, ocupa a janela ----------

const root = ref<HTMLElement>()
const full = ref(false)

function enterFull() {
  full.value = true
  root.value?.requestFullscreen?.()?.catch(() => undefined)
}

function exitFull() {
  full.value = false
  if (document.fullscreenElement) document.exitFullscreen()
}

function onFullscreenChange() {
  if (!document.fullscreenElement) full.value = false
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && full.value) exitFull()
}

onMounted(() => {
  document.addEventListener('fullscreenchange', onFullscreenChange)
  document.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('fullscreenchange', onFullscreenChange)
  document.removeEventListener('keydown', onKeydown)
  if (full.value) exitFull()
})

async function share() {
  const ok = await copyLink(window.location.href)
  shareMsg.value = ok ? 'Link copiado' : 'Não deu para copiar o link'
  setTimeout(() => (shareMsg.value = ''), 2500)
}
</script>

<template>
  <main v-if="status === 'loading'" class="tl-container" aria-busy="true"></main>
  <main v-else-if="status === 'error'" class="tl-container body">
    <p class="body__note" role="alert">{{ statusError }}</p>
  </main>
  <NotFoundView v-else-if="!timeline" what="timeline" />
  <div v-else ref="root" class="view" :class="{ 'view--full': full }">
    <section class="top">
      <div class="tl-container top__inner">
        <nav aria-label="Você está em" class="crumbs">
          <RouterLink to="/">Explorar</RouterLink>
          <span aria-hidden="true">/</span>
          <RouterLink :to="{ path: '/', query: { cat: timeline.category } }">{{ timeline.category }}</RouterLink>
        </nav>

        <div class="top__head">
          <div class="top__text">
            <h1 class="top__title">{{ timeline.title }}</h1>
            <p v-if="timeline.description" class="top__desc">{{ timeline.description }}</p>
            <div class="top__meta">
              <span class="top__owner">
                <TlAvatar v-if="owner" :name="owner.handle" :tint="owner.tint" size="xs" decorative />
                {{ handleLabel(timeline.owner.handle) }}
              </span>
              <span>{{ rangeLabel(timeline) }}</span>
              <span>{{ timeline.eventCount.toLocaleString('pt-BR') }} eventos</span>
              <span>{{ timeline.contributorCount }} {{ timeline.contributorCount === 1 ? 'contribuidor' : 'contribuidores' }}</span>
              <span>Atualizada {{ timeAgo(timeline.updatedAt) }}</span>
              <TlBadge v-if="timeline.draft" tone="warning">Rascunho</TlBadge>
              <TlBadge v-else-if="timeline.visibility === 'privada'" tone="neutral">Privada</TlBadge>
              <TlBadge v-else-if="timeline.contributions === 'sugestoes'" tone="success">Aberta a contribuições</TlBadge>
            </div>
          </div>
          <div class="top__actions">
            <TlButton v-if="mine" :to="`/t/${timeline.slug}/editar`" variant="dark">Editar</TlButton>
            <TlButton v-else-if="timeline.contributions === 'sugestoes'" :to="`/t/${timeline.slug}/contribuir`" variant="primary" icon="plus">
              Sugerir evento
            </TlButton>
            <TlButton :to="{ path: '/comparar', query: { a: timeline.slug } }" icon="compare">Comparar com…</TlButton>
            <TlButton v-if="!mine" :aria-pressed="following" :disabled="followBusy" @click="toggleFollow">
              {{ following ? 'Seguindo' : 'Seguir' }}
            </TlButton>
            <VideoExport :timeline="timeline" />
            <TlButton icon="share" icon-only label="Compartilhar" @click="share" />
            <span class="top__toast" role="status">{{ shareMsg }}</span>
          </div>
        </div>

        <TlCard v-if="range" tone="muted" padding="sm">
          <TlOverviewBar
            v-model:window="overviewWindow"
            :start="range[0]"
            :end="range[1]"
            :buckets="overviewBuckets"
            :ticks="overviewTicks"
            :color="timeline.color"
            :periods="showPeriods ? periods : undefined"
            :interactive="mode === 'horizontal'"
          />
        </TlCard>

        <div class="top__controls">
          <div v-if="periods.length" role="group" aria-label="Filtrar por período" class="top__chips">
            <TlChip :pressed="era === ALL" @click="era = ALL">{{ ALL }}</TlChip>
            <TlChip v-for="p in periods" :key="p.name" :pressed="era === p.name" :title="periodRange(p)" @click="era = p.name">
              <span class="top__swatch" :style="{ background: `var(--tl-period-${p.color})` }" aria-hidden="true"></span>{{ p.name }}
            </TlChip>
            <TlButton size="sm" variant="ghost" :aria-pressed="showPeriods" @click="showPeriods = !showPeriods">
              {{ showPeriods ? 'Esconder faixas' : 'Mostrar faixas' }}
            </TlButton>
          </div>
          <span v-else></span>
          <div class="top__view">
            <TlSegmented
              v-model="mode"
              size="sm"
              label="Modo de visualização"
              :options="[
                { value: 'vertical', label: 'Vertical' },
                { value: 'horizontal', label: 'Horizontal' },
              ]"
            />
            <TlButton v-if="full" size="sm" icon="close" @click="exitFull">Sair da tela cheia</TlButton>
            <TlButton v-else size="sm" icon="expand" @click="enterFull">Tela cheia</TlButton>
          </div>
        </div>
      </div>
    </section>

    <div class="tl-container body" :class="{ 'body--wide': mode === 'horizontal' }">
      <main class="body__main">
        <TlCard v-if="!timeline.eventCount" tone="dashed" class="empty">
          <p>Esta timeline ainda não tem eventos.</p>
          <TlButton v-if="mine" :to="`/t/${timeline.slug}/editar`" variant="dark" icon="plus">Adicionar eventos</TlButton>
        </TlCard>

        <template v-else-if="mode === 'vertical'">
          <section v-for="(g, gi) in groups" :key="`${gi}-${g.name}`" class="era">
            <div v-if="g.name" class="era__head" :style="g.color ? { '--period': `var(--tl-period-${g.color})` } : undefined">
              <span v-if="g.color" class="era__swatch" aria-hidden="true"></span>
              <h2 class="era__name">{{ g.name }}</h2>
              <span v-if="g.range" class="era__range">{{ g.range }}</span>
              <span v-if="g.parent" class="era__parent">parte de {{ g.parent }}</span>
            </div>
            <TlEventItem
              v-for="e in g.events"
              :key="e.id"
              :year="yearLabel(e)"
              :date="detailedDateLabel(e)"
              :title="e.title"
              :description="e.description"
              :source-url="e.sourceUrl"
              :by="e.by"
              :color="timeline.color"
              class="body__event"
            />
          </section>
          <p v-if="!loaded.length && !loading" class="body__note">Nenhum evento neste período.</p>
          <p v-if="loadError" class="body__note" role="alert">{{ loadError }}</p>
          <div v-if="nextCursor" ref="sentinel" class="more">
            <TlButton :disabled="loading" @click="loadPage(false)">{{ loading ? 'Carregando…' : 'Carregar mais eventos' }}</TlButton>
          </div>
        </template>

        <template v-else>
          <TlCompareTracks v-model:selected="selected" v-model:start="viewStart" v-model:end="viewEnd" :lanes="lanes" :min="bounds[0]" :max="bounds[1]" zoomable />
          <p v-if="grouped" class="body__note">
            {{ grouped.total.toLocaleString('pt-BR') }} eventos neste trecho, agrupados por período. Clique num grupo ou aproxime para ver cada evento.
          </p>
          <TlCard v-if="selectedEvent" padding="lg" class="detail">
            <span class="detail__date">{{ dateLabel(selectedEvent) }}</span>
            <h2 class="detail__title">{{ selectedEvent.title }}</h2>
            <p>{{ selectedEvent.description }}</p>
            <span class="detail__by">
              <a v-if="selectedEvent.sourceUrl" :href="selectedEvent.sourceUrl" target="_blank" rel="noopener">Fonte</a>
              adicionado por {{ selectedEvent.by }}
            </span>
          </TlCard>
        </template>
      </main>

      <aside class="body__side">
        <TlCard class="side">
          <h2 class="side__title">Contribuidores</h2>
          <div class="side__people">
            <TlAvatar :name="timeline.owner.handle" :tint="timeline.owner.tint" :title="handleLabel(timeline.owner.handle)" />
            <TlAvatar v-if="timeline.contributorCount > 1" :name="`+${timeline.contributorCount - 1}`" :initial="`+${timeline.contributorCount - 1}`" />
          </div>
          <p class="side__text">
            <template v-if="timeline.contributions === 'sugestoes'">Contribuições passam por aprovação do dono antes de aparecer.</template>
            <template v-else-if="timeline.contributions === 'convidados'">Só pessoas convidadas editam esta timeline.</template>
            <template v-else>Só o dono edita esta timeline.</template>
          </p>
          <RouterLink v-if="pending && mine" :to="`/t/${timeline.slug}/contribuir`" class="side__link">
            Ver {{ pending }} {{ pending === 1 ? 'sugestão pendente' : 'sugestões pendentes' }}
          </RouterLink>
        </TlCard>

        <TlCard v-if="related.length" class="side">
          <h2 class="side__title">Compare com</h2>
          <RouterLink v-for="(r, i) in related" :key="r.slug" :to="{ path: '/comparar', query: { a: timeline.slug, b: r.slug } }" class="side__pair">
            <span class="side__swatch" :style="{ background: `var(--tl-track-${relatedColors[i]})` }" aria-hidden="true"></span>
            {{ r.title }}
          </RouterLink>
        </TlCard>

        <TlCard v-if="timeline.tags.length" class="side">
          <h2 class="side__title">Tags</h2>
          <div class="side__tags">
            <TlChip v-for="tag in timeline.tags" :key="tag" kind="tag" size="sm" :to="{ path: '/', query: { q: tag } }">{{ tag }}</TlChip>
          </div>
        </TlCard>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.top {
  background: var(--tl-color-surface);
  border-bottom: 1px solid var(--tl-color-border);
}
.top__inner {
  padding-top: var(--tl-space-7);
  padding-bottom: var(--tl-space-6);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-5);
}
.crumbs {
  display: flex;
  gap: var(--tl-space-2);
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.crumbs a {
  color: var(--tl-color-text-muted);
}
.top__head {
  display: flex;
  flex-wrap: wrap;
  align-items: start;
  justify-content: space-between;
  gap: var(--tl-space-5);
}
.top__text {
  flex: 1 1 520px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.top__title {
  font-size: var(--tl-text-5xl);
  line-height: var(--tl-leading-tight);
  letter-spacing: var(--tl-tracking-tight);
}
.top__desc {
  max-width: 680px;
  font-size: var(--tl-text-lg);
  color: var(--tl-color-text-muted);
}
.top__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2) 18px;
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.top__owner {
  display: inline-flex;
  align-items: center;
  gap: var(--tl-space-2);
}
.top__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
  position: relative;
}
.top__toast {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.top__toast:empty {
  display: none;
}
.top__controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-3);
}
.top__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.top__view {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
}

.view--full {
  position: fixed;
  inset: 0;
  z-index: 100;
  overflow: auto;
  background: var(--tl-color-bg);
}
.view--full .tl-container {
  max-width: none;
}
.view--full .top__inner {
  padding-top: var(--tl-space-4);
  padding-bottom: var(--tl-space-4);
  gap: var(--tl-space-3);
}
.view--full .top__title {
  font-size: var(--tl-text-2xl);
}
.view--full .crumbs,
.view--full .top__desc,
.view--full .top__meta,
.view--full .top__actions,
.view--full .body__side {
  display: none;
}
.view--full .body {
  padding-top: var(--tl-space-5);
}

.body {
  padding-top: var(--tl-space-8);
  padding-bottom: calc(var(--tl-space-8) * 2);
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-8);
  align-items: start;
}
.body__main {
  flex: 999 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.body__side {
  flex: 1 1 300px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
/* No modo horizontal a linha do tempo fica sozinha na largura toda; o resto vai para baixo. */
.body--wide .body__main {
  flex-basis: 100%;
}
.body--wide .body__side {
  flex-basis: 100%;
  flex-direction: row;
  flex-wrap: wrap;
  align-items: start;
}
.body--wide .side {
  flex: 1 1 280px;
}
.era {
  display: flex;
  flex-direction: column;
}
.era__head {
  flex-wrap: wrap;
  display: flex;
  align-items: baseline;
  gap: var(--tl-space-3);
  padding: var(--tl-space-4) 0 var(--tl-space-3);
}
.era__name {
  font-size: 22px;
}
.era__range,
.era__parent {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.era__parent {
  font-style: italic;
}
.era__swatch {
  flex: none;
  align-self: center;
  width: 4px;
  height: 22px;
  border-radius: var(--tl-radius-pill);
  background: var(--period);
}
.top__swatch {
  display: inline-block;
  width: 8px;
  height: 8px;
  margin-right: 6px;
  border-radius: var(--tl-radius-pill);
  vertical-align: middle;
}
.body__event {
  /* O navegador só desenha os eventos que estão perto da tela. */
  content-visibility: auto;
  contain-intrinsic-size: auto 120px;
}
.body__note {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.more {
  display: flex;
  justify-content: center;
  padding: var(--tl-space-4) 0;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: var(--tl-space-4);
}
.detail {
  margin-top: var(--tl-space-4);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail__date,
.detail__by {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.detail__by {
  display: flex;
  gap: var(--tl-space-4);
}
.detail__title {
  font-size: var(--tl-text-2xl);
}
.side {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
}
.side__title {
  font-family: var(--tl-font-body);
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  letter-spacing: 0;
}
.side__people,
.side__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.side__text {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.side__link {
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-md);
  padding: 6px 0;
}
.side__pair {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: var(--tl-control-md);
  padding: 10px 12px;
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface-muted);
  border: 1px solid var(--tl-color-border-subtle);
  text-decoration: none;
  color: var(--tl-color-text);
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-medium);
}
.side__pair:hover {
  border-color: var(--tl-color-border-strong);
  color: var(--tl-color-text);
}
.side__swatch {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: var(--tl-radius-pill);
}

@media (max-width: 640px) {
  .top__title {
    font-size: var(--tl-text-4xl);
  }
}
</style>
