import { describe, expect, it } from 'vitest'
import { innermostPeriod, parentPeriod, periodColorOf, periodRange, stackPeriods } from '../components/periodLayout'

const brasil = [
  { name: 'República Velha', start: 1889, end: 1930 },
  { name: 'Estado Novo', start: 1937, end: 1945 },
  { name: 'Era Vargas', start: 1930, end: 1945 },
  { name: 'Nova República', start: 1985 },
]

describe('stackPeriods', () => {
  it('empilha só os que se sobrepõem; os que se encostam dividem a linha', () => {
    const rows = Object.fromEntries(stackPeriods(brasil, 2026).map((p) => [p.name, p.row]))
    expect(rows).toEqual({ 'República Velha': 0, 'Era Vargas': 0, 'Estado Novo': 1, 'Nova República': 0 })
  })

  it('período sem fim vai até openEnd e a cor segue a posição na lista', () => {
    const placed = stackPeriods(brasil, 2026)
    const nova = placed.find((p) => p.name === 'Nova República')!
    expect(nova.to).toBe(2026)
    expect(nova.color).toBe(periodColorOf({}, 3))
    expect(periodColorOf({ color: 'cinza' }, 0)).toBe('cinza')
  })
})

describe('períodos dentro de períodos', () => {
  it('o ano cai no período mais curto que o contém', () => {
    expect(innermostPeriod(brasil, 1940, 2026)?.name).toBe('Estado Novo')
    expect(innermostPeriod(brasil, 1932, 2026)?.name).toBe('Era Vargas')
    expect(innermostPeriod(brasil, 1960, 2026)).toBeUndefined()
    expect(innermostPeriod(brasil, 2000, 2026)?.name).toBe('Nova República')
  })

  it('acha quem contém o período', () => {
    expect(parentPeriod(brasil, brasil[1], 2026)?.name).toBe('Era Vargas')
    expect(parentPeriod(brasil, brasil[2], 2026)).toBeUndefined()
  })

  it('escreve o intervalo', () => {
    expect(periodRange({ start: 1930, end: 1945 })).toBe('1930–1945')
    expect(periodRange({ start: 1985 })).toBe('1985–hoje')
    expect(periodRange({ start: -27, end: 476 })).toBe('-27 a 476')
  })
})
