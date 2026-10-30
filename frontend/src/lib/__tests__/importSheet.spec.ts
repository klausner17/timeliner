import { describe, expect, it } from 'vitest'
import { parseLine, parseSheet, parseWhen } from '../importSheet'

describe('parseWhen', () => {
  it('aceita só o ano', () => {
    expect(parseWhen('1888')).toEqual({ year: 1888 })
    expect(parseWhen('-500')).toEqual({ year: -500 })
  })

  it('aceita data no formato dd/mm/aaaa', () => {
    expect(parseWhen('13/05/1888')).toEqual({ year: 1888, month: 5, day: 13 })
    expect(parseWhen('1/7/1994')).toEqual({ year: 1994, month: 7, day: 1 })
  })

  it('aceita mês e ano no formato mm/aaaa', () => {
    expect(parseWhen('02/1922')).toEqual({ year: 1922, month: 2 })
  })

  it('aceita data no formato aaaa-mm-dd e aaaa-mm', () => {
    expect(parseWhen('1822-09-07')).toEqual({ year: 1822, month: 9, day: 7 })
    expect(parseWhen('1930-10')).toEqual({ year: 1930, month: 10 })
  })

  it('aceita hora depois do dia', () => {
    expect(parseWhen('13/05/1888 14:30')).toEqual({ year: 1888, month: 5, day: 13, time: '14:30' })
    expect(parseWhen('1969-07-20T23:56')).toEqual({ year: 1969, month: 7, day: 20, time: '23:56' })
    expect(parseWhen('1969-07-20 9:05')).toEqual({ year: 1969, month: 7, day: 20, time: '09:05' })
  })

  it('recusa mês, dia e hora inválidos e texto', () => {
    expect(parseWhen('10/13/1990')).toBeUndefined()
    expect(parseWhen('29/02/1900')).toBeUndefined()
    expect(parseWhen('31/04/1961')).toBeUndefined()
    expect(parseWhen('13/05/1888 24:00')).toBeUndefined()
    expect(parseWhen('05/1888 10:00')).toBeUndefined()
    expect(parseWhen('ano')).toBeUndefined()
  })

  it('aceita 29 de fevereiro em ano bissexto', () => {
    expect(parseWhen('29/02/2000')).toEqual({ year: 2000, month: 2, day: 29 })
  })
})

describe('parseLine', () => {
  it('lê data, título, descrição e fonte', () => {
    expect(parseLine('13/05/1888; Lei Áurea; Abole a escravidão; https://x.org')).toEqual({
      year: 1888,
      month: 5,
      day: 13,
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
    expect(parseSheet(text).map((e) => [e.year, e.month, e.day])).toEqual([
      [1822, 9, 7],
      [1889, undefined, undefined],
    ])
  })
})
