/*
 * Tipos no formato da API (api/openapi.yaml). Campos que a API manda como null
 * chegam aqui como undefined (ver store.ts), para os componentes que esperam
 * opcionais.
 */
import type { TrackColor } from '@/design-system'

export type Tint = TrackColor | 'neutral'

export type UserSummary = {
  handle: string
  name: string
  tint: Tint
}

/** O usuário logado (GET /me). */
export type User = UserSummary & {
  email: string
  bio: string
  memberSince: string
}

/** Perfil público (GET /users/{handle}). */
export type Profile = UserSummary & {
  bio: string
  memberSince: string
  following: boolean
  stats: { timelines: number; events: number; approvedContributions: number; followers: number }
}

export type TimelineEvent = {
  id: string
  year: number
  /** Mês de 1 a 12. */
  month?: number
  /** Dia do mês; só existe com o mês. */
  day?: number
  /** Hora "14:30"; só existe com o dia. */
  time?: string
  /** Fim de um período (ex.: 1964–1985). */
  endYear?: number
  title: string
  description: string
  sourceUrl?: string
  /** Handle de quem adicionou. */
  by: string
  /** Nome do período da timeline em que o evento entra. */
  era?: string
  createdAt?: string
}

/** O que se manda para criar ou corrigir um evento. */
export type EventInput = Omit<TimelineEvent, 'id' | 'by' | 'createdAt'>

/** Correção de evento: só os campos mandados mudam; null apaga um opcional. */
export type EventPatch = { [K in keyof EventInput]?: EventInput[K] | null }

export type Era = { name: string; start: number; end?: number }

export type Visibility = 'publica' | 'link' | 'privada'
export type ContributionMode = 'sugestoes' | 'convidados' | 'fechada'

export const categories = ['História', 'Política', 'Esporte', 'Ciência e tecnologia', 'Cultura', 'Pessoal'] as const
export type Category = (typeof categories)[number]

/** Resumo usado em listas e cartões (sem os eventos). */
export type Timeline = {
  slug: string
  title: string
  description: string
  category: Category
  color: TrackColor
  owner: UserSummary
  visibility: Visibility
  contributions: ContributionMode
  draft: boolean
  eras: Era[]
  tags: string[]
  startYear?: number
  endYear?: number
  eventCount: number
  contributorCount: number
  pendingSuggestions: number
  followers: number
  following: boolean
  /** Posições de 0 a 100 dos eventos no eixo, para o TlMiniTrack. */
  points: number[]
  createdAt: string
  updatedAt: string
}

/** Detalhe: o resumo mais a primeira página de eventos. */
export type TimelineDetail = Timeline & { events: TimelineEvent[]; eventsNextCursor: string | null }

export type TimelineInput = {
  title: string
  description?: string
  category: Category
  color?: TrackColor
  visibility?: Visibility
  contributions?: ContributionMode
  draft?: boolean
  eras?: Era[]
  tags?: string[]
}

export type TimelinePatch = Partial<TimelineInput>

export type TimelinePage = { items: Timeline[]; page: number; pageSize: number; total: number }

export type SuggestionStatus = 'pendente' | 'aprovada' | 'recusada' | 'ajuste'
export type SuggestionKind = 'new_event' | 'description_fix' | 'date_fix'

export const suggestionKindLabel: Record<SuggestionKind, string> = {
  new_event: 'Novo evento',
  description_fix: 'Correção na descrição',
  date_fix: 'Correção de data',
}

export type Suggestion = {
  id: string
  timeline: { slug: string; title: string; color: TrackColor }
  kind: SuggestionKind
  /** Handle de quem sugeriu. */
  by: string
  status: SuggestionStatus
  /** Evento corrigido, nas correções. */
  eventId?: string
  event: EventInput
  /** Recado de quem sugeriu para o dono. */
  note?: string
  /** O que o dono pediu para mudar ou por que recusou. */
  feedback?: string
  /** Depois de aprovada: o evento criado ou corrigido. */
  applied?: string
  /** Na correção aprovada, o evento como era antes. */
  previous?: TimelineEvent
  createdAt: string
  reviewedAt?: string
}

export type SavedComparison = {
  id: string
  timelines: string[]
  start: number
  end: number
  createdAt: string
}
