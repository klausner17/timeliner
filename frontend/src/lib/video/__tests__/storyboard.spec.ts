import { describe, expect, it } from 'vitest'
import { HOLD, INTRO, OUTRO, beatAt, buildStoryboard, pickHighlights, yearAt, type StoryEvent } from '../storyboard'

const ev = (year: number): StoryEvent => ({ year, dateLabel: `${year}`, title: `Evento ${year}`, description: '' })

describe('pickHighlights', () => {
  it('mantém todos quando cabem', () => {
    expect(pickHighlights([ev(1), ev(2)], 5).map((e) => e.year)).toEqual([1, 2])
  })

  it('espalha a escolha e guarda o primeiro e o último', () => {
    const events = Array.from({ length: 100 }, (_, i) => ev(1900 + i))
    const picked = pickHighlights(events, 5).map((e) => e.year)
    expect(picked).toEqual([1900, 1925, 1950, 1974, 1999])
  })
})

describe('buildStoryboard', () => {
  it('não monta vídeo sem eventos', () => {
    expect(buildStoryboard('Vazia', [])).toBeUndefined()
  })

  it('ordena os eventos e soma abertura, eventos e fechamento', () => {
    const sb = buildStoryboard('Teste', [ev(2000), ev(1990)])!
    expect(sb.beats.map((b) => b.event.year)).toEqual([1990, 2000])
    expect([sb.first, sb.last]).toEqual([1990, 2000])
    expect(sb.beats[0].arrive).toBeGreaterThan(INTRO)
    expect(sb.beats[1].arrive).toBeGreaterThan(sb.beats[0].leave)
    expect(sb.duration).toBeCloseTo(sb.beats[1].leave + OUTRO)
    expect(sb.beats[1].leave - sb.beats[1].arrive).toBeCloseTo(HOLD)
  })
})

describe('yearAt e beatAt', () => {
  const sb = buildStoryboard('Teste', [ev(1900), ev(2000)])!
  const [a, b] = sb.beats

  it('fica no primeiro ano durante a abertura', () => {
    expect(yearAt(sb, 0)).toBe(1900)
    expect(beatAt(sb, 0)).toBeUndefined()
  })

  it('para no ano do evento enquanto o cartão aparece', () => {
    expect(yearAt(sb, a.arrive + 0.1)).toBe(1900)
    expect(beatAt(sb, a.arrive + 0.1)).toBe(a)
  })

  it('anda entre os anos na viagem até o próximo evento', () => {
    const middle = (a.leave + b.arrive) / 2
    expect(yearAt(sb, middle)).toBeCloseTo(1950)
    expect(beatAt(sb, middle)).toBeUndefined()
  })

  it('termina no último ano', () => {
    expect(yearAt(sb, sb.duration)).toBe(2000)
  })
})
