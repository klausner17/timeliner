<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { TlButton } from '@/design-system'
import { sortedEvents } from '@/data/store'
import type { Timeline, TimelineEvent } from '@/data/types'
import { dateLabel } from '@/lib/format'
import { readTheme } from '@/lib/video/render'
import { buildStoryboard, type StoryEvent } from '@/lib/video/storyboard'

const props = defineProps<{ timeline: Timeline }>()

type State =
  | { kind: 'idle' }
  | { kind: 'rendering'; percent: number }
  | { kind: 'ready'; url: string; fileName: string }
  | { kind: 'error'; message: string }

const state = ref<State>({ kind: 'idle' })
const dialog = ref<HTMLDialogElement>()
let abort: AbortController | undefined

const toStory = (e: TimelineEvent): StoryEvent => ({ year: e.year, dateLabel: dateLabel(e), title: e.title, description: e.description, era: e.era })
const canExport = computed(() => props.timeline.events.length > 0)

function release() {
  if (state.value.kind === 'ready') URL.revokeObjectURL(state.value.url)
  state.value = { kind: 'idle' }
}

function showProgress(done: number) {
  const percent = Math.round(done * 100)
  if (state.value.kind === 'rendering' && state.value.percent !== percent) state.value = { kind: 'rendering', percent }
}

async function generate() {
  release()
  dialog.value?.showModal()
  const sb = buildStoryboard(props.timeline.title, sortedEvents(props.timeline).map(toStory))
  if (!sb) {
    state.value = { kind: 'error', message: 'A timeline ainda não tem eventos.' }
    return
  }
  state.value = { kind: 'rendering', percent: 0 }
  abort = new AbortController()
  const { signal } = abort
  try {
    const { renderVideo } = await import('@/lib/video/encode')
    const file = await renderVideo(sb, readTheme(props.timeline.color), showProgress, signal)
    state.value = { kind: 'ready', url: URL.createObjectURL(file.blob), fileName: `${props.timeline.slug}${file.extension}` }
  } catch (e) {
    if (signal.aborted) return
    state.value = { kind: 'error', message: e instanceof Error ? e.message : 'Não deu para gerar o vídeo.' }
  }
}

function close() {
  abort?.abort()
  dialog.value?.close()
}

function onClosed() {
  abort?.abort()
  release()
}

onBeforeUnmount(() => {
  abort?.abort()
  release()
})
</script>

<template>
  <TlButton :disabled="!canExport" @click="generate">Gerar vídeo</TlButton>
  <dialog ref="dialog" class="video" aria-labelledby="video-title" @close="onClosed">
    <div class="video__head">
      <h2 id="video-title" class="video__title">Vídeo da timeline</h2>
      <TlButton icon="close" icon-only size="sm" label="Fechar" @click="close" />
    </div>

    <div v-if="state.kind === 'rendering'" class="video__progress" role="status">
      <p>Gerando o vídeo… {{ state.percent }}%</p>
      <progress :value="state.percent" max="100" aria-label="Progresso do vídeo"></progress>
      <p class="video__hint">O vídeo é montado aqui no navegador. Deixe esta janela aberta.</p>
    </div>

    <template v-else-if="state.kind === 'ready'">
      <video class="video__player" :src="state.url" controls playsinline></video>
      <div class="video__actions">
        <TlButton :href="state.url" :download="state.fileName" variant="primary">Baixar vídeo</TlButton>
      </div>
    </template>

    <p v-else-if="state.kind === 'error'" class="video__error" role="alert">{{ state.message }}</p>
  </dialog>
</template>

<style scoped>
.video {
  width: min(420px, calc(100vw - 32px));
  max-height: calc(100vh - 32px);
  padding: var(--tl-space-5);
  border: 1px solid var(--tl-color-border);
  border-radius: 18px;
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
}
.video::backdrop {
  background: rgb(21 23 28 / 0.5);
}
.video[open] {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.video__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-3);
}
.video__title {
  font-size: var(--tl-text-xl);
}
.video__progress {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.video__progress progress {
  width: 100%;
  accent-color: var(--tl-color-text);
}
.video__hint {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.video__player {
  width: 100%;
  max-height: 60vh;
  aspect-ratio: 9 / 16;
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-bg);
}
.video__actions {
  display: flex;
  justify-content: flex-end;
}
.video__error {
  color: var(--tl-color-text-muted);
}
</style>
