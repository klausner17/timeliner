import { describe, expect, it } from 'vitest'
import { plingNote, soundCues, type Cue } from '../sound'
import { buildStoryboard, type StoryEvent } from '../storyboard'

const ev = (year: number): StoryEvent => ({ year, dateLabel: `${year}`, title: `Evento ${year}`, description: '' })
const sb = buildStoryboard('Teste', [ev(1900), ev(1950), ev(2000)])!
const cues = soundCues(sb)
const of = <K extends Cue['kind']>(kind: K) => cues.filter((c): c is Extract<Cue, { kind: K }> => c.kind === kind)

describe('soundCues', () => {
  it('abre e fecha com um acorde', () => {
    const chords = of('chord')
    expect(chords).toHaveLength(2)
    expect(chords[0].time).toBeLessThan(sb.introEnd)
    expect(chords[1].time).toBeGreaterThan(sb.outroStart)
  })

  it('toca uma nota quando cada evento chega', () => {
    expect(of('pling').map((c) => c.time)).toEqual(sb.beats.map((b) => b.arrive))
  })

  it('faz tique só nas viagens entre eventos, sem passar de 14 por viagem', () => {
    const ticks = of('tick')
    const [a, b, c] = sb.beats
    const between = (from: number, to: number) => ticks.filter((t) => t.time >= from && t.time < to).length
    expect(between(0, a.arrive)).toBe(0)
    expect(between(a.leave, b.arrive)).toBeGreaterThan(5)
    expect(between(a.leave, b.arrive)).toBeLessThanOrEqual(14)
    expect(between(b.leave, c.arrive)).toBeLessThanOrEqual(14)
    expect(ticks.every((t) => !sb.beats.some((beat) => t.time > beat.arrive && t.time < beat.leave))).toBe(true)
  })
})

describe('plingNote', () => {
  it('sobe pela escala pentatônica e volta depois de duas oitavas', () => {
    expect([0, 1, 2, 3, 4, 5].map(plingNote)).toEqual([60, 62, 64, 67, 69, 72])
    expect(plingNote(10)).toBe(60)
  })
})
