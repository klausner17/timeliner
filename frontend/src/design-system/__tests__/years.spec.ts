import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { TlYearInput, axisTicks, formatYear, formatYearRange, signedYear, yearFields } from '..'

describe('formatYear', () => {
  it('marca a.C. nos anos negativos', () => {
    expect(formatYear(-500)).toBe('500 a.C.')
    expect(formatYear(1500)).toBe('1500')
  })

  it('junta o a.C. no fim do intervalo e põe d.C. quando cruza o ano 1', () => {
    expect(formatYearRange(1930, 1945)).toBe('1930–1945')
    expect(formatYearRange(-500, -450)).toBe('500–450 a.C.')
    expect(formatYearRange(-27, 14)).toBe('27 a.C.–14 d.C.')
    expect(formatYearRange(-44, -44)).toBe('44 a.C.')
  })
})

describe('signedYear', () => {
  it('vira negativo em a.C. e recusa zero, negativo e texto', () => {
    expect(signedYear('500', 'aC')).toBe(-500)
    expect(signedYear(' 1961 ', 'dC')).toBe(1961)
    expect(signedYear('0', 'dC')).toBeUndefined()
    expect(signedYear('-5', 'aC')).toBeUndefined()
    expect(signedYear('ano', 'dC')).toBeUndefined()
    expect(signedYear('', 'dC')).toBeUndefined()
  })

  it('volta para os campos do formulário', () => {
    expect(yearFields(-44)).toEqual({ text: '44', epoch: 'aC' })
    expect(yearFields(1961)).toEqual({ text: '1961', epoch: 'dC' })
    expect(yearFields(undefined)).toEqual({ text: '', epoch: 'dC' })
  })
})

describe('TlYearInput', () => {
  it('passa id e aria para o campo de texto e troca a.C./d.C. no seletor', async () => {
    const w = mount(TlYearInput, { props: { modelValue: '44', epoch: 'dC', id: 'ano', 'aria-describedby': 'dica' } })
    expect(w.find('input').attributes('id')).toBe('ano')
    expect(w.find('input').attributes('aria-describedby')).toBe('dica')
    expect(w.find('select').attributes('aria-label')).toBe('Antes ou depois de Cristo')
    await w.find('select').setValue('aC')
    expect(w.emitted('update:epoch')?.[0]).toEqual(['aC'])
  })
})

describe('axisTicks', () => {
  it('deixa folga nas pontas para os rótulos com a.C. não se encostarem', () => {
    expect(axisTicks(-753, 476)).toEqual([-753, -400, -200, 0, 200, 476])
  })
})
