/*
 * Como os componentes escrevem anos. O padrão é o calendário gregoriano ("1930", meses,
 * "até hoje"). Uma tela de timeline ficcional fornece outro com provideYearFormat
 * (ex.: "T.A. 3019", só anos e marcações que recomeçam em cada era).
 */
import { computed, inject, provide, toValue, type ComputedRef, type InjectionKey, type MaybeRefOrGetter } from 'vue'

export type YearFormat = {
  /** "1930" ou "T.A. 3019". */
  year(y: number): string
  /** Marcações do eixo em [lo, hi] com o passo dado; sem isso, múltiplos do passo. */
  ticks?(lo: number, hi: number, step: number): number[]
  /** Só anos: sem meses no eixo nem nas datas. */
  yearOnly: boolean
  /** Fim de um período que não acabou: o ano atual no gregoriano, o fim do eixo na ficção. */
  openEnd(axisEnd: number): number
  /** Como se escreve o fim de um período que não acabou ("hoje"). */
  openLabel: string
}

const THIS_YEAR = new Date().getFullYear()

export const gregorianYears: YearFormat = {
  year: (y) => `${y}`,
  yearOnly: false,
  openEnd: (axisEnd) => Math.max(THIS_YEAR, axisEnd),
  openLabel: 'hoje',
}

const KEY: InjectionKey<ComputedRef<YearFormat>> = Symbol('yearFormat')

/** A tela diz como os componentes abaixo dela escrevem os anos. */
export function provideYearFormat(format: MaybeRefOrGetter<YearFormat>) {
  provide(
    KEY,
    computed(() => toValue(format)),
  )
}

export function useYearFormat(): ComputedRef<YearFormat> {
  return inject(
    KEY,
    computed(() => gregorianYears),
    true,
  )
}
