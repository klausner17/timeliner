import { yearAt, type Storyboard } from './storyboard'

export type Cue =
  | { kind: 'tick'; time: number; frequency: number }
  | { kind: 'pling'; time: number; note: number }
  | { kind: 'chord'; time: number; notes: number[]; length: number }

export const SAMPLE_RATE = 48000
const PENTATONIC = [0, 2, 4, 7, 9]
const ROOT = 60
const TICKS_PER_TRAVEL = 14
const TICK_SCAN = 1 / 240
const INTRO_CHORD = [48, 55, 60, 64]
const OUTRO_CHORD = [48, 55, 60, 64, 67, 72]

export function plingNote(index: number): number {
  const octave = Math.floor(index / PENTATONIC.length) % 2
  return ROOT + PENTATONIC[index % PENTATONIC.length] + octave * 12
}

function travelTicks(sb: Storyboard, from: number, to: number, departed: number, arrive: number): Cue[] {
  const gap = Math.abs(to - from)
  if (gap === 0) return []
  const step = Math.max(1, gap / TICKS_PER_TRAVEL)
  const ticks: Cue[] = []
  let passed = 0
  for (let t = departed; t < arrive; t += TICK_SCAN) {
    const crossed = Math.floor(Math.abs(yearAt(sb, t) - from) / step)
    if (crossed <= passed) continue
    passed = crossed
    ticks.push({ kind: 'tick', time: t, frequency: 1400 + 900 * (crossed / TICKS_PER_TRAVEL) })
  }
  return ticks
}

export function soundCues(sb: Storyboard): Cue[] {
  const cues: Cue[] = [{ kind: 'chord', time: 0.15, notes: INTRO_CHORD, length: sb.introEnd }]
  sb.beats.forEach((beat, i) => {
    const previous = sb.beats[i - 1]
    const from = previous ? previous.event.year : sb.first
    const departed = previous ? previous.leave : sb.introEnd
    cues.push(...travelTicks(sb, from, beat.event.year, departed, beat.arrive))
    cues.push({ kind: 'pling', time: beat.arrive, note: plingNote(i) })
  })
  cues.push({ kind: 'chord', time: sb.outroStart + 0.2, notes: OUTRO_CHORD, length: sb.duration - sb.outroStart })
  return cues
}

const frequencyOf = (note: number) => 440 * 2 ** ((note - 69) / 12)

type Audio = BaseAudioContext

function envelope(ctx: Audio, destination: AudioNode, time: number, peak: number, attack: number, length: number): GainNode {
  const gain = ctx.createGain()
  gain.gain.setValueAtTime(0.0001, time)
  gain.gain.exponentialRampToValueAtTime(peak, time + attack)
  gain.gain.exponentialRampToValueAtTime(0.0001, time + length)
  gain.connect(destination)
  return gain
}

function tone(ctx: Audio, type: OscillatorType, frequency: number, time: number, length: number, gain: GainNode) {
  const osc = ctx.createOscillator()
  osc.type = type
  osc.frequency.setValueAtTime(frequency, time)
  osc.connect(gain)
  osc.start(time)
  osc.stop(time + length + 0.05)
}

function playTick(ctx: Audio, out: AudioNode, cue: Extract<Cue, { kind: 'tick' }>) {
  tone(ctx, 'triangle', cue.frequency, cue.time, 0.04, envelope(ctx, out, cue.time, 0.05, 0.002, 0.04))
}

function playPling(ctx: Audio, out: AudioNode, cue: Extract<Cue, { kind: 'pling' }>) {
  const f = frequencyOf(cue.note)
  tone(ctx, 'sine', f, cue.time, 1.2, envelope(ctx, out, cue.time, 0.22, 0.005, 1.2))
  tone(ctx, 'sine', f * 2, cue.time, 0.6, envelope(ctx, out, cue.time, 0.06, 0.005, 0.6))
}

function playChord(ctx: Audio, out: AudioNode, cue: Extract<Cue, { kind: 'chord' }>) {
  cue.notes.forEach((note, i) => {
    const gain = envelope(ctx, out, cue.time + i * 0.04, 0.06, 0.35, cue.length)
    tone(ctx, 'sine', frequencyOf(note), cue.time + i * 0.04, cue.length, gain)
  })
}

export async function renderSoundtrack(sb: Storyboard): Promise<AudioBuffer> {
  const ctx = new OfflineAudioContext(2, Math.ceil(sb.duration * SAMPLE_RATE), SAMPLE_RATE)
  const compressor = ctx.createDynamicsCompressor()
  compressor.connect(ctx.destination)
  const master = ctx.createGain()
  master.gain.value = 0.9
  master.connect(compressor)
  for (const cue of soundCues(sb)) {
    if (cue.kind === 'tick') playTick(ctx, master, cue)
    if (cue.kind === 'pling') playPling(ctx, master, cue)
    if (cue.kind === 'chord') playChord(ctx, master, cue)
  }
  return ctx.startRendering()
}
