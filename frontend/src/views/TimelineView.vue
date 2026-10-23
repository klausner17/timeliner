<script setup lang="ts">
import { computed, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { useRoute } from 'vue-router'
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
  type CompareEvent,
  type CompareLane,
} from '@/design-system'
import {
  canEdit,
  canView,
  contributors,
  findTimeline,
  findUser,
  isFollowingTimeline,
  loadEvents,
  loadSummary,
  rangeLabel,
  relatedTimelines,
  suggestionsFor,
  toggleFollowTimeline,
  yearRange,
} from '@/data/store'
import type { TimelineEvent } from '@/data/types'
import type { EventWindow } from '@/data/windowing'
import { copyLink, dateLabel, yearLabel } from '@/lib/format'
import { useLatest } from '@/lib/useLatest'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const timeline = computed(() => {
  const t = findTimeline(String(route.params.slug))
  return t && canView(t) ? t : undefined
})

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
watch([() => timeline.value?.slug, eraRange, () => timeline.value?.events.length], () => loadPage(true), { immediate: true })


type Group = { name: string; range?: string; events: TimelineEvent[] }

const groups = computed<Group[]>(() => {
  const t = timeline.value
  if (!t) return []
  const events = loaded.value
  if (!t.eras.length) return [{ name: '', events }]
  const out: Group[] = t.eras.map((e) => ({ name: e.name, range: `${e.start}–${e.end ?? 'hoje'}`, events: [] }))
  const loose: TimelineEvent[] = []
  for (const e of events) {
    const g = out.find((x) => x.name === e.era)
    if (g) g.events.push(e)
    else loose.push(e)
  }
  if (loose.length) out.push({ name: 'Outros eventos', events: loose })
  return out.filter((g) => g.events.length)
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
watch([bounds, eraRange], () => (view.value = eraRange.value ?? bounds.value), { immediate: true })
const viewStart = computed({ get: () => view.value[0], set: (v) => (view.value = [v, view.value[1]]) })
const viewEnd = computed({ get: () => view.value[1], set: (v) => (view.value = [view.value[0], v]) })
const overviewWindow = computed<[number, number] | undefined>({
  get: () => (mode.value === 'horizontal' ? view.value : eraRange.value),
  set: (v) => v && (view.value = v),
})

const { data: windowData } = useLatest(
  () => (timeline.value && mode.value === 'horizontal' ? loadSummary(timeline.value.slug, { from: view.value[0], to: view.value[1], buckets: 60 }) : undefined),
  undefined as EventWindow | undefined,
)
const toCompare = (e: TimelineEvent): CompareEvent => ({ id: e.id, year: e.year, title: e.title })
const lanes = computed<CompareLane[]>(() => {
  const t = timeline.value
  const w = windowData.value
  if (!t || !w) return []
  return [{ id: t.slug, name: t.title, color: t.color, events: w.events.map(toCompare), buckets: w.buckets.map((b) => ({ ...b, sample: b.sample.map(toCompare) })) }]
})
const windowEvents = computed(() => {
  const w = windowData.value
  return w ? [...w.events, ...w.buckets.flatMap((b) => b.sample)] : []
})
const selectedEvent = computed(() => windowEvents.value.find((e) => e.id === selected.value) ?? windowEvents.value[0])
const grouped = computed(() => (windowData.value?.mode === 'faixas' ? windowData.value : undefined))

const people = computed(() => (timeline.value ? contributors(timeline.value).map(findUser) : []))
const pending = computed(() => (timeline.value ? suggestionsFor(timeline.value.slug).filter((s) => s.status === 'pendente').length : 0))
const related = computed(() => (timeline.value ? relatedTimelines(timeline.value) : []))
const owner = computed(() => (timeline.value ? findUser(timeline.value.owner) : undefined))
const mine = computed(() => !!timeline.value && canEdit(timeline.value))
const following = computed(() => !!timeline.value && isFollowingTimeline(timeline.value.slug))

const relatedColors = ['b', 'c', 'd'] as const

async function share() {
  const ok = await copyLink(window.location.href)
  shareMsg.value = ok ? 'Link copiado' : 'Não deu para copiar o link'
  setTimeout(() => (shareMsg.value = ''), 2500)
}
</script>

<template>
  <NotFoundView v-if="!timeline" what="timeline" />
  <div v-else>
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
                {{ timeline.owner }}
              </span>
              <span>{{ rangeLabel(timeline) }}</span>
              <span>{{ timeline.events.length.toLocaleString('pt-BR') }} eventos</span>
              <span>{{ people.length }} contribuidores</span>
              <span>{{ timeline.updatedLabel }}</span>
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
            <TlButton v-if="!mine" :aria-pressed="following" @click="toggleFollowTimeline(timeline)">
              {{ following ? 'Seguindo' : 'Seguir' }}
            </TlButton>
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
            :interactive="mode === 'horizontal'"
          />
        </TlCard>

        <div class="top__controls">
          <div v-if="timeline.eras.length" role="group" aria-label="Filtrar por período" class="top__chips">
            <TlChip :pressed="era === ALL" @click="era = ALL">{{ ALL }}</TlChip>
            <TlChip v-for="e in timeline.eras" :key="e.name" :pressed="era === e.name" @click="era = e.name">{{ e.name }}</TlChip>
          </div>
          <span v-else></span>
          <TlSegmented
            v-model="mode"
            size="sm"
            label="Modo de visualização"
            :options="[
              { value: 'vertical', label: 'Vertical' },
              { value: 'horizontal', label: 'Horizontal' },
            ]"
          />
        </div>
      </div>
    </section>

    <div class="tl-container body">
      <main class="body__main">
        <TlCard v-if="!timeline.events.length" tone="dashed" class="empty">
          <p>Esta timeline ainda não tem eventos.</p>
          <TlButton v-if="mine" :to="`/t/${timeline.slug}/editar`" variant="dark" icon="plus">Adicionar eventos</TlButton>
        </TlCard>

        <template v-else-if="mode === 'vertical'">
          <section v-for="g in groups" :key="g.name" class="era">
            <div v-if="g.name" class="era__head">
              <h2 class="era__name">{{ g.name }}</h2>
              <span v-if="g.range" class="era__range">{{ g.range }}</span>
            </div>
            <TlEventItem
              v-for="e in g.events"
              :key="e.id"
              :year="yearLabel(e)"
              :date="e.date ?? (e.endYear ? dateLabel(e) : undefined)"
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
            <TlAvatar v-for="p in people.slice(0, 7)" :key="p.handle" :name="p.handle" :tint="p.tint" :title="p.handle" />
            <TlAvatar v-if="people.length > 7" :name="`+${people.length - 7}`" :initial="`+${people.length - 7}`" />
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
.era {
  display: flex;
  flex-direction: column;
}
.era__head {
  display: flex;
  align-items: baseline;
  gap: var(--tl-space-3);
  padding: var(--tl-space-4) 0 var(--tl-space-3);
}
.era__name {
  font-size: 22px;
}
.era__range {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
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
