export type StoryEvent = { year: number; dateLabel: string; title: string; description: string; era?: string }

export type Beat = { event: StoryEvent; arrive: number; leave: number }

export type Storyboard = {
  title: string
  first: number
  last: number
  beats: Beat[]
  introEnd: number
  outroStart: number
  duration: number
}

export const INTRO = 2.5
export const OUTRO = 3
export const HOLD = 2.4
export const MAX_EVENTS = 18
const MIN_TRAVEL = 0.4
const MAX_TRAVEL = 1.4

export function pickHighlights(events: StoryEvent[], max = MAX_EVENTS): StoryEvent[] {
  if (events.length <= max) return events
  const step = (events.length - 1) / (max - 1)
  return Array.from({ length: max }, (_, i) => events[Math.round(i * step)])
}

function travelTime(gap: number, averageGap: number): number {
  if (averageGap <= 0) return MIN_TRAVEL
  return Math.min(MAX_TRAVEL, MIN_TRAVEL + (gap / averageGap) * 0.5)
}

function scheduleBeats(events: StoryEvent[]): Beat[] {
  const span = events[events.length - 1].year - events[0].year
  const averageGap = span / Math.max(1, events.length - 1)
  const beats: Beat[] = []
  let clock = INTRO
  events.forEach((event, i) => {
    const gap = i === 0 ? 0 : event.year - events[i - 1].year
    const arrive = clock + travelTime(gap, averageGap)
    beats.push({ event, arrive, leave: arrive + HOLD })
    clock = arrive + HOLD
  })
  return beats
}

export function buildStoryboard(title: string, events: StoryEvent[]): Storyboard | undefined {
  const chosen = pickHighlights([...events].sort((a, b) => a.year - b.year))
  if (!chosen.length) return undefined
  const beats = scheduleBeats(chosen)
  const outroStart = beats[beats.length - 1].leave
  return {
    title,
    first: chosen[0].year,
    last: chosen[chosen.length - 1].year,
    beats,
    introEnd: INTRO,
    outroStart,
    duration: outroStart + OUTRO,
  }
}

export const easeInOut = (x: number) => (x < 0.5 ? 4 * x * x * x : 1 - (-2 * x + 2) ** 3 / 2)

export function yearAt(sb: Storyboard, t: number): number {
  const next = sb.beats.findIndex((b) => t < b.leave)
  if (next === -1) return sb.last
  const beat = sb.beats[next]
  if (t >= beat.arrive) return beat.event.year
  const from = next === 0 ? sb.first : sb.beats[next - 1].event.year
  const departed = next === 0 ? sb.introEnd : sb.beats[next - 1].leave
  const progress = Math.max(0, (t - departed) / (beat.arrive - departed))
  return from + (beat.event.year - from) * easeInOut(progress)
}

export function beatAt(sb: Storyboard, t: number): Beat | undefined {
  return sb.beats.find((b) => t >= b.arrive && t < b.leave)
}
