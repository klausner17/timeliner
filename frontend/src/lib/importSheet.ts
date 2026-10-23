import type { TimelineEvent } from '@/data/types'
import { composeDate, months } from './format'

export type ImportedEvent = Pick<TimelineEvent, 'year' | 'date' | 'title' | 'description' | 'sourceUrl'>

type When = Pick<TimelineEvent, 'year' | 'date'>

const brazilianDate = /^(?:(\d{1,2})\/)?(\d{1,2})\/(-?\d{1,4})$/
const isoDate = /^(-?\d{1,4})-(\d{1,2})(?:-(\d{1,2}))?$/

function fromParts(day: string, month: string, year: string): When | undefined {
  const label = months[Number(month) - 1]
  const y = Number(year)
  if (!label || !Number.isInteger(y)) return undefined
  return { year: y, date: composeDate(day, label, y) }
}

export function parseWhen(cell: string): When | undefined {
  if (/^-?\d+$/.test(cell)) return { year: Number(cell) }
  const br = cell.match(brazilianDate)
  if (br) return fromParts(br[1] ?? '', br[2], br[3])
  const iso = cell.match(isoDate)
  if (iso) return fromParts(iso[3] ?? '', iso[2], iso[1])
  return undefined
}

function separatorOf(line: string): string {
  if (line.includes(';')) return ';'
  return line.includes('\t') ? '\t' : ','
}

export function parseLine(line: string): ImportedEvent | undefined {
  const [when, title, description = '', source = ''] = line.split(separatorOf(line)).map((c) => c.trim().replace(/^"|"$/g, ''))
  const parsed = parseWhen(when)
  if (!parsed || !title) return undefined
  return { ...parsed, title, description, sourceUrl: source || undefined }
}

export function parseSheet(text: string): ImportedEvent[] {
  return text
    .split(/\r?\n/)
    .filter((l) => l.trim())
    .map(parseLine)
    .filter((e): e is ImportedEvent => e !== undefined)
}
