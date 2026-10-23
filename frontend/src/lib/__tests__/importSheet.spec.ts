import { describe, expect, it } from 'vitest'
import { parseLine, parseSheet, parseWhen } from '../importSheet'

describe('parseWhen', () => {
  it('aceita só o ano', () => {
    expect(parseWhen('1888')).toEqual({ year: 1888 })
    expect(parseWhen('-500')).toEqual({ year: -500 })
  })

  it('aceita data no formato dd/mm/aaaa', () => {
    expect(parseWhen('13/05/1888')).toEqual({ year: 1888, date: '13 de maio de 1888' })
    expect(parseWhen('1/7/1994')).toEqual({ year: 1994, date: '1º de julho de 1994' })
  })

  it('aceita mês e ano no formato mm/aaaa', () => {
    expect(parseWhen('02/1922')).toEqual({ year: 1922, date: 'Fevereiro de 1922' })
  })

  it('aceita data no formato aaaa-mm-dd e aaaa-mm', () => {
    expect(parseWhen('1822-09-07')).toEqual({ year: 1822, date: '7 de setembro de 1822' })
    expect(parseWhen('1930-10')).toEqual({ year: 1930, date: 'Outubro de 1930' })
  })

  it('recusa mês inválido e texto', () => {
    expect(parseWhen('10/13/1990')).toBeUndefined()
    expect(parseWhen('ano')).toBeUndefined()
  })
})

describe('parseLine', () => {
  it('lê data, título, descrição e fonte', () => {
    expect(parseLine('13/05/1888; Lei Áurea; Abole a escravidão; https://x.org')).toEqual({
      year: 1888,
      date: '13 de maio de 1888',
      title: 'Lei Áurea',
      description: 'Abole a escravidão',
      sourceUrl: 'https://x.org',
    })
  })

  it('mantém o formato antigo, só com o ano', () => {
    expect(parseLine('1500,Chegada de Cabral')).toEqual({ year: 1500, title: 'Chegada de Cabral', description: '', sourceUrl: undefined })
  })

  it('recusa linha sem título', () => {
    expect(parseLine('1500;')).toBeUndefined()
  })
})

describe('parseSheet', () => {
  it('pula cabeçalho e linhas vazias', () => {
    const text = 'data;título;descrição;fonte\r\n\r\n07/09/1822;Independência\n1889;República'
    expect(parseSheet(text).map((e) => [e.year, e.date])).toEqual([
      [1822, '7 de setembro de 1822'],
      [1889, undefined],
    ])
  })
})
