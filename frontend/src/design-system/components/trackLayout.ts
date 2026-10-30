/*
 * Posicionamento dos rótulos nas trilhas horizontais. Os itens são eventos soltos ou
 * grupos ("23 eventos · 1930–1945"). Quando os rótulos não cabem em MAX_ROWS linhas,
 * os itens próximos viram grupos por coluna de LABEL_W pixels, e aí sempre cabem.
 */

export type CompareEvent = { id: string; year: number; title: string }
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

/** Passo das marcações do eixo para ter no máximo cerca de `target` marcações. */
export function niceStep(span: number, target = 8): number {
  for (const s of [1, 2, 5, 10, 20, 25, 50, 100, 200, 250, 500, 1000, 2000, 5000]) if (span / s <= target) return s
  return 10000
}

/** Marcações de um eixo de lo a hi: as pontas e os múltiplos de um passo legível, sem encostar nelas. */
export function axisTicks(lo: number, hi: number, target = 8): number[] {
  const step = niceStep(hi - lo, target)
  const clearance = step * 0.8
  const out = [lo]
  for (let y = Math.ceil(lo / step) * step; y <= hi; y += step) if (y - lo >= clearance && hi - y >= clearance) out.push(y)
  return hi > lo ? [...out, hi] : out
}

/**
 * Nova janela depois de um zoom com fator `factor` (< 1 aproxima) em volta de `anchor`,
 * limitada a [min, max] e a pelo menos minSpan anos.
 */
export function zoomWindow(start: number, end: number, factor: number, anchor: number, bounds: { min?: number; max?: number; minSpan?: number } = {}): [number, number] {
  const minSpan = bounds.minSpan ?? 2
  const full = bounds.min !== undefined && bounds.max !== undefined ? bounds.max - bounds.min : Infinity
  const span = Math.min(full, Math.max(minSpan, (end - start) * factor))
  const t = end > start ? (anchor - start) / (end - start) : 0.5
  return clampWindow(anchor - span * t, anchor - span * t + span, bounds)
}

/** Arredonda para anos inteiros e encaixa a janela em [min, max] sem mudar o tamanho. */
export function clampWindow(start: number, end: number, bounds: { min?: number; max?: number } = {}): [number, number] {
  let s = Math.round(start)
  let e = Math.round(end)
  if (e <= s) e = s + 1
  const span = e - s
  if (bounds.min !== undefined && s < bounds.min) [s, e] = [bounds.min, bounds.min + span]
  if (bounds.max !== undefined && e > bounds.max) [s, e] = [bounds.max - span, bounds.max]
  if (bounds.min !== undefined && s < bounds.min) s = bounds.min
  return [s, e]
}
