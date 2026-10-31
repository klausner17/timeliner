/*
 * Faixas de período (República Velha, Era Vargas, Estado Novo...) sobre um eixo de anos.
 * Períodos podem se sobrepor; os que se sobrepõem vão para linhas de baixo.
 */
import { periodColors, type PeriodColor } from '../tokens'

/** Um período como vem da timeline. Sem `end`, vai até hoje. */
export type TrackPeriod = { name: string; start: number; end?: number; color?: PeriodColor }
export type PlacedPeriod = { name: string; start: number; end?: number; color: PeriodColor; to: number; row: number }

/** A cor do período: a escolhida ou, sem ela, a da posição dele na lista da timeline. */
export const periodColorOf = (p: Pick<TrackPeriod, 'color'>, index: number): PeriodColor => p.color ?? periodColors[index % periodColors.length]

/** Quem começa antes vem antes; no mesmo início, o mais longo (quem contém) primeiro. */
export const byStart = (a: { start: number; to: number }, b: { start: number; to: number }) => a.start - b.start || b.to - a.to

/**
 * Põe cada período na primeira linha em que ele não se sobrepõe a outro. Períodos que só
 * se encostam (um termina em 1930, o outro começa em 1930) dividem a linha.
 * `openEnd` é o ano usado como fim de quem não tem fim.
 */
export function stackPeriods(periods: TrackPeriod[], openEnd: number): PlacedPeriod[] {
  const withColor = periods.map((p, i) => ({ ...p, color: periodColorOf(p, i), to: p.end ?? Math.max(openEnd, p.start) }))
  const rowEnds: number[] = []
  return withColor.sort(byStart).map((p) => {
    let row = rowEnds.findIndex((end) => p.start >= end)
    if (row === -1) row = rowEnds.length
    rowEnds[row] = p.to
    return { ...p, row }
  })
}

/** "1930–1945", "1985–hoje", ou "-500 a -27" quando o traço confundiria com o sinal. */
export function periodRange(p: Pick<TrackPeriod, 'start' | 'end'>): string {
  const end = p.end ?? 'hoje'
  return p.start < 0 || (p.end ?? 0) < 0 ? `${p.start} a ${end}` : `${p.start}–${end}`
}

/**
 * O período mais curto que contém o ano: com Era Vargas (1930–1945) e Estado Novo
 * (1937–1945), 1940 cai no Estado Novo e 1932 na Era Vargas.
 */
export function innermostPeriod<P extends TrackPeriod>(periods: P[], year: number, openEnd: number): P | undefined {
  let best: P | undefined
  let bestSpan = Infinity
  for (const p of periods) {
    const to = p.end ?? Math.max(openEnd, p.start)
    if (year < p.start || year > to) continue
    if (to - p.start < bestSpan) {
      best = p
      bestSpan = to - p.start
    }
  }
  return best
}

/** O menor período, diferente de `p`, que contém `p` inteiro (para dizer "parte da Era Vargas"). */
export function parentPeriod<P extends TrackPeriod>(periods: P[], p: P, openEnd: number): P | undefined {
  const pTo = p.end ?? Math.max(openEnd, p.start)
  let best: P | undefined
  let bestSpan = Infinity
  for (const q of periods) {
    if (q === p) continue
    const qTo = q.end ?? Math.max(openEnd, q.start)
    const span = qTo - q.start
    // Mesmo intervalo exato: não é pai de ninguém, para não virar um ciclo.
    if (q.start <= p.start && qTo >= pTo && span > pTo - p.start && span < bestSpan) {
      best = q
      bestSpan = span
    }
  }
  return best
}
