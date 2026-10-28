import type { TimelineEvent } from '@/data/types'

/** Ano curto para a coluna da timeline: "1992" ou "1983–84". */
export function yearLabel(e: Pick<TimelineEvent, 'year' | 'endYear'>): string {
  if (!e.endYear || e.endYear === e.year) return `${e.year}`
  const end = `${e.endYear}`
  return `${e.year}–${end.slice(0, 2) === `${e.year}`.slice(0, 2) ? end.slice(2) : end}`
}

/** Data por extenso, ou o período em anos quando não há data. */
export function dateLabel(e: Pick<TimelineEvent, 'year' | 'endYear' | 'date'>): string {
  if (e.date) return e.date
  return e.endYear && e.endYear !== e.year ? `${e.year}–${e.endYear}` : `${e.year}`
}

export const months = [
  'Janeiro', 'Fevereiro', 'Março', 'Abril', 'Maio', 'Junho',
  'Julho', 'Agosto', 'Setembro', 'Outubro', 'Novembro', 'Dezembro',
]

/** Monta "29 de dezembro de 1992" a partir dos campos do formulário. */
export function composeDate(day: string, month: string, year: number): string | undefined {
  const d = Number(day)
  if (!month) return undefined
  const m = month.toLowerCase()
  if (!day || !Number.isInteger(d) || d < 1 || d > 31) return `${month} de ${year}`
  return `${d === 1 ? '1º' : d} de ${m} de ${year}`
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
