import {
  AudioBufferSource,
  BufferTarget,
  CanvasSource,
  Mp4OutputFormat,
  Output,
  QUALITY_HIGH,
  WebMOutputFormat,
  getFirstEncodableAudioCodec,
  getFirstEncodableVideoCodec,
} from 'mediabunny'
import { HEIGHT, WIDTH, drawFrame, loadFonts, type Theme } from './render'
import { SAMPLE_RATE, renderSoundtrack } from './sound'
import type { Storyboard } from './storyboard'

export const FPS = 30

export type VideoFile = { blob: Blob; extension: string }

async function pickFormat() {
  const video = await getFirstEncodableVideoCodec(['avc', 'vp9', 'vp8'], { width: WIDTH, height: HEIGHT })
  if (!video) throw new Error('Este navegador não consegue gerar vídeo.')
  const audioOptions = { numberOfChannels: 2, sampleRate: SAMPLE_RATE }
  if (video === 'avc') {
    const audio = await getFirstEncodableAudioCodec(['aac', 'opus'], audioOptions)
    return { video, audio, format: new Mp4OutputFormat({ fastStart: 'in-memory' }) }
  }
  const audio = await getFirstEncodableAudioCodec(['opus', 'vorbis'], audioOptions)
  return { video, audio, format: new WebMOutputFormat() }
}

export async function renderVideo(sb: Storyboard, theme: Theme, onProgress: (done: number) => void, signal?: AbortSignal): Promise<VideoFile> {
  await loadFonts(theme)
  const canvas = new OffscreenCanvas(WIDTH, HEIGHT)
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('Não deu para desenhar o vídeo.')

  const { video, audio, format } = await pickFormat()
  const output = new Output({ format, target: new BufferTarget() })
  const source = new CanvasSource(canvas, { codec: video, bitrate: QUALITY_HIGH })
  output.addVideoTrack(source, { frameRate: FPS })
  const sound = audio ? new AudioBufferSource({ codec: audio, bitrate: QUALITY_HIGH }) : undefined
  if (sound) output.addAudioTrack(sound)
  await output.start()

  if (sound) {
    await sound.add(await renderSoundtrack(sb))
    sound.close()
  }

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
