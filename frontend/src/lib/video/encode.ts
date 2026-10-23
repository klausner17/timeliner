import { BufferTarget, CanvasSource, Mp4OutputFormat, Output, QUALITY_HIGH, WebMOutputFormat, getFirstEncodableVideoCodec } from 'mediabunny'
import { HEIGHT, WIDTH, drawFrame, loadFonts, type Theme } from './render'
import type { Storyboard } from './storyboard'

export const FPS = 30

export type VideoFile = { blob: Blob; extension: string }

async function pickFormat() {
  const codec = await getFirstEncodableVideoCodec(['avc', 'vp9', 'vp8'], { width: WIDTH, height: HEIGHT })
  if (!codec) throw new Error('Este navegador não consegue gerar vídeo.')
  if (codec === 'avc') return { codec, format: new Mp4OutputFormat({ fastStart: 'in-memory' }) }
  return { codec, format: new WebMOutputFormat() }
}

export async function renderVideo(sb: Storyboard, theme: Theme, onProgress: (done: number) => void, signal?: AbortSignal): Promise<VideoFile> {
  await loadFonts(theme)
  const canvas = new OffscreenCanvas(WIDTH, HEIGHT)
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Não deu para desenhar o vídeo.')

  const { codec, format } = await pickFormat()
  const output = new Output({ format, target: new BufferTarget() })
  const source = new CanvasSource(canvas, { codec, bitrate: QUALITY_HIGH })
  output.addVideoTrack(source, { frameRate: FPS })
  await output.start()

  const frames = Math.ceil(sb.duration * FPS)
  for (let i = 0; i < frames; i++) {
    if (signal?.aborted) {
      await output.cancel()
      throw new DOMException('Vídeo cancelado.', 'AbortError')
    }
    drawFrame(ctx, sb, theme, i / FPS)
    await source.add(i / FPS, 1 / FPS)
    onProgress((i + 1) / frames)
  }
  await output.finalize()

  const buffer = output.target.buffer
  if (!buffer) throw new Error('O vídeo saiu vazio.')
  return { blob: new Blob([buffer], { type: format.mimeType }), extension: format.fileExtension }
}
