import { describe, expect, it } from 'vitest'
import { gregorianYears } from '@/design-system'
import type { TimelineEvent } from '@/data/types'
import { OUTSIDE_PERIODS, axisFraction, buildSlides, groupByPeriod, mediaOf } from '../presentation'

const ev = (id: string, year: number, over: Partial<TimelineEvent> = {}): TimelineEvent => ({ id, year, title: id, description: '', by: 'k', ...over })

const periods = [
  { name: 'Era Vargas', start: 1930, end: 1945, color: 'azul' as const },
  { name: 'Estado Novo', start: 1937, end: 1945, color: 'laranja' as const },
]

describe('groupByPeriod', () => {
  it('sem períodos, tudo num grupo sem nome', () => {
    const g = groupByPeriod([ev('a', 1900), ev('b', 1950)], [], gregorianYears, 2000)
    expect(g).toEqual([{ name: '', events: [ev('a', 1900), ev('b', 1950)] }])
  })

  it('põe cada evento no período mais curto e marca o pai', () => {
    const g = groupByPeriod([ev('a', 1920), ev('b', 1932), ev('c', 1940), ev('d', 1941)], periods, gregorianYears, 2000)
    expect(g.map((x) => [x.name, x.events.map((e) => e.id), x.parent])).toEqual([
      [OUTSIDE_PERIODS, ['a'], undefined],
      ['Era Vargas', ['b'], undefined],
      ['Estado Novo', ['c', 'd'], 'Era Vargas'],
    ])
  })
})

describe('buildSlides', () => {
  const groups = groupByPeriod([ev('a', 1920), ev('b', 1932), ev('c', 1940)], periods, gregorianYears, 2000)

  it('abre, mostra cada período antes dos eventos dele e numera os eventos', () => {
    const s = buildSlides(groups, true)
    expect(s.map((x) => x.kind)).toEqual(['intro', 'event', 'period', 'event', 'period', 'event', 'end'])
    expect(s.filter((x) => x.kind === 'event').map((x) => (x.kind === 'event' ? x.number : 0))).toEqual([1, 2, 3])
  })

  it('sem todos os eventos, não tem o fim', () => {
    expect(buildSlides(groups, false).at(-1)?.kind).toBe('event')
  })

  it('carregar mais eventos não muda os slides que já existiam', () => {
    const first = buildSlides(groupByPeriod([ev('a', 1920), ev('b', 1932)], periods, gregorianYears, 2000), false)
    const more = buildSlides(groupByPeriod([ev('a', 1920), ev('b', 1932), ev('c', 1933), ev('d', 1940)], periods, gregorianYears, 2000), false)
    expect(more.slice(0, first.length).map((x) => x.key)).toEqual(first.map((x) => x.key))
  })
})

describe('mediaOf', () => {
  it('sem mídia, só a fonte', () => {
    expect(mediaOf(ev('a', 1, { sourceUrl: 'https://www.exemplo.com/x' }))).toEqual({
      imageUrl: undefined,
      imageAlt: undefined,
      imageCaption: undefined,
      longDescription: undefined,
      links: [{ url: 'https://www.exemplo.com/x', label: 'Fonte' }],
    })
  })

  it('lê imagem, descrição longa e links em formatos diferentes', () => {
    const e = { ...ev('a', 1, { description: 'curta' }), image: { url: 'https://img.com/a.jpg', alt: 'Foto', caption: 'Legenda' }, longDescription: 'Texto longo', links: ['https://www.wiki.org/a', { url: 'https://b.com', label: 'Livro' }, { href: 'javascript:alert(1)' }] }
    const m = mediaOf(e)
    expect(m.imageUrl).toBe('https://img.com/a.jpg')
    expect(m.imageAlt).toBe('Foto')
    expect(m.imageCaption).toBe('Legenda')
    expect(m.longDescription).toBe('Texto longo')
    expect(m.links).toEqual([
      { url: 'https://www.wiki.org/a', label: 'wiki.org' },
      { url: 'https://b.com', label: 'Livro' },
    ])
  })

  it('ignora imagem que não é http e descrição longa igual à curta', () => {
    const m = mediaOf({ ...ev('a', 1, { description: 'igual' }), imageUrl: 'javascript:x', longDescription: 'igual' })
    expect(m.imageUrl).toBeUndefined()
    expect(m.longDescription).toBeUndefined()
  })
})

describe('axisFraction', () => {
  it('fica entre 0 e 1', () => {
    expect(axisFraction({ year: 1950 }, 1900, 2000)).toBe(0.5)
    expect(axisFraction({ year: 1800 }, 1900, 2000)).toBe(0)
    expect(axisFraction({ year: 1950 }, 2000, 2000)).toBe(0)
  })
})
