import { describe, expect, it } from 'vitest'
import { calendarBounds, eraTicks, formatEraRange, formatEraYear, fromEraYear, parseEraYear, toEraYear } from '@/lib/calendar'

const arda = {
  eras: [
    { id: 'p', name: 'Primeira Era', abbr: 'P.E.', years: 590 },
    { id: 's', name: 'Segunda Era', abbr: 'S.E.', years: 3441 },
    { id: 't', name: 'Terceira Era', abbr: 'T.A.', years: 3021 },
    { id: 'q', name: 'Quarta Era', abbr: 'Q.E.' },
  ],
}
const TA = 590 + 3441

describe('calendário ficcional', () => {
  it('converte era + ano em ano contínuo e volta', () => {
    expect(fromEraYear('t', 3019, arda)).toBe(TA + 3019)
    expect(toEraYear(TA + 3019, arda)).toMatchObject({ year: 3019, index: 2 })
    expect(fromEraYear('p', 1, arda)).toBe(1)
    expect(toEraYear(590, arda)?.era.id).toBe('p')
    expect(toEraYear(591, arda)).toMatchObject({ year: 1, index: 1 })
  })

  it('recusa ano que não cabe na era', () => {
    expect(fromEraYear('p', 591, arda)).toBeUndefined()
    expect(fromEraYear('p', 0, arda)).toBeUndefined()
    expect(fromEraYear('x', 1, arda)).toBeUndefined()
    expect(toEraYear(0, arda)).toBeUndefined()
  })

  it('escreve anos e intervalos', () => {
    expect(formatEraYear(TA + 3019, arda)).toBe('T.A. 3019')
    expect(formatEraYear(-10, arda)).toBe('P.E. -10')
    expect(formatEraRange(TA + 3018, TA + 3019, arda)).toBe('T.A. 3018–3019')
    expect(formatEraRange(TA + 3019, TA + 3021 + 6, arda)).toBe('T.A. 3019 – Q.E. 6')
  })

  it('a última era sem duração vai longe', () => {
    expect(calendarBounds(arda)).toEqual([1, TA + 3021 + 100000])
  })

  it('marca o começo das eras e múltiplos contados dentro de cada uma', () => {
    const ticks = eraTicks(1, TA + 3021, 1000, arda).map((y) => formatEraYear(y, arda))
    expect(ticks).toEqual(['P.E. 1', 'S.E. 1', 'S.E. 1000', 'S.E. 2000', 'T.A. 1', 'T.A. 1000', 'T.A. 2000'])
  })

  it('lê sigla e ano', () => {
    expect(parseEraYear('T.A. 3019', arda)).toBe(TA + 3019)
    expect(parseEraYear('terceira era 1', arda)).toBe(TA + 1)
    expect(parseEraYear('X 1', arda)).toBeUndefined()
  })
})
