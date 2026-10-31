import type { TimelineCalendar, TimelineEvent } from '@/data/types'
import { parseEraYear } from './calendar'
import { daysInMonth, timePattern } from './format'

export type ImportedEvent = Pick<TimelineEvent, 'year' | 'month' | 'day' | 'time' | 'title' | 'description' | 'sourceUrl'>

type When = Pick<TimelineEvent, 'year' | 'month' | 'day' | 'time'>

const brazilianDate = /^(?:(\d{1,2})\/)?(\d{1,2})\/(-?\d{1,4})(?:\s+(\d{1,2}:\d{2}))?$/
const isoDate = /^(-?\d{1,4})-(\d{1,2})(?:-(\d{1,2})(?:[T\s](\d{1,2}:\d{2}))?)?$/

function fromParts(day: string, month: string, year: string, time: string): When | undefined {
  const m = Number(month)
  const y = Number(year)
  if (!Number.isInteger(y) || m < 1 || m > 12 || (time && !day)) return undefined
  if (!day) return { year: y, month: m }
  const d = Number(day)
  const t = time.padStart(5, '0')
  if (d < 1 || d > daysInMonth(y, m) || (time && !timePattern.test(t))) return undefined
  return time ? { year: y, month: m, day: d, time: t } : { year: y, month: m, day: d }
}

export function parseWhen(cell: string): When | undefined {
  if (/^-?\d+$/.test(cell)) return { year: Number(cell) }
  const br = cell.match(brazilianDate)
  if (br) return fromParts(br[1] ?? '', br[2], br[3], br[4] ?? '')
  const iso = cell.match(isoDate)
  if (iso) return fromParts(iso[3] ?? '', iso[2], iso[1], iso[4] ?? '')
  return undefined
}

function separatorOf(line: string): string {
  if (line.includes(';')) return ';'
  return line.includes('\t') ? '\t' : ','
}

/** Com calendário ficcional, a primeira coluna é sigla e ano ("T.A. 3019"). */
function parseEraWhen(cell: string, cal: TimelineCalendar): When | undefined {
  const year = parseEraYear(cell, cal)
  return year === undefined ? undefined : { year }
}

export function parseLine(line: string, cal?: TimelineCalendar): ImportedEvent | undefined {
  const [when, title, description = '', source = ''] = line.split(separatorOf(line)).map((c) => c.trim().replace(/^"|"$/g, ''))
  const parsed = cal ? parseEraWhen(when, cal) : parseWhen(when)
  if (!parsed || !title) return undefined
  return { ...parsed, title, description, sourceUrl: source || undefined }
}

export function parseSheet(text: string, cal?: TimelineCalendar): ImportedEvent[] {
  return text
    .split(/\r?\n/)
    .filter((l) => l.trim())
    .map((l) => parseLine(l, cal))
    .filter((e): e is ImportedEvent => e !== undefined)
}
