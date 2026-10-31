/*
 * Tipos no formato da API (api/openapi.yaml). Campos que a API manda como null
 * chegam aqui como undefined (ver store.ts), para os componentes que esperam
 * opcionais.
 */
import type { PeriodColor, TrackColor } from '@/design-system'

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

/**
 * Período da timeline (República Velha, Era Vargas, Estado Novo...). Podem se sobrepor.
 * Sem `end`, vai até hoje; sem `color`, a tela escolhe pela posição na lista.
 */
export type Era = {
  name: string
  start: number
  end?: number
  color?: PeriodColor
  /** Mês e dia do início e do fim, opcionais como nos eventos. */
  startMonth?: number
  startDay?: number
  endMonth?: number
  endDay?: number
}

/** Período como se manda para a API: `previousName` avisa que ele foi renomeado. */
export type EraInput = {
  name: string
  start: number
  end?: number | null
  color?: PeriodColor | null
  startMonth?: number | null
  startDay?: number | null
  endMonth?: number | null
  endDay?: number | null
  previousName?: string
}

/** Era de contagem de um calendário ficcional; os anos começam do 1 em cada era. */
export type CalendarEra = {
  id: string
  name: string
  /** Sigla que vai antes do ano: "T.A." em "T.A. 3019". */
  abbr: string
  /** Duração em anos; sem ela (só a última era), continua indefinidamente. */
  years?: number
}

/** Calendário de um mundo ficcional (Terra-média...), que várias timelines podem usar. */
export type Calendar = {
  id: string
  name: string
  description: string
  /** Handle do dono. */
  owner: string
  /** Qualquer pessoa pode usar nas próprias timelines. */
  shared: boolean
  eras: CalendarEra[]
  createdAt: string
  updatedAt: string
}

/** Unidade de uma timeline fixa: a posição dos eventos conta a partir do início (0). */
export type Scale = 'dias' | 'semanas' | 'meses' | 'anos'

/** O calendário que vem dentro da timeline. */
export type TimelineCalendar = Pick<Calendar, 'id' | 'name' | 'eras'>

/** Era como se manda para a API: sem id é uma era nova. */
export type CalendarEraInput = { id?: string; name: string; abbr: string; years: number | null }

export type CalendarInput = { name?: string; description?: string; shared?: boolean; eras?: CalendarEraInput[] }

export type Visibility = 'publica' | 'link' | 'privada'
export type ContributionMode = 'sugestoes' | 'convidados' | 'fechada'

export const categories = ['História', 'Política', 'Esporte', 'Ciência e tecnologia', 'Cultura', 'Natureza', 'Ficção', 'Pessoal'] as const
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
  /** Calendário ficcional; sem ele, a timeline usa o gregoriano. */
  calendar?: TimelineCalendar
  /**
   * Timeline fixa (ciclo de vida do lagarto...): sem calendário, o `year` de eventos e
   * períodos é a posição nessa unidade a partir do início.
   */
  scale?: Scale
  /** Timeline fixa em que o fim volta ao começo. */
  cyclic: boolean
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
  eras?: EraInput[]
  tags?: string[]
  /** Só na criação: o calendário ficcional da timeline. */
  calendarId?: string
  /** Só na criação: faz uma timeline fixa nessa escala. */
  scale?: Scale
  cyclic?: boolean
}

export type TimelinePatch = Partial<Omit<TimelineInput, 'calendarId' | 'scale'>>

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
