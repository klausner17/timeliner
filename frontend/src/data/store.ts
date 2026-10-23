/*
 * Estado do app em memória, a partir dos dados de exemplo. Quando o backend Go
 * tiver as rotas, estas funções passam a chamar a API e as telas não mudam.
 * As mudanças ficam no localStorage para sobreviver a um recarregamento.
 */
import { computed, reactive, watch } from 'vue'
import * as mock from './mock'
import type { SavedComparison, Suggestion, Timeline, TimelineEvent, User } from './types'

const STORAGE_KEY = 'timeliner:v1'

type State = {
  timelines: Timeline[]
  suggestions: Suggestion[]
  savedComparisons: SavedComparison[]
  following: string[]
  followedTimelines: string[]
  bio: string
}

function initialState(): State {
  return structuredClone({
    timelines: mock.timelines,
    suggestions: mock.suggestions,
    savedComparisons: mock.savedComparisons,
    following: mock.following,
    followedTimelines: [],
    bio: '',
  })
}

function load(): State {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...initialState(), ...JSON.parse(raw) }
  } catch {
    // Sem storage (aba anônima, testes): começa dos dados de exemplo.
  }
  return initialState()
}

export const state = reactive<State>(load())

watch(
  state,
  (s) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(s))
    } catch {
      // Ignora: o app funciona sem persistência.
    }
  },
  { deep: true },
)

/** Volta aos dados de exemplo (útil em testes e demonstrações). */
export function resetStore() {
  Object.assign(state, initialState())
}

const uid = (prefix: string) => `${prefix}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 6)}`

// ---------- Usuários ----------

export const currentUser = computed(() => findUser(mock.currentUserHandle))

export function findUser(handle: string): User {
  return mock.users.find((u) => u.handle === handle) ?? { handle, name: handle.replace(/^@/, ''), tint: 'neutral' }
}

export function isFollowingUser(handle: string) {
  return state.following.includes(handle)
}

export function toggleFollowUser(handle: string) {
  state.following = isFollowingUser(handle) ? state.following.filter((h) => h !== handle) : [...state.following, handle]
}

// ---------- Timelines ----------

export function sortedEvents(t: Timeline): TimelineEvent[] {
  return [...t.events].sort((a, b) => a.year - b.year)
}

export function yearRange(t: Timeline): [number, number] | undefined {
  if (!t.events.length) return undefined
  const years = t.events.flatMap((e) => (e.endYear ? [e.year, e.endYear] : [e.year]))
  return [Math.min(...years), Math.max(...years)]
}

export function rangeLabel(t: Timeline): string {
  const r = yearRange(t)
  return r ? (r[0] === r[1] ? `${r[0]}` : `${r[0]}–${r[1]}`) : 'Sem datas'
}

/** Posições (0–100) para a faixa decorativa dos cartões: até 5 pontos espalhados. */
export function trackPoints(t: Timeline): number[] {
  const r = yearRange(t)
  if (!r) return []
  const span = Math.max(1, r[1] - r[0])
  const pts = sortedEvents(t).map((e) => Math.round(((e.year - r[0]) / span) * 92 + 4))
  if (pts.length <= 5) return pts
  return [0, 1, 2, 3, 4].map((i) => pts[Math.round((i * (pts.length - 1)) / 4)])
}

export function contributors(t: Timeline): string[] {
  const counts = new Map<string, number>()
  for (const e of t.events) counts.set(e.by, (counts.get(e.by) ?? 0) + 1)
  // Dono primeiro, depois quem mais contribuiu.
  return [...counts.keys()].sort((a, b) => (a === t.owner ? -1 : b === t.owner ? 1 : (counts.get(b) ?? 0) - (counts.get(a) ?? 0)))
}

/** Timelines que qualquer pessoa vê na busca. */
export const publicTimelines = computed(() => state.timelines.filter((t) => t.visibility === 'publica' && !t.draft))

export function findTimeline(slug: string): Timeline | undefined {
  return state.timelines.find((t) => t.slug === slug)
}

export function timelinesOf(handle: string) {
  return state.timelines.filter((t) => t.owner === handle).sort((a, b) => a.createdAt - b.createdAt)
}

/** Quem pode ver a timeline: pública, por link ou do próprio usuário. */
export function canView(t: Timeline) {
  return t.visibility !== 'privada' || t.owner === mock.currentUserHandle
}

export function canEdit(t: Timeline) {
  return t.owner === mock.currentUserHandle
}

function slugify(title: string) {
  const base = title
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-|-$/g, '')
  let slug = base || 'timeline'
  for (let i = 2; findTimeline(slug); i++) slug = `${base}-${i}`
  return slug
}

export function createTimeline(title = 'Nova timeline'): Timeline {
  // Reaproveita um rascunho vazio em vez de acumular vários iguais.
  const blank = state.timelines.find((t) => t.owner === mock.currentUserHandle && t.draft && t.title === title && !t.events.length)
  if (blank) return blank
  const t: Timeline = {
    slug: slugify(title),
    title,
    description: '',
    category: 'História',
    color: 'a',
    owner: mock.currentUserHandle,
    visibility: 'privada',
    contributions: 'fechada',
    draft: true,
    eras: [],
    events: [],
    tags: [],
    createdAt: Math.max(0, ...state.timelines.map((x) => x.createdAt)) + 1,
    updatedLabel: 'Ainda não publicada',
    followers: 0,
  }
  state.timelines.push(t)
  return findTimeline(t.slug)!
}

export function publishTimeline(t: Timeline) {
  t.draft = false
  t.updatedLabel = 'Atualizada agora'
}

export function deleteTimeline(slug: string) {
  state.timelines = state.timelines.filter((t) => t.slug !== slug)
}

export function addEvent(t: Timeline, e: Omit<TimelineEvent, 'id' | 'by'>, by = mock.currentUserHandle): TimelineEvent {
  const event = { ...e, id: uid('e'), by }
  t.events.push(event)
  t.updatedLabel = 'Atualizada agora'
  return event
}

export function updateEvent(t: Timeline, id: string, patch: Partial<Omit<TimelineEvent, 'id'>>) {
  const e = t.events.find((x) => x.id === id)
  if (e) Object.assign(e, patch)
  t.updatedLabel = 'Atualizada agora'
}

export function removeEvent(t: Timeline, id: string) {
  t.events = t.events.filter((e) => e.id !== id)
  t.updatedLabel = 'Atualizada agora'
}

export function isFollowingTimeline(slug: string) {
  return state.followedTimelines.includes(slug)
}

export function toggleFollowTimeline(t: Timeline) {
  const on = isFollowingTimeline(t.slug)
  state.followedTimelines = on ? state.followedTimelines.filter((s) => s !== t.slug) : [...state.followedTimelines, t.slug]
  t.followers += on ? -1 : 1
}

/** Outras timelines com anos em comum, para "Comparada com frequência". */
export function relatedTimelines(t: Timeline, limit = 3): Timeline[] {
  const r = yearRange(t)
  if (!r) return []
  return publicTimelines.value
    .filter((o) => o.slug !== t.slug)
    .map((o) => {
      const or = yearRange(o)
      const overlap = or ? Math.min(r[1], or[1]) - Math.max(r[0], or[0]) : -1
      return { o, overlap }
    })
    .filter((x) => x.overlap > 0)
    .sort((a, b) => b.overlap - a.overlap)
    .slice(0, limit)
    .map((x) => x.o)
}

// ---------- Sugestões ----------

export function suggestionsFor(slug: string) {
  return state.suggestions.filter((s) => s.timeline === slug)
}

export function suggestionsBy(handle: string) {
  return state.suggestions.filter((s) => s.by === handle)
}

export function suggest(s: Omit<Suggestion, 'id' | 'status' | 'when' | 'by'>): Suggestion {
  const created: Suggestion = { ...s, id: uid('s'), status: 'pendente', when: 'agora', by: mock.currentUserHandle }
  state.suggestions.unshift(created)
  return created
}

/** Aprovar coloca o evento na timeline; desfazer tira de novo. */
export function decideSuggestion(id: string, status: 'aprovada' | 'recusada' | 'ajuste', feedback?: string) {
  const s = state.suggestions.find((x) => x.id === id)
  const t = s && findTimeline(s.timeline)
  if (!s || !t || s.status !== 'pendente') return
  s.status = status
  s.feedback = feedback
  if (status !== 'aprovada') return
  const same = s.kind !== 'Novo evento' ? t.events.find((e) => e.title === s.event.title) : undefined
  if (same) {
    s.previous = { ...same }
    s.applied = same.id
    updateEvent(t, same.id, s.event)
  } else {
    s.applied = addEvent(t, s.event, s.by).id
  }
}

export function undoSuggestion(id: string) {
  const s = state.suggestions.find((x) => x.id === id)
  const t = s && findTimeline(s.timeline)
  if (!s || !t) return
  if (s.status === 'aprovada' && s.applied) {
    if (s.previous) updateEvent(t, s.previous.id, s.previous)
    else removeEvent(t, s.applied)
  }
  s.applied = undefined
  s.previous = undefined
  s.feedback = undefined
  s.status = 'pendente'
}

// ---------- Comparações ----------

export function saveComparison(timelines: string[], start: number, end: number): SavedComparison {
  const found = findSavedComparison(timelines, start, end)
  if (found) return found
  const c: SavedComparison = { id: uid('c'), timelines: [...timelines], start, end, when: 'agora' }
  state.savedComparisons.unshift(c)
  return c
}

export function findSavedComparison(timelines: string[], start: number, end: number) {
  return state.savedComparisons.find((c) => c.start === start && c.end === end && c.timelines.join() === timelines.join())
}
