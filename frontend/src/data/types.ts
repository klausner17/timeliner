import type { TrackColor } from '@/design-system'

export type User = {
  handle: string
  name: string
  tint: TrackColor | 'neutral'
  bio?: string
  memberSince?: string
}

export type TimelineEvent = {
  id: string
  year: number
  /** Fim de um período (ex.: 1964–1985). */
  endYear?: number
  /** Data por extenso, quando conhecida. Sem ela, mostra só o ano. */
  date?: string
  title: string
  description: string
  sourceUrl?: string
  /** Handle de quem adicionou. */
  by: string
  /** Nome do período da timeline em que o evento entra. */
  era?: string
}

export type Era = { name: string; start: number; end?: number }

export type Visibility = 'publica' | 'link' | 'privada'
export type ContributionMode = 'sugestoes' | 'convidados' | 'fechada'

export const categories = ['História', 'Política', 'Esporte', 'Ciência e tecnologia', 'Cultura', 'Pessoal'] as const
export type Category = (typeof categories)[number]

export type Timeline = {
  slug: string
  title: string
  description: string
  category: Category
  color: TrackColor
  owner: string
  visibility: Visibility
  contributions: ContributionMode
  draft?: boolean
  eras: Era[]
  events: TimelineEvent[]
  tags: string[]
  /** Ordem de criação: maior = mais recente. */
  createdAt: number
  updatedLabel: string
  followers: number
  /** Gerada para demonstração (não vai para o localStorage). */
  generated?: boolean
}

export type SuggestionStatus = 'pendente' | 'aprovada' | 'recusada' | 'ajuste'

export type Suggestion = {
  id: string
  timeline: string
  kind: 'Novo evento' | 'Correção na descrição' | 'Correção de data'
  by: string
  when: string
  status: SuggestionStatus
  event: Omit<TimelineEvent, 'id' | 'by'>
  /** Recado de quem sugeriu para o dono. */
  note?: string
  /** O que o dono pediu para mudar. */
  feedback?: string
  /** Depois de aprovada: o evento criado ou corrigido. */
  applied?: string
  /** Na correção aprovada, o evento como era antes, para desfazer. */
  previous?: TimelineEvent
}

export type SavedComparison = {
  id: string
  timelines: string[]
  start: number
  end: number
  when: string
}
