/*
 * Posicionamento dos rótulos nas trilhas horizontais. Os itens são eventos soltos ou
 * grupos ("23 eventos · 1930–1945"). Quando os rótulos não cabem em MAX_ROWS linhas,
 * os itens próximos viram grupos por coluna de LABEL_W pixels, e aí sempre cabem.
 */

import { MONTHS, datePos } from './periodLayout'

/** Um evento na trilha. Com mês e dia, ele fica na posição exata dentro do ano. */
export type CompareEvent = { id: string; year: number; month?: number; day?: number; title: string }

/** Posição do evento no eixo de anos (ano com fração). */
export const eventPos = (e: Pick<CompareEvent, 'year' | 'month' | 'day'>) => datePos(e.year, e.month, e.day)

/** Menor janela do zoom: três meses. */
export const MIN_SPAN = 0.25
/** Uma faixa de anos com muitos eventos (vem do backend em mode=faixas). */
export type CompareBucket = { from: number; to: number; count: number; sample?: CompareEvent[] }

export type TrackItem =
  | { kind: 'event'; key: string; event: CompareEvent; year: number; px: number }
  | { kind: 'cluster'; key: string; from: number; to: number; count: number; sample: CompareEvent[]; px: number }

export type PlacedItem = TrackItem & { row: number; alignEnd: boolean }

export const LABEL_W = 176 // max-width do rótulo + folga
export const MAX_ROWS = 3

/** Encaixa cada rótulo na primeira linha livre; undefined se precisar de mais de maxRows. */
export function pack(items: TrackItem[], width: number, maxRows = MAX_ROWS): PlacedItem[] | undefined {
  const rowEnds: number[] = []
  const out: PlacedItem[] = []
  for (const item of items) {
    const alignEnd = item.px + LABEL_W > width
    const from = alignEnd ? item.px - LABEL_W : item.px
    let row = rowEnds.findIndex((end) => from >= end)
    if (row === -1) {
      if (rowEnds.length >= maxRows) return undefined
      row = rowEnds.length
    }
    rowEnds[row] = from + LABEL_W
    out.push({ ...item, row, alignEnd })
  }
  return out
}

const fromOf = (i: TrackItem) => (i.kind === 'event' ? i.year : i.from)
const toOf = (i: TrackItem) => (i.kind === 'event' ? i.year : i.to)
const countOf = (i: TrackItem) => (i.kind === 'event' ? 1 : i.count)

/** Junta num grupo os itens da mesma coluna de LABEL_W pixels; com `only`, só nessas colunas. */
export function mergeByColumn(items: TrackItem[], yearToPx: (y: number) => number, only?: Set<number>): TrackItem[] {
  const out: TrackItem[] = []
  let col = NaN
  let group: TrackItem[] = []
  const flush = () => {
    if (group.length === 1 || (only && !only.has(col))) out.push(...group)
    else if (group.length > 1) {
      const from = Math.min(...group.map(fromOf))
      const to = Math.max(...group.map(toOf))
      const sample = group.flatMap((i) => (i.kind === 'event' ? [i.event] : i.sample)).slice(0, 3)
      out.push({ kind: 'cluster', key: `c${from}-${to}`, from, to, count: group.reduce((n, i) => n + countOf(i), 0), sample, px: yearToPx((from + to + 1) / 2) })
    }
    group = []
  }
  for (const item of items) {
    const c = columnOf(item)
    if (c !== col) {
      flush()
      col = c
    }
    group.push(item)
  }
  flush()
  return out
}

const columnOf = (i: TrackItem) => Math.floor(i.px / LABEL_W)

/**
 * Rótulos de todos os itens que couberem em MAX_ROWS linhas. Onde não cabe, a coluna
 * vira um grupo; se ainda faltar espaço, todas as colunas viram grupos (aí sempre cabe).
 */
export function layout(items: TrackItem[], width: number, yearToPx: (y: number) => number): PlacedItem[] {
  const placed = pack(items, width)
  if (placed) return placed
  const crowded = new Set<number>()
  const rowEnds: number[] = []
  for (const item of items) {
    const from = item.px + LABEL_W > width ? item.px - LABEL_W : item.px
    const row = rowEnds.findIndex((end) => from >= end)
    if (row !== -1) rowEnds[row] = from + LABEL_W
    else if (rowEnds.length < MAX_ROWS) rowEnds.push(from + LABEL_W)
    else crowded.add(columnOf(item))
  }
  return pack(mergeByColumn(items, yearToPx, crowded), width) ?? pack(mergeByColumn(items, yearToPx), width, Infinity)!
}

/**
 * Passo das marcações do eixo para ter no máximo cerca de `target` marcações. Em janelas
 * de até dois anos, o passo pode ser de meses (1/12, 1/6, 1/4 ou 1/2 ano).
 */
export function niceStep(span: number, target = 8): number {
  if (span <= 2) for (const m of [1, 2, 3, 6]) if (span / (m / 12) <= target) return m / 12
  for (const s of [1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 5000]) if (span / s <= target) return s
  return 10000
}

/** Marcações de um eixo de lo a hi: as pontas e os múltiplos de um passo legível, sem encostar nelas. */
export function axisTicks(lo: number, hi: number, target = 8): number[] {
  const step = niceStep(hi - lo, target)
  const out = [lo]
  for (let y = Math.ceil(lo / step) * step; y <= hi; y += step) if (y - lo >= step * 0.6 && hi - y >= step * 0.6) out.push(y)
  return hi > lo ? [...out, hi] : out
}

/** Marcações da janela [lo, hi] com o passo dado, contadas em meses para não acumular erro. */
export function windowTicks(lo: number, hi: number, step: number): number[] {
  const stepM = Math.max(1, Math.round(step * 12))
  const out: number[] = []
  for (let m = Math.ceil((lo * 12) / stepM) * stepM; m <= hi * 12 + 1e-9; m += stepM) out.push(m / 12)
  return out
}

/** Rótulo de uma posição do eixo: "1930" no começo do ano, "out." nos outros meses. */
export function tickLabel(pos: number): string {
  const months = Math.round(pos * 12)
  const m = months - Math.floor(months / 12) * 12
  return m === 0 ? `${months / 12}` : MONTHS[m]
}

/** Posição como texto: "1930" ou, no meio do ano, "out. 1930". */
export function posLabel(pos: number): string {
  const months = Math.round(pos * 12)
  const y = Math.floor(months / 12)
  const m = months - y * 12
  return m === 0 ? `${y}` : `${MONTHS[m]} ${y}`
}

/**
 * Nova janela depois de um zoom com fator `factor` (< 1 aproxima) em volta de `anchor`,
 * limitada a [min, max] e a pelo menos minSpan anos.
 */
export function zoomWindow(start: number, end: number, factor: number, anchor: number, bounds: { min?: number; max?: number; minSpan?: number } = {}): [number, number] {
  const minSpan = bounds.minSpan ?? MIN_SPAN
  const full = bounds.min !== undefined && bounds.max !== undefined ? bounds.max - bounds.min : Infinity
  const span = Math.min(full, Math.max(minSpan, (end - start) * factor))
  const t = end > start ? (anchor - start) / (end - start) : 0.5
  return clampWindow(anchor - span * t, anchor - span * t + span, bounds)
}

/** Arredonda para meses inteiros e encaixa a janela em [min, max] sem mudar o tamanho. */
export function clampWindow(start: number, end: number, bounds: { min?: number; max?: number } = {}): [number, number] {
  // Conta em meses inteiros para a janela não acumular frações.
  let s = Math.round(start * 12)
  let e = Math.round(end * 12)
  if (e <= s) e = s + 1
  const span = e - s
  const min = bounds.min !== undefined ? Math.round(bounds.min * 12) : undefined
  const max = bounds.max !== undefined ? Math.round(bounds.max * 12) : undefined
  if (min !== undefined && s < min) [s, e] = [min, min + span]
  if (max !== undefined && e > max) [s, e] = [max - span, max]
  if (min !== undefined && s < min) s = min
  return [s / 12, e / 12]
}
