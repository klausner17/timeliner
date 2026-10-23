import { describe, expect, it } from 'vitest'
import type { TimelineEvent } from '../types'
import { bucketOf, bucketRange, chronological, extent, overlapsOf, pageOf, windowOf } from '../windowing'

const ev = (id: string, year: number, endYear?: number): TimelineEvent => ({ id, year, endYear, title: id, description: '', by: '@x' })
const many = (n: number, from: number, step = 1) => Array.from({ length: n }, (_, i) => ev(`e${String(i).padStart(5, '0')}`, from + i * step))

describe('pageOf', () => {
  it('pagina por cursor sem repetir nem pular, mesmo com anos repetidos', () => {
    const events = chronological([...many(7, 1900, 0), ...many(5, 1950, 10).map((e) => ({ ...e, id: `x${e.id}` }))])
    const seen: string[] = []
    let cursor: string | null = null
    let pages = 0
    do {
      const page = pageOf(events, { cursor, limit: 5 })
      seen.push(...page.items.map((e) => e.id))
      cursor = page.nextCursor
      pages++
    } while (cursor)
    expect(pages).toBe(3)
    expect(seen).toEqual(events.map((e) => e.id))
  })

  it('filtra por eventos que tocam o intervalo', () => {
    const events = chronological([ev('a', 1960, 1970), ev('b', 1980)])
    expect(pageOf(events, { from: 1965 }).items.map((e) => e.id)).toEqual(['a', 'b'])
    expect(pageOf(events, { from: 1975 }).items.map((e) => e.id)).toEqual(['b'])
  })

  it('recusa cursor inválido', () => {
    expect(() => pageOf([], { cursor: 'xyz' })).toThrow('Cursor inválido')
  })
})

describe('windowOf', () => {
  it('poucos eventos num trecho enorme vêm todos', () => {
    const w = windowOf([ev('a', -500), ev('b', 1500)], -500, 1500)
    expect(w.mode).toBe('eventos')
    expect(w.events).toHaveLength(2)
  })

  it('muitos eventos viram faixas com contagem e exemplos', () => {
    const w = windowOf(many(300, 1700), 1700, 1999, { buckets: 10 })
    expect(w.mode).toBe('faixas')
    expect(w.total).toBe(300)
    expect(w.buckets).toHaveLength(10)
    expect(w.buckets[1]).toMatchObject({ from: 1730, to: 1759, count: 30 })
    expect(w.buckets[1].sample.map((e) => e.year)).toEqual([1730, 1731, 1732])
  })

  it('faixas nunca ficam menores que um ano', () => {
    const w = windowOf(many(50, 1800, 0), 1800, 1804, { buckets: 50, maxEvents: 1 })
    expect(w.buckets).toHaveLength(1)
    expect(w.buckets[0]).toMatchObject({ from: 1800, to: 1800, count: 50 })
  })
})

describe('faixas', () => {
  it('cobrem cada ano exatamente uma vez', () => {
    for (const [from, span, n] of [[1900, 100, 7], [-500, 2001, 200], [0, 5, 5]]) {
      let next = from
      for (let i = 0; i < n; i++) {
        const [lo, hi] = bucketRange(i, from, span, n)
        expect(lo).toBe(next)
        for (let y = lo; y <= hi; y++) expect(bucketOf(y, from, span, n)).toBe(i)
        next = hi + 1
      }
      expect(next).toBe(from + span)
    }
  })
})

describe('overlapsOf', () => {
  it('acha os anos em comum na ordem das trilhas, com teto e total', () => {
    const lanes = [
      { id: 'b', events: [ev('b1', 1992), ev('b2', 2000)] },
      { id: 'a', events: [ev('a1', 1992), ev('a2', 2000), ev('a3', 2010)] },
    ]
    expect(overlapsOf(lanes, 1900, 2020, 1)).toEqual({ overlaps: [{ year: 1992, laneIds: ['b', 'a'] }], overlapsTotal: 2 })
  })
})

it('extent aguenta muitos valores', () => {
  expect(extent(Array.from({ length: 300000 }, (_, i) => i - 5))).toEqual([-5, 299994])
  expect(extent([])).toBeUndefined()
})
