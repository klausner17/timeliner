import { describe, expect, it } from 'vitest'
import { dateLabel, daysInMonth, detailedDateLabel, momentLabel, readMoment, yearLabel } from '../format'

describe('momentLabel', () => {
  it('monta a data com mês, dia e hora', () => {
    expect(momentLabel({ year: 1922, month: 2 })).toBe('Fevereiro de 1922')
    expect(momentLabel({ year: 1994, month: 7, day: 1 })).toBe('1º de julho de 1994')
    expect(momentLabel({ year: 1969, month: 7, day: 20, time: '23:56' })).toBe('20 de julho de 1969, às 23:56')
    expect(momentLabel({ year: 1969 })).toBeUndefined()
  })
})

describe('dateLabel', () => {
  it('usa a data montada e, sem mês, os anos', () => {
    expect(dateLabel({ year: 1961, month: 1, day: 31, endYear: 1961 })).toBe('31 de janeiro de 1961')
    expect(dateLabel({ year: 1964, endYear: 1985 })).toBe('1964–1985')
    expect(dateLabel({ year: 1500 })).toBe('1500')
  })

  it('só mostra a data ao lado do ano quando ela diz mais que o ano', () => {
    expect(detailedDateLabel({ year: 1500 })).toBeUndefined()
    expect(detailedDateLabel({ year: 1500, month: 4, day: 22 })).toBe('22 de abril de 1500')
  })
})

describe('readMoment', () => {
  const read = (year: number, month = '', day = '', time = '') => readMoment(year, { month, day, time })

  it('lê os campos preenchidos', () => {
    expect(read(2024, '2', '29', '08:15')).toEqual({ value: { month: 2, day: 29, time: '08:15' }, errors: {} })
    expect(read(2024)).toEqual({ value: { month: undefined, day: undefined, time: undefined }, errors: {} })
  })

  it('aponta dia sem mês, dia que não existe e hora sem dia', () => {
    expect(read(2024, '', '3').errors.day).toBeTruthy()
    expect(read(1900, '2', '29').errors.day).toBeTruthy()
    expect(read(1961, '4', '31').errors.day).toBeTruthy()
    expect(read(1961, '4', '', '10:00').errors.time).toBeTruthy()
  })
})

describe('antes de Cristo', () => {
  it('mostra a.C. na data, no ano curto e no período', () => {
    expect(momentLabel({ year: -44, month: 3, day: 15 })).toBe('15 de março de 44 a.C.')
    expect(yearLabel({ year: -27, endYear: 14 })).toBe('27 a.C.–14 d.C.')
    expect(yearLabel({ year: -500 })).toBe('500 a.C.')
    expect(dateLabel({ year: -500, endYear: -450 })).toBe('500–450 a.C.')
  })

  it('usa o ano astronômico no bissexto: 1 a.C. e 5 a.C. são bissextos, 4 a.C. não', () => {
    expect(daysInMonth(-1, 2)).toBe(29)
    expect(daysInMonth(-5, 2)).toBe(29)
    expect(daysInMonth(-4, 2)).toBe(28)
  })
})
