/*
 * Dados do app vindos da API (api/openapi.yaml): sessão, leituras e escritas. As
 * telas guardam o que carregaram; aqui só ficam a sessão e as chamadas.
 */
import { computed, reactive } from 'vue'
import { ApiError, del, get, getToken, handleUnauthorized, patch, post, setToken } from './api'
import type {
  EventInput,
  EventPatch,
  Profile,
  SavedComparison,
  Suggestion,
  SuggestionKind,
  Timeline,
  TimelineDetail,
  TimelineEvent,
  TimelinePage,
  TimelinePatch,
  TimelineInput,
  User,
  UserSummary,
} from './types'
import type { EventPage, EventWindow, Overlap, WindowOptions } from './windowing'
import { formatYearRange } from '@/design-system/years'

// ---------- Nulos da API viram undefined ----------

type Nullable<T> = { [K in keyof T]: T[K] | null }

/** Tira os campos null, para os tipos com opcionais (endYear?, month?...). */
function clean<T extends object>(o: Nullable<T>): T {
  return Object.fromEntries(Object.entries(o).filter(([, v]) => v !== null)) as T
}

const cleanEvent = (e: Nullable<TimelineEvent>) => clean<TimelineEvent>(e)
const cleanTimeline = <T extends Timeline>(t: Nullable<T>): T => {
  const out = clean<T>(t)
  out.eras = out.eras.map((e) => clean(e))
  return out
}
const cleanWindow = (w: EventWindow): EventWindow => ({
  ...w,
  events: w.events.map(cleanEvent),
  buckets: w.buckets.map((b) => ({ ...b, sample: b.sample.map(cleanEvent) })),
})
function cleanSuggestion(s: Nullable<Suggestion>): Suggestion {
  const out = clean<Suggestion>(s)
  out.event = clean(out.event)
  if (out.previous) out.previous = cleanEvent(out.previous)
  return out
}

// ---------- Sessão ----------

export const session = reactive({ user: null as User | null, ready: false })
export const currentUser = computed(() => session.user)

handleUnauthorized(() => (session.user = null))

let starting: Promise<void> | undefined

/** Confere o token guardado uma vez, na primeira navegação. */
export function initSession(): Promise<void> {
  starting ??= (async () => {
    if (getToken()) {
      try {
        session.user = await get<User>('/me')
      } catch (e) {
        // Fora do ar: segue sem usuário, mas guarda o token para a próxima carga.
        if (e instanceof ApiError && e.status === 401) setToken(null)
      }
    }
    session.ready = true
  })()
  return starting
}

export async function login(email: string, password: string) {
  const out = await post<{ token: string; user: User }>('/auth/login', { email, password })
  setToken(out.token)
  session.user = out.user
}

export function logout() {
  setToken(null)
  session.user = null
}

export async function updateMe(input: { name?: string; bio?: string; tint?: User['tint'] }) {
  session.user = await patch<User>('/me', input)
}

// ---------- Ajudas sobre o resumo ----------

export const handleLabel = (handle: string) => `@${handle}`

export function yearRange(t: Pick<Timeline, 'startYear' | 'endYear'>): [number, number] | undefined {
  return t.startYear !== undefined && t.endYear !== undefined ? [t.startYear, t.endYear] : undefined
}

export function rangeLabel(t: Pick<Timeline, 'startYear' | 'endYear'>): string {
  const r = yearRange(t)
  return r ? formatYearRange(r[0], r[1]) : 'Sem datas'
}

export function canEdit(t: Pick<Timeline, 'owner'>) {
  return !!session.user && session.user.handle === t.owner.handle
}

/** Timelines com anos em comum, as que mais se sobrepõem primeiro ("Compare com"). */
export function relatedTo(t: Timeline, others: Timeline[], limit = 3): Timeline[] {
  const r = yearRange(t)
  if (!r) return []
  return others
    .filter((o) => o.slug !== t.slug)
    .map((o) => {
      const or = yearRange(o)
      return { o, overlap: or ? Math.min(r[1], or[1]) - Math.max(r[0], or[0]) : -1 }
    })
    .filter((x) => x.overlap > 0)
    .sort((a, b) => b.overlap - a.overlap)
    .slice(0, limit)
    .map((x) => x.o)
}

// ---------- Timelines ----------

export type TimelineQuery = {
  q?: string
  cat?: string
  tag?: string
  /** Handle do dono, ou "me" para as próprias (com rascunhos e privadas). */
  owner?: string
  openToContributions?: boolean
  sort?: 'popular' | 'recent'
  page?: number
  pageSize?: number
}

export async function listTimelines(query: TimelineQuery = {}): Promise<TimelinePage> {
  const page = await get<TimelinePage>('/timelines', query)
  return { ...page, items: page.items.map(cleanTimeline) }
}

/** Detalhe da timeline, ou undefined se não existe ou o usuário não pode ver. */
export async function getTimeline(slug: string): Promise<TimelineDetail | undefined> {
  try {
    const t = cleanTimeline(await get<TimelineDetail>(`/timelines/${encodeURIComponent(slug)}`))
    return { ...t, events: t.events.map(cleanEvent) }
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) return undefined
    throw e
  }
}

export const createTimeline = async (input: TimelineInput) => cleanTimeline(await post<TimelineDetail>('/timelines', input))
export const updateTimeline = async (slug: string, input: TimelinePatch) => cleanTimeline(await patch<TimelineDetail>(`/timelines/${slug}`, input))
export const deleteTimeline = (slug: string) => del(`/timelines/${slug}`)

export function followTimeline(slug: string, on: boolean) {
  return on ? post<void>(`/timelines/${slug}/follow`) : del(`/timelines/${slug}/follow`)
}

export async function listTags(): Promise<{ tag: string; count: number }[]> {
  return (await get<{ items: { tag: string; count: number }[] }>('/tags')).items
}

// ---------- Eventos ----------

/** GET /timelines/{slug}/events: uma página em ordem cronológica, por cursor. */
export async function loadEvents(slug: string, opts: { from?: number; to?: number; cursor?: string | null; limit?: number } = {}): Promise<EventPage> {
  const page = await get<EventPage>(`/timelines/${slug}/events`, opts)
  return { ...page, items: page.items.map(cleanEvent) }
}

/** Todos os eventos, página a página (editor e vídeo). */
export async function loadAllEvents(slug: string): Promise<TimelineEvent[]> {
  const out: TimelineEvent[] = []
  let cursor: string | null = null
  do {
    const page: EventPage = await loadEvents(slug, { cursor, limit: 500 })
    out.push(...page.items)
    cursor = page.nextCursor
  } while (cursor)
  return out
}

/** GET /timelines/{slug}/summary: os eventos do trecho, ou faixas de anos se forem muitos. */
export async function loadSummary(slug: string, opts: WindowOptions & { from?: number; to?: number } = {}): Promise<EventWindow> {
  return cleanWindow(await get<EventWindow>(`/timelines/${slug}/summary`, opts))
}

export const createEvent = async (slug: string, input: EventInput) => cleanEvent(await post<TimelineEvent>(`/timelines/${slug}/events`, input))
export const updateEvent = async (slug: string, id: string, input: EventPatch) =>
  cleanEvent(await patch<TimelineEvent>(`/timelines/${slug}/events/${id}`, input))
export const deleteEvent = (slug: string, id: string) => del(`/timelines/${slug}/events/${id}`)

// ---------- Comparação ----------

export type ComparisonLane = EventWindow & { id: string; name: string; color: Timeline['color']; timeline: Timeline }
export type Comparison = { start: number; end: number; lanes: ComparisonLane[]; overlaps: Overlap[]; overlapsTotal: number }

/** GET /compare: o mesmo nível de detalhe de loadSummary em cada trilha, mais as coincidências. */
export async function loadComparison(slugs: string[], start: number, end: number, opts: WindowOptions = {}): Promise<Comparison> {
  const c = await get<Comparison>('/compare', { slugs: slugs.join(','), start, end, ...opts })
  return { ...c, lanes: c.lanes.map((l) => ({ ...cleanWindow(l), id: l.id, name: l.name, color: l.color, timeline: cleanTimeline(l.timeline) })) }
}

export async function listSavedComparisons(): Promise<SavedComparison[]> {
  return (await get<{ items: SavedComparison[] }>('/me/comparisons')).items
}

export const saveComparison = (timelines: string[], start: number, end: number) => post<SavedComparison>('/me/comparisons', { timelines, start, end })
export const deleteSavedComparison = (id: string) => del(`/me/comparisons/${id}`)

// ---------- Sugestões ----------

export async function listSuggestions(slug: string): Promise<Suggestion[]> {
  return (await get<{ items: Nullable<Suggestion>[] }>(`/timelines/${slug}/suggestions`)).items.map(cleanSuggestion)
}

export async function suggest(slug: string, input: { kind: SuggestionKind; event: EventInput; eventId?: string; note?: string }): Promise<Suggestion> {
  return cleanSuggestion(await post<Nullable<Suggestion>>(`/timelines/${slug}/suggestions`, input))
}

export async function reviewSuggestion(id: string, decision: 'approve' | 'reject' | 'request_changes', feedback?: string): Promise<Suggestion> {
  return cleanSuggestion(await post<Nullable<Suggestion>>(`/suggestions/${id}/review`, { decision, feedback }))
}

/** Desfaz a revisão: volta para pendente e, se estava aprovada, desfaz o efeito. */
export async function reopenSuggestion(id: string): Promise<Suggestion> {
  return cleanSuggestion(await post<Nullable<Suggestion>>(`/suggestions/${id}/reopen`))
}

// ---------- Usuários ----------

export const getProfile = (handle: string) => get<Profile>(`/users/${handle}`)

export async function listFollowing(handle: string): Promise<{ users: UserSummary[]; timelines: Timeline[] }> {
  const out = await get<{ users: UserSummary[]; timelines: Timeline[] }>(`/users/${handle}/following`)
  return { users: out.users, timelines: out.timelines.map(cleanTimeline) }
}

export async function listContributions(handle: string): Promise<Suggestion[]> {
  return (await get<{ items: Nullable<Suggestion>[] }>(`/users/${handle}/contributions`)).items.map(cleanSuggestion)
}

export function followUser(handle: string, on: boolean) {
  return on ? post<void>(`/users/${handle}/follow`) : del(`/users/${handle}/follow`)
}
