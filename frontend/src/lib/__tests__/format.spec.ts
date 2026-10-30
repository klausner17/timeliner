import { describe, expect, it } from 'vitest'
import { dateLabel, detailedDateLabel, momentLabel, readMoment } from '../format'

describe('momentLabel', () => {
  it('monta a data com mês, dia e hora', () => {
    expect(momentLabel({ year: 1922, month: 2 })).toBe('Fevereiro de 1922')
    expect(momentLabel({ year: 1994, month: 7, day: 1 })).toBe('1º de julho de 1994')
    expect(momentLabel({ year: 1969, month: 7, day: 20, time: '23:56' })).toBe('20 de julho de 1969, às 23:56')
    expect(momentLabel({ year: 1969 })).toBeUndefined()
  })
})

describe('dateLabel', () => {
  it('prefere o rótulo livre, depois a data montada, depois os anos', () => {
    expect(dateLabel({ year: 1969, month: 7, date: 'verão de 1969' })).toBe('verão de 1969')
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
