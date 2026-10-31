/*
 * Timelines fixas (api/openapi.yaml, Scale): sem calendário, cada evento fica a tantos
 * dias, semanas, meses ou anos do início, como no ciclo de vida do lagarto (ovo no dia 0,
 * eclosão no dia 60...). Por baixo, o `year` do evento é essa posição.
 */
import type { Scale, TimelineCalendar } from '@/data/types'
import type { YearFormat } from '@/design-system/components/yearFormat'

export const scales: Scale[] = ['dias', 'semanas', 'meses', 'anos']

/** Maior posição aceita pelo backend. */
export const MAX_SCALE_POS = 1_000_000

const units: Record<Scale, { one: string; many: string }> = {
  dias: { one: 'Dia', many: 'Dias' },
  semanas: { one: 'Semana', many: 'Semanas' },
  meses: { one: 'Mês', many: 'Meses' },
  anos: { one: 'Ano', many: 'Anos' },
}

/** "Dia", para rótulos de campo. */
export const unitLabel = (s: Scale) => units[s].one

/** "dias", "semanas"... como aparece numa frase. */
export const scaleName = (s: Scale) => units[s].many.toLowerCase()

export const scaleOptions = scales.map((s) => ({ value: s, label: units[s].many }))

/** "Dia 12". */
export function formatScalePos(pos: number, s: Scale): string {
  return `${units[s].one} ${Math.floor(pos)}`
}

/** "Dia 12" ou "Dias 12–30". */
export function formatScaleRange(from: number, to: number, s: Scale): string {
  if (from === to) return formatScalePos(from, s)
  return `${units[s].many} ${Math.floor(from)}–${Math.floor(to)}`
}

/** Marcações em [lo, hi]: múltiplos do passo, sem nada antes do início (0). */
export function scaleTicks(lo: number, hi: number, step: number): number[] {
  const s = Math.max(1, Math.round(step))
  const out: number[] = []
  for (let p = Math.max(0, Math.ceil(lo / s) * s); p <= hi; p += s) out.push(p)
  return out
}

/** Lê "12", "Dia 12" ou "12 dias"; undefined se não for uma posição válida. */
export function parseScalePos(text: string, s: Scale): number | undefined {
  const t = text.trim().toLowerCase()
  const { one, many } = units[s]
  const m = t.match(new RegExp(`^(?:(?:${one}|${many})\\s+)?(\\d+)(?:\\s+(?:${one}|${many}))?$`, 'i'))
  if (!m) return undefined
  const n = Number(m[1])
  return n <= MAX_SCALE_POS ? n : undefined
}

export function scaleYearFormat(s: Scale): YearFormat {
  return {
    year: (y) => formatScalePos(y, s),
    range: (a, b) => formatScaleRange(a, b, s),
    ticks: scaleTicks,
    yearOnly: true,
    openEnd: (axisEnd) => axisEnd,
    openLabel: 'em diante',
  }
}

/**
 * Como uma timeline conta o tempo: um calendário ficcional, uma escala fixa ou, sem
 * nenhum dos dois, o gregoriano.
 */
export type Reckoning = TimelineCalendar | Scale

export const isScale = (r: Reckoning | undefined): r is Scale => typeof r === 'string'

/** O que a timeline usa para contar o tempo (ver Reckoning). */
export const reckoningOf = (t?: { calendar?: TimelineCalendar; scale?: Scale }): Reckoning | undefined => t?.calendar ?? t?.scale
