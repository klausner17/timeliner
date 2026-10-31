/*
 * Faixas de período (República Velha, Era Vargas, Estado Novo...) sobre um eixo de anos.
 * Períodos podem se sobrepor; os que se sobrepõem vão para linhas de baixo. Início e fim
 * podem ter mês e dia; no eixo eles viram anos com fração (15/11/1889 ≈ 1889,87).
 */
import { periodColors, type PeriodColor } from '../tokens'
import { gregorianYears, type YearFormat } from './yearFormat'

/** Um período como vem da timeline. Sem `end`, vai até hoje. Mês e dia são opcionais. */
export type TrackPeriod = {
  name: string
  start: number
  end?: number
  color?: PeriodColor
  startMonth?: number
  startDay?: number
  endMonth?: number
  endDay?: number
}
export type PlacedPeriod = TrackPeriod & { color: PeriodColor; from: number; to: number; row: number }

/** A cor do período: a escolhida ou, sem ela, a da posição dele na lista da timeline. */
export const periodColorOf = (p: Pick<TrackPeriod, 'color'>, index: number): PeriodColor => p.color ?? periodColors[index % periodColors.length]

const isLeap = (y: number) => (y % 4 === 0 && y % 100 !== 0) || y % 400 === 0
const DAYS = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]

/** Posição no eixo de anos: o ano mais a fração já passada dele até o mês e o dia. */
export function datePos(year: number, month?: number, day?: number): number {
  if (!month) return year
  const days = isLeap(year) ? 366 : 365
  let before = 0
  for (let m = 1; m < month; m++) before += m === 2 && isLeap(year) ? 29 : DAYS[m - 1]
  return year + (before + (day ? day - 1 : 0)) / days
}

/** Onde o período começa no eixo. */
export const periodFrom = (p: TrackPeriod) => datePos(p.start, p.startMonth, p.startDay)

/**
 * Onde o período termina no eixo. Um fim só com o ano marca o começo daquele ano, para
 * 1889–1930 e 1930–1945 se encostarem sem se sobrepor; sem fim, vai até `openEnd`.
 */
export const periodTo = (p: TrackPeriod, openEnd: number) =>
  p.end === undefined ? Math.max(openEnd, periodFrom(p)) : Math.max(periodFrom(p), datePos(p.end, p.endMonth, p.endDay))

/** Até onde o período vai para conter uma data: um fim só com o ano cobre o ano todo. */
function coversUntil(p: TrackPeriod, openEnd: number): number {
  if (p.end === undefined) return Math.max(openEnd, periodFrom(p)) + 1
  if (!p.endMonth) return p.end + 1
  if (!p.endDay) return p.endMonth === 12 ? p.end + 1 : datePos(p.end, p.endMonth + 1)
  return datePos(p.end, p.endMonth, p.endDay) + 1 / 366
}

/** Quem começa antes vem antes; no mesmo início, o mais longo (quem contém) primeiro. */
export const byStart = (a: { from: number; to: number }, b: { from: number; to: number }) => a.from - b.from || b.to - a.to

/**
 * Põe cada período na primeira linha em que ele não se sobrepõe a outro. Períodos que só
 * se encostam (um termina em 1930, o outro começa em 1930) dividem a linha.
 * `openEnd` é o ano usado como fim de quem não tem fim.
 */
export function stackPeriods(periods: TrackPeriod[], openEnd: number): PlacedPeriod[] {
  const withColor = periods.map((p, i) => ({ ...p, color: periodColorOf(p, i), from: periodFrom(p), to: periodTo(p, openEnd) }))
  const rowEnds: number[] = []
  return withColor.sort(byStart).map((p) => {
    let row = rowEnds.findIndex((end) => p.from >= end)
    if (row === -1) row = rowEnds.length
    rowEnds[row] = p.to
    return { ...p, row }
  })
}

export const MONTHS = ['jan.', 'fev.', 'mar.', 'abr.', 'maio', 'jun.', 'jul.', 'ago.', 'set.', 'out.', 'nov.', 'dez.']
const pad = (n: number) => String(n).padStart(2, '0')

/** "15/11/1889", "nov. 1889" ou "1889" (ou "T.A. 3019" com outro formato de ano). */
export function periodDate(year: number, month?: number, day?: number, fmt: YearFormat = gregorianYears): string {
  const y = fmt.year(year)
  if (month && day) return `${pad(day)}/${pad(month)}/${y}`
  if (month) return `${MONTHS[month - 1]} ${y}`
  return y
}

/** "1930–1945", "15/11/1889 a 24/10/1930", "1985–hoje", ou "-500 a -27" quando o traço confundiria. */
export function periodRange(p: Omit<TrackPeriod, 'name' | 'color'>, fmt: YearFormat = gregorianYears): string {
  const from = periodDate(p.start, p.startMonth, p.startDay, fmt)
  const to = p.end === undefined ? fmt.openLabel : periodDate(p.end, p.endMonth, p.endDay, fmt)
  if (fmt !== gregorianYears) return `${from} a ${to}`
  const plain = !p.startMonth && !p.endMonth && p.start >= 0 && (p.end ?? 0) >= 0
  return plain ? `${from}–${to}` : `${from} a ${to}`
}

/**
 * O período mais curto que contém a data (posição de `datePos`): com Era Vargas
 * (1930–1945) e Estado Novo (1937–1945), 1940 cai no Estado Novo e 1932 na Era Vargas.
 */
export function innermostPeriod<P extends TrackPeriod>(periods: P[], pos: number, openEnd: number): P | undefined {
  let best: P | undefined
  let bestSpan = Infinity
  for (const p of periods) {
    const from = periodFrom(p)
    const until = coversUntil(p, openEnd)
    if (pos < from || pos >= until) continue
    if (until - from < bestSpan) {
      best = p
      bestSpan = until - from
    }
  }
  return best
}

/** O menor período, diferente de `p`, que contém `p` inteiro (para dizer "parte da Era Vargas"). */
export function parentPeriod<P extends TrackPeriod>(periods: P[], p: P, openEnd: number): P | undefined {
  const pFrom = periodFrom(p)
  const pTo = coversUntil(p, openEnd)
  let best: P | undefined
  let bestSpan = Infinity
  for (const q of periods) {
    if (q === p) continue
    const qFrom = periodFrom(q)
    const qTo = coversUntil(q, openEnd)
    const span = qTo - qFrom
    // Mesmo intervalo exato: não é pai de ninguém, para não virar um ciclo.
    if (qFrom <= pFrom && qTo >= pTo && span > pTo - pFrom && span < bestSpan) {
      best = q
      bestSpan = span
    }
  }
  return best
}
