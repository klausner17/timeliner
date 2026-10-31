import { describe, expect, it } from 'vitest'
import { formatScalePos, formatScaleRange, parseScalePos, reckoningOf, scaleTicks, scaleYearFormat } from '../scale'
import { yearFormatOf, overviewTicksFor } from '../calendar'
import { dateLabel, yearLabel } from '../format'
import { parseLine } from '../importSheet'
import { sameCalendar, rangeLabel } from '@/data/store'
import { periodRange } from '@/design-system/components/periodLayout'

describe('timelines fixas', () => {
  it('escreve a posição na unidade', () => {
    expect(formatScalePos(0, 'dias')).toBe('Dia 0')
    expect(formatScalePos(3, 'meses')).toBe('Mês 3')
    expect(formatScaleRange(12, 30, 'dias')).toBe('Dias 12–30')
    expect(formatScaleRange(2, 2, 'anos')).toBe('Ano 2')
    expect(yearLabel({ year: 60 }, 'dias')).toBe('Dia 60')
    expect(dateLabel({ year: 1, endYear: 4 }, 'semanas')).toBe('Semanas 1–4')
    expect(rangeLabel({ startYear: 0, endYear: 365, scale: 'dias' })).toBe('Dias 0–365')
  })

  it('não marca o eixo antes do início', () => {
    expect(scaleTicks(-5, 20, 10)).toEqual([0, 10, 20])
    expect(overviewTicksFor(-3, 60, 'dias').every((t) => t >= 0)).toBe(true)
    expect(yearFormatOf('dias').yearOnly).toBe(true)
  })

  it('escreve os períodos (fases) como um trecho', () => {
    expect(periodRange({ start: 0, end: 59 }, scaleYearFormat('dias'))).toBe('Dias 0–59')
    expect(periodRange({ start: 365 }, scaleYearFormat('dias'))).toBe('Dia 365 em diante')
  })

  it('lê a posição da planilha', () => {
    expect(parseScalePos('60', 'dias')).toBe(60)
    expect(parseScalePos('Dia 60', 'dias')).toBe(60)
    expect(parseScalePos('3 meses', 'meses')).toBe(3)
    expect(parseScalePos('mês 3', 'meses')).toBe(3)
    expect(parseScalePos('-1', 'dias')).toBeUndefined()
    expect(parseScalePos('12/05/2020', 'dias')).toBeUndefined()
    expect(parseLine('60;Eclosão;Sai do ovo', 'dias')).toMatchObject({ year: 60, title: 'Eclosão' })
  })

  it('só compara na mesma escala', () => {
    expect(sameCalendar({ scale: 'dias' }, { scale: 'dias' })).toBe(true)
    expect(sameCalendar({ scale: 'dias' }, { scale: 'anos' })).toBe(false)
    expect(sameCalendar({ scale: 'anos' }, {})).toBe(false)
    expect(reckoningOf({ scale: 'dias' })).toBe('dias')
    expect(reckoningOf({})).toBeUndefined()
  })
})
