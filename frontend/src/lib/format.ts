import type { TimelineEvent } from '@/data/types'

/** Ano curto para a coluna da timeline: "1992" ou "1983–84". */
export function yearLabel(e: Pick<TimelineEvent, 'year' | 'endYear'>): string {
  if (!e.endYear || e.endYear === e.year) return `${e.year}`
  const end = `${e.endYear}`
  return `${e.year}–${end.slice(0, 2) === `${e.year}`.slice(0, 2) ? end.slice(2) : end}`
}

export type Moment = Pick<TimelineEvent, 'year' | 'month' | 'day' | 'time'>
type Dated = Moment & Pick<TimelineEvent, 'endYear'>

/** Data montada com mês, dia e hora, ou o período em anos. */
export function dateLabel(e: Dated): string {
  return momentLabel(e) ?? (e.endYear && e.endYear !== e.year ? `${e.year}–${e.endYear}` : `${e.year}`)
}

/** A data ao lado do ano na timeline, só quando ela diz mais que o ano. */
export function detailedDateLabel(e: Dated): string | undefined {
  return e.month || (e.endYear && e.endYear !== e.year) ? dateLabel(e) : undefined
}

export const months = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro',
]

export const monthOptions = [{ value: '', label: '—' }, ...months.map((m, i) => ({ value: String(i + 1), label: m }))]

/** "1º de julho de 1994, às 14:30", "Fevereiro de 1922", ou undefined sem mês. */
export function momentLabel(e: Moment): string | undefined {
  const name = e.month ? months[e.month - 1] : undefined
  if (!name) return undefined
  if (!e.day) return `${name} de ${e.year}`
  const date = `${e.day === 1 ? '1º' : e.day} de ${name.toLowerCase()} de ${e.year}`
  return e.time ? `${date}, às ${e.time}` : date
}

/** Dias do mês no calendário gregoriano, com a mesma regra de bissexto do backend. */
export function daysInMonth(year: number, month: number): number {
  if (month === 2) return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28
  return [4, 6, 9, 11].includes(month) ? 30 : 31
}

export const timePattern = /^([01]\d|2[0-3]):[0-5]\d$/

export type MomentFields = { month: string; day: string; time: string }
export type MomentErrors = Partial<Record<keyof MomentFields, string>>

/** Lê mês, dia e hora do formulário, com as mesmas regras do backend. */
export function readMoment(year: number, f: MomentFields): { value: Omit<Moment, 'year'>; errors: MomentErrors } {
  const month = f.month ? Number(f.month) : undefined
  const day = f.day.trim() ? Number(f.day) : undefined
  const time = f.time.trim() || undefined
  const errors: MomentErrors = {}
  if (day !== undefined && month === undefined) errors.day = 'Escolha o mês antes do dia.'
  else if (day !== undefined && (!Number.isInteger(day) || day < 1 || day > daysInMonth(year, month!))) errors.day = 'Este dia não existe no mês.'
  if (time && day === undefined) errors.time = 'Informe o dia antes da hora.'
  else if (time && !timePattern.test(time)) errors.time = 'Use a hora no formato 14:30.'
  return { value: { month, day, time }, errors }
}

/** Campos do formulário a partir do evento. */
export function momentFields(e: Omit<Moment, 'year'>): MomentFields {
  return { month: e.month ? String(e.month) : '', day: e.day ? String(e.day) : '', time: e.time ?? '' }
}

/** Copia um link e diz se deu certo. */
export async function copyLink(url: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(url)
    return true
  } catch {
    return false
  }
}

/** "agora", "há 5 minutos", "há 3 dias"... a partir de uma data ISO da API. */
export function timeAgo(iso: string, now = Date.now()): string {
  const s = Math.max(0, Math.round((now - Date.parse(iso)) / 1000))
  const steps: [number, string, string][] = [
    [60, 'minuto', 'minutos'],
    [3600, 'hora', 'horas'],
    [86400, 'dia', 'dias'],
    [2592000, 'mês', 'meses'],
    [31536000, 'ano', 'anos'],
  ]
  if (s < 60) return 'agora'
  let label = ''
  for (const [size, one, many] of steps) {
    if (s < size) break
    const n = Math.floor(s / size)
    label = `há ${n} ${n === 1 ? one : many}`
  }
  return label
}

/** "outubro de 2026", para "Membro desde". */
export function monthYear(iso: string): string {
  const d = new Date(iso)
  return `${months[d.getUTCMonth()].toLowerCase()} de ${d.getUTCFullYear()}`
}
