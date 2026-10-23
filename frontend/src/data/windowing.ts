/*
 * Paginação e faixas de anos, com as mesmas regras do backend (api/openapi.yaml):
 * GET /timelines/{slug}/events, /timelines/{slug}/summary e /compare. Enquanto o app
 * roda sobre os dados de exemplo, o store usa estas funções; quando ligar na API,
 * só o store muda.
 */
import type { TimelineEvent } from './types'

export const DEFAULT_PAGE_SIZE = 100
export const DEFAULT_BUCKETS = 50
export const DEFAULT_MAX_EVENTS = 150
export const MAX_OVERLAPS = 200
const SAMPLE_SIZE = 3

export type EventPage = { items: TimelineEvent[]; nextCursor: string | null }

export type YearBucket = { from: number; to: number; count: number; sample: TimelineEvent[] }

/** Até maxEvents eventos no trecho: vêm todos. Mais que isso: faixas de anos. */
export type EventWindow = {
  from: number
  to: number
  total: number
  mode: 'eventos' | 'faixas'
  events: TimelineEvent[]
  buckets: YearBucket[]
}

export type Overlap = { year: number; laneIds: string[] }

export type WindowOptions = { buckets?: number; maxEvents?: number }

const endOf = (e: TimelineEvent) => e.endYear ?? e.year

type Key = [number, number, string]
const keyOf = (e: TimelineEvent): Key => [e.year, endOf(e), e.id]
function compareKey(a: Key, b: Key) {
  return a[0] - b[0] || a[1] - b[1] || (a[2] < b[2] ? -1 : a[2] > b[2] ? 1 : 0)
}

/** Ordem cronológica: ano, fim do período, id. */
export function chronological(events: readonly TimelineEvent[]): TimelineEvent[] {
  return [...events].sort((a, b) => compareKey(keyOf(a), keyOf(b)))
}

/** Eventos que tocam [from, to]: um período 1960–1970 entra em from=1965. */
export function touching(sorted: readonly TimelineEvent[], from?: number, to?: number): TimelineEvent[] {
  return sorted.filter((e) => (from === undefined || endOf(e) >= from) && (to === undefined || e.year <= to))
}

function encodeCursor(e: TimelineEvent) {
  return btoa(encodeURIComponent(JSON.stringify(keyOf(e))))
}

function decodeCursor(cursor: string): Key | undefined {
  try {
    const k = JSON.parse(decodeURIComponent(atob(cursor)))
    return Array.isArray(k) && k.length === 3 ? (k as Key) : undefined
  } catch {
    return undefined
  }
}

export function pageOf(sorted: readonly TimelineEvent[], opts: { from?: number; to?: number; cursor?: string | null; limit?: number } = {}): EventPage {
  const limit = opts.limit ?? DEFAULT_PAGE_SIZE
  let list = touching(sorted, opts.from, opts.to)
  if (opts.cursor) {
    const after = decodeCursor(opts.cursor)
    if (!after) throw new Error('Cursor inválido. Recarregue a página.')
    list = list.filter((e) => compareKey(keyOf(e), after) > 0)
  }
  const items = list.slice(0, limit)
  return { items, nextCursor: list.length > limit ? encodeCursor(items[items.length - 1]) : null }
}

/** Em qual das n faixas iguais de [from, from+span) cai o ano (quem começa antes de from fica na primeira). */
export function bucketOf(year: number, from: number, span: number, n: number) {
  return Math.floor(((Math.max(year, from) - from) * n) / span)
}

/** Primeiro e último ano da faixa i, na mesma conta de bucketOf. */
export function bucketRange(i: number, from: number, span: number, n: number): [number, number] {
  return [from + Math.ceil((i * span) / n), from + Math.ceil(((i + 1) * span) / n) - 1]
}

export function windowOf(sorted: readonly TimelineEvent[], from: number, to: number, opts: WindowOptions = {}): EventWindow {
  const maxEvents = opts.maxEvents ?? DEFAULT_MAX_EVENTS
  const list = touching(sorted, from, to)
  if (list.length <= maxEvents) return { from, to, total: list.length, mode: 'eventos', events: list, buckets: [] }
  const span = to - from + 1
  const n = Math.min(opts.buckets ?? DEFAULT_BUCKETS, span)
  const buckets: YearBucket[] = []
  let last = -1
  for (const e of list) {
    const b = bucketOf(e.year, from, span, n)
    if (b !== last) {
      const [lo, hi] = bucketRange(b, from, span, n)
      buckets.push({ from: lo, to: hi, count: 0, sample: [] })
      last = b
    }
    const cur = buckets[buckets.length - 1]
    cur.count++
    if (cur.sample.length < SAMPLE_SIZE) cur.sample.push(e)
  }
  return { from, to, total: list.length, mode: 'faixas', events: [], buckets }
}

/** Anos de [from, to] em que eventos de duas ou mais trilhas começam. */
export function overlapsOf(lanes: { id: string; events: readonly TimelineEvent[] }[], from: number, to: number, limit = MAX_OVERLAPS) {
  const byYear = new Map<number, string[]>()
  for (const lane of lanes) {
    for (const e of lane.events) {
      if (e.year < from || e.year > to) continue
      const ids = byYear.get(e.year) ?? []
      if (!ids.includes(lane.id)) byYear.set(e.year, [...ids, lane.id])
    }
  }
  const all: Overlap[] = [...byYear]
    .filter(([, ids]) => ids.length >= 2)
    .map(([year, ids]) => ({ year, laneIds: lanes.map((l) => l.id).filter((id) => ids.includes(id)) }))
    .sort((a, b) => a.year - b.year)
  return { overlaps: all.slice(0, limit), overlapsTotal: all.length }
}

/** Menor e maior valor sem espalhar o array em argumentos (estoura a pilha com muitos itens). */
export function extent(values: Iterable<number>): [number, number] | undefined {
  let lo = Infinity
  let hi = -Infinity
  for (const v of values) {
    if (v < lo) lo = v
    if (v > hi) hi = v
  }
  return lo <= hi ? [lo, hi] : undefined
}
