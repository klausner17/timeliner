/*
 * Calendários ficcionais (api/openapi.yaml, Calendar). Por baixo, o ano é contínuo: o
 * ano 1 da primeira era é o ano 1 e cada era começa logo depois da anterior. Aqui ele
 * vira "T.A. 3019" e volta.
 */
import type { CalendarEra, TimelineCalendar } from '@/data/types'
import { gregorianYears, type YearFormat } from '@/design-system/components/yearFormat'
import { axisTicks, niceStep } from '@/design-system/components/trackLayout'
import { isScale, scaleTicks, scaleYearFormat, type Reckoning } from './scale'

/** Até onde vai, no eixo, a era sem duração (a última). Igual ao backend. */
export const OPEN_ERA_YEARS = 100000

export type EraSpan = { era: CalendarEra; index: number; start: number; end: number }

/** O trecho de cada era no ano contínuo, na ordem do calendário. */
export function eraSpans(cal: Pick<TimelineCalendar, 'eras'>): EraSpan[] {
  let start = 1
  return cal.eras.map((era, index) => {
    const n = era.years ?? OPEN_ERA_YEARS
    const span = { era, index, start, end: start + n - 1 }
    start += n
    return span
  })
}

/** Primeiro e último ano contínuo do calendário. */
export function calendarBounds(cal: Pick<TimelineCalendar, 'eras'>): [number, number] {
  const s = eraSpans(cal)
  return s.length ? [s[0].start, s[s.length - 1].end] : [1, 0]
}

export type EraYear = { era: CalendarEra; index: number; year: number }

/** O ano contínuo como era + ano dentro dela, ou undefined fora do calendário. */
export function toEraYear(abs: number, cal: Pick<TimelineCalendar, 'eras'>): EraYear | undefined {
  const y = Math.floor(abs)
  const s = eraSpans(cal).find((s) => y >= s.start && y <= s.end)
  return s && { era: s.era, index: s.index, year: y - s.start + 1 }
}

/** Era + ano dentro dela como ano contínuo, ou undefined se o ano não cabe na era. */
export function fromEraYear(eraId: string, year: number, cal: Pick<TimelineCalendar, 'eras'>): number | undefined {
  const s = eraSpans(cal).find((s) => s.era.id === eraId)
  if (!s || !Number.isInteger(year) || year < 1 || s.start + year - 1 > s.end) return undefined
  return s.start + year - 1
}

/** "T.A. 3019"; fora do calendário, o número como veio. */
export function formatEraYear(abs: number, cal: Pick<TimelineCalendar, 'eras'>): string {
  const e = toEraYear(abs, cal)
  if (e) return `${e.era.abbr} ${e.year}`
  // Antes do começo (a folga do eixo): conta para trás a partir da primeira era.
  const first = cal.eras[0]
  return first && abs < 1 ? `${first.abbr} ${Math.floor(abs)}` : `${Math.floor(abs)}`
}

/** "T.A. 3018–3019" na mesma era, "T.A. 3019 – Q.E. 6" entre eras. */
export function formatEraRange(from: number, to: number, cal: Pick<TimelineCalendar, 'eras'>): string {
  if (from === to) return formatEraYear(from, cal)
  const a = toEraYear(from, cal)
  const b = toEraYear(to, cal)
  if (a && b && a.index === b.index) return `${a.era.abbr} ${a.year}–${b.year}`
  return `${formatEraYear(from, cal)} – ${formatEraYear(to, cal)}`
}

/**
 * Marcações do eixo em [lo, hi]: o começo de cada era e, dentro dela, os múltiplos do passo
 * contados na própria era (T.A. 500, T.A. 1000...), sem encostar no começo da era.
 */
export function eraTicks(lo: number, hi: number, step: number, cal: Pick<TimelineCalendar, 'eras'>): number[] {
  const s = Math.max(1, Math.round(step))
  const out: number[] = []
  for (const span of eraSpans(cal)) {
    if (span.end < lo || span.start > hi) continue
    if (span.start >= lo) out.push(span.start)
    // Ano n da era fica em span.start + n - 1; múltiplos de s a partir de s.
    const first = Math.max(s, Math.ceil((lo - span.start + 1) / s) * s)
    for (let n = first; span.start + n - 1 <= Math.min(hi, span.end); n += s) {
      const pos = span.start + n - 1
      // Longe do começo desta era e do começo da próxima, para os rótulos não se encostarem.
      if (pos - span.start >= s * 0.6 && span.end + 1 - pos >= s * 0.6) out.push(pos)
    }
  }
  return out
}

/** Lê "T.A. 3019" (sigla e ano, como na planilha) com as siglas do calendário. */
export function parseEraYear(text: string, cal: Pick<TimelineCalendar, 'eras'>): number | undefined {
  const m = text.trim().match(/^(.*?)\s*(\d+)$/)
  if (!m) return undefined
  const abbr = m[1].trim().toLowerCase()
  const era = cal.eras.find((e) => e.abbr.toLowerCase() === abbr || e.name.toLowerCase() === abbr)
  return era ? fromEraYear(era.id, Number(m[2]), cal) : undefined
}

/** Como os componentes escrevem os anos de uma timeline: gregoriano, com as eras ou na escala fixa. */
export function yearFormatOf(cal?: Reckoning): YearFormat {
  if (!cal) return gregorianYears
  if (isScale(cal)) return scaleYearFormat(cal)
  return {
    year: (y) => formatEraYear(y, cal),
    ticks: (lo, hi, step) => eraTicks(lo, hi, step, cal),
    yearOnly: true,
    openEnd: (axisEnd) => axisEnd,
    openLabel: 'em diante',
  }
}

/** Marcações do minimapa: as de sempre no gregoriano, ou as das eras ou da escala (cerca de 6). */
export function overviewTicksFor(lo: number, hi: number, cal?: Reckoning): number[] {
  if (isScale(cal)) return scaleTicks(lo, hi, niceStep(hi - lo, 6))
  return cal ? eraTicks(lo, hi, niceStep(hi - lo, 6), cal) : axisTicks(lo, hi)
}
