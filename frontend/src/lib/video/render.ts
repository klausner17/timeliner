import type { TrackColor } from '@/design-system'
import { HOLD, beatAt, yearAt, type Beat, type Storyboard } from './storyboard'

export const WIDTH = 1080
export const HEIGHT = 1920
const MARGIN = 90
const CONTENT = WIDTH - MARGIN * 2
const AXIS_Y = 860

export type Theme = {
  accent: string
  bg: string
  surface: string
  text: string
  muted: string
  border: string
  display: string
  body: string
}

export function readTheme(color: TrackColor): Theme {
  const css = getComputedStyle(document.documentElement)
  const token = (name: string) => css.getPropertyValue(`--tl-${name}`).trim()
  return {
    accent: token(`track-${color}`),
    bg: token('color-bg'),
    surface: token('color-surface'),
    text: token('color-text'),
    muted: token('color-text-muted'),
    border: token('color-border'),
    display: token('font-display'),
    body: token('font-body'),
  }
}

export async function loadFonts(theme: Theme) {
  await Promise.all([
    document.fonts.load(`700 100px ${theme.display}`),
    document.fonts.load(`600 100px ${theme.display}`),
    document.fonts.load(`400 40px ${theme.body}`),
    document.fonts.load(`500 40px ${theme.body}`),
  ])
}

type Ctx = CanvasRenderingContext2D | OffscreenCanvasRenderingContext2D

const clamp01 = (x: number) => Math.min(1, Math.max(0, x))
const fade = (t: number, start: number, length: number) => clamp01((t - start) / length)
const easeOut = (x: number) => 1 - (1 - x) ** 3

function ellipsize(ctx: Ctx, line: string, maxWidth: number): string {
  let text = line
  while (text && ctx.measureText(`${text}…`).width > maxWidth) text = text.slice(0, -1)
  return `${text.trimEnd()}…`
}

function wrap(ctx: Ctx, text: string, maxWidth: number, maxLines: number): string[] {
  const lines: string[] = []
  let line = ''
  for (const word of text.split(/\s+/).filter(Boolean)) {
    const candidate = line ? `${line} ${word}` : word
    if (!line || ctx.measureText(candidate).width <= maxWidth) {
      line = candidate
      continue
    }
    lines.push(line)
    line = word
  }
  if (line) lines.push(line)
  if (lines.length <= maxLines) return lines
  const kept = lines.slice(0, maxLines)
  kept[maxLines - 1] = ellipsize(ctx, kept[maxLines - 1], maxWidth)
  return kept
}

function drawLines(ctx: Ctx, lines: string[], x: number, y: number, lineHeight: number): number {
  lines.forEach((l, i) => ctx.fillText(l, x, y + i * lineHeight))
  return y + lines.length * lineHeight
}

function setText(ctx: Ctx, font: string, color: string, align: CanvasTextAlign = 'left') {
  ctx.font = font
  ctx.fillStyle = color
  ctx.textAlign = align
  ctx.textBaseline = 'alphabetic'
}

function period(sb: Storyboard) {
  return sb.first === sb.last ? `${sb.first}` : `${sb.first}–${sb.last}`
}

function drawIntro(ctx: Ctx, sb: Storyboard, theme: Theme, t: number) {
  const rise = (1 - easeOut(fade(t, 0, 0.7))) * 50
  setText(ctx, `500 34px ${theme.body}`, theme.muted, 'center')
  ctx.fillText('TIMELINER', WIDTH / 2, 640 + rise)
  setText(ctx, `700 96px ${theme.display}`, theme.text, 'center')
  const after = drawLines(ctx, wrap(ctx, sb.title, CONTENT, 4), WIDTH / 2, 780 + rise, 108)
  setText(ctx, `600 60px ${theme.display}`, theme.accent, 'center')
  ctx.fillText(period(sb), WIDTH / 2, after + 50 + rise)
  setText(ctx, `400 36px ${theme.body}`, theme.muted, 'center')
  ctx.fillText(`${sb.beats.length} momentos`, WIDTH / 2, after + 120 + rise)
}

function drawHeader(ctx: Ctx, sb: Storyboard, theme: Theme, beat: Beat | undefined) {
  setText(ctx, `500 30px ${theme.body}`, theme.muted)
  ctx.fillText('TIMELINER', MARGIN, 150)
  setText(ctx, `600 46px ${theme.display}`, theme.text)
  const after = drawLines(ctx, wrap(ctx, sb.title, CONTENT, 2), MARGIN, 215, 56)
  if (!beat?.event.era) return
  drawChip(ctx, theme, beat.event.era, after + 10)
}

function drawChip(ctx: Ctx, theme: Theme, label: string, y: number) {
  setText(ctx, `600 30px ${theme.body}`, theme.accent)
  const width = ctx.measureText(label).width + 48
  ctx.save()
  ctx.globalAlpha *= 0.12
  ctx.fillStyle = theme.accent
  ctx.beginPath()
  ctx.roundRect(MARGIN, y, width, 56, 28)
  ctx.fill()
  ctx.restore()
  ctx.fillText(label, MARGIN + 24, y + 39)
}

function drawYear(ctx: Ctx, theme: Theme, year: number) {
  setText(ctx, `700 230px ${theme.display}`, theme.text, 'center')
  ctx.fillText(`${Math.round(year)}`, WIDTH / 2, 720)
}

function axisX(sb: Storyboard, year: number) {
  const left = MARGIN + 20
  const right = WIDTH - MARGIN - 20
  const span = sb.last - sb.first
  if (span === 0) return WIDTH / 2
  return left + ((year - sb.first) / span) * (right - left)
}

function drawAxis(ctx: Ctx, sb: Storyboard, theme: Theme, year: number) {
  ctx.save()
  ctx.lineCap = 'round'
  ctx.lineWidth = 6
  ctx.strokeStyle = theme.border
  ctx.beginPath()
  ctx.moveTo(axisX(sb, sb.first), AXIS_Y)
  ctx.lineTo(axisX(sb, sb.last), AXIS_Y)
  ctx.stroke()
  ctx.strokeStyle = theme.accent
  ctx.beginPath()
  ctx.moveTo(axisX(sb, sb.first), AXIS_Y)
  ctx.lineTo(axisX(sb, year), AXIS_Y)
  ctx.stroke()
  ctx.restore()
  sb.beats.forEach((b) => drawDot(ctx, theme, axisX(sb, b.event.year), b.event.year <= year))
  drawPlayhead(ctx, theme, axisX(sb, year))
  setText(ctx, `500 30px ${theme.body}`, theme.muted, 'left')
  ctx.fillText(`${sb.first}`, axisX(sb, sb.first) - 10, AXIS_Y + 70)
  setText(ctx, `500 30px ${theme.body}`, theme.muted, 'right')
  ctx.fillText(`${sb.last}`, axisX(sb, sb.last) + 10, AXIS_Y + 70)
}

function drawDot(ctx: Ctx, theme: Theme, x: number, passed: boolean) {
  ctx.beginPath()
  ctx.arc(x, AXIS_Y, 11, 0, Math.PI * 2)
  ctx.fillStyle = passed ? theme.accent : theme.surface
  ctx.fill()
  ctx.lineWidth = 4
  ctx.strokeStyle = theme.accent
  ctx.stroke()
}

function drawPlayhead(ctx: Ctx, theme: Theme, x: number) {
  ctx.fillStyle = theme.text
  ctx.beginPath()
  ctx.roundRect(x - 3, AXIS_Y - 40, 6, 80, 3)
  ctx.fill()
}

function drawCard(ctx: Ctx, theme: Theme, beat: Beat, t: number) {
  const local = t - beat.arrive
  const enter = easeOut(fade(local, 0, 0.4))
  const leave = fade(local, HOLD - 0.3, 0.3)
  ctx.save()
  ctx.globalAlpha *= enter * (1 - leave)
  const top = 1000 + (1 - enter) * 70 - leave * 30
  const pad = 60
  const textWidth = CONTENT - pad * 2

  setText(ctx, `600 64px ${theme.display}`, theme.text)
  const titleLines = wrap(ctx, beat.event.title, textWidth, 3)
  setText(ctx, `400 38px ${theme.body}`, theme.muted)
  const descLines = beat.event.description ? wrap(ctx, beat.event.description, textWidth, 6) : []
  const height = pad + 40 + 30 + titleLines.length * 76 + (descLines.length ? 30 + descLines.length * 56 : 0) + pad - 20

  ctx.fillStyle = theme.surface
  ctx.strokeStyle = theme.border
  ctx.lineWidth = 2
  ctx.beginPath()
  ctx.roundRect(MARGIN, top, CONTENT, height, 36)
  ctx.fill()
  ctx.stroke()
  ctx.fillStyle = theme.accent
  ctx.beginPath()
  ctx.roundRect(MARGIN, top + 36, 10, height - 72, 5)
  ctx.fill()

  setText(ctx, `500 36px ${theme.body}`, theme.accent)
  ctx.fillText(beat.event.dateLabel, MARGIN + pad, top + pad + 30)
  setText(ctx, `600 64px ${theme.display}`, theme.text)
  const afterTitle = drawLines(ctx, titleLines, MARGIN + pad, top + pad + 40 + 30 + 56, 76)
  setText(ctx, `400 38px ${theme.body}`, theme.muted)
  drawLines(ctx, descLines, MARGIN + pad, afterTitle + 30, 56)
  ctx.restore()
}

function drawMain(ctx: Ctx, sb: Storyboard, theme: Theme, t: number) {
  const beat = beatAt(sb, t)
  const year = yearAt(sb, t)
  drawHeader(ctx, sb, theme, beat)
  drawYear(ctx, theme, year)
  drawAxis(ctx, sb, theme, year)
  if (beat) drawCard(ctx, theme, beat, t)
}

function drawOutro(ctx: Ctx, sb: Storyboard, theme: Theme, t: number) {
  const rise = (1 - easeOut(fade(t, sb.outroStart, 0.8))) * 50
  setText(ctx, `500 40px ${theme.body}`, theme.muted, 'center')
  ctx.fillText('Fim em', WIDTH / 2, 760 + rise)
  setText(ctx, `700 200px ${theme.display}`, theme.accent, 'center')
  ctx.fillText(`${sb.last}`, WIDTH / 2, 950 + rise)
  setText(ctx, `600 52px ${theme.display}`, theme.text, 'center')
  drawLines(ctx, wrap(ctx, sb.title, CONTENT, 3), WIDTH / 2, 1080 + rise, 64)
  setText(ctx, `500 34px ${theme.body}`, theme.muted, 'center')
  ctx.fillText('Feito no Timeliner', WIDTH / 2, 1700)
}

function drawProgress(ctx: Ctx, sb: Storyboard, theme: Theme, t: number) {
  ctx.fillStyle = theme.border
  ctx.fillRect(0, HEIGHT - 12, WIDTH, 12)
  ctx.fillStyle = theme.accent
  ctx.fillRect(0, HEIGHT - 12, WIDTH * clamp01(t / sb.duration), 12)
}

function withAlpha(ctx: Ctx, alpha: number, draw: () => void) {
  if (alpha <= 0) return
  ctx.save()
  ctx.globalAlpha = alpha
  draw()
  ctx.restore()
}

export function drawFrame(ctx: Ctx, sb: Storyboard, theme: Theme, t: number) {
  ctx.fillStyle = theme.bg
  ctx.fillRect(0, 0, WIDTH, HEIGHT)
  ctx.fillStyle = theme.accent
  ctx.fillRect(0, 0, WIDTH, 14)
  const intro = 1 - fade(t, sb.introEnd - 0.5, 0.5)
  const main = fade(t, sb.introEnd - 0.3, 0.5) * (1 - fade(t, sb.outroStart, 0.5))
  const outro = fade(t, sb.outroStart + 0.2, 0.6)
  withAlpha(ctx, intro, () => drawIntro(ctx, sb, theme, t))
  withAlpha(ctx, main, () => drawMain(ctx, sb, theme, t))
  withAlpha(ctx, outro, () => drawOutro(ctx, sb, theme, t))
  drawProgress(ctx, sb, theme, t)
}
