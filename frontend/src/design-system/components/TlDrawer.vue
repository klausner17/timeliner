<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import TlButton from './TlButton.vue'

/**
 * Painel lateral sobre a página (embaixo, no celular). Usa o <dialog> nativo: Esc fecha,
 * o foco fica dentro dele e volta para onde estava ao fechar. Clicar fora também fecha.
 */
const props = defineProps<{
  open: boolean
  /** Nome do painel para leitores de tela. */
  label: string
}>()
const emit = defineEmits<{ 'update:open': [value: boolean] }>()

const dialog = ref<HTMLDialogElement>()

function sync() {
  const d = dialog.value
  if (!d) return
  if (props.open && !d.open) {
    // jsdom não tem showModal.
    if (typeof d.showModal === 'function') d.showModal()
    else d.setAttribute('open', '')
  } else if (!props.open && d.open) {
    if (typeof d.close === 'function') d.close()
    else d.removeAttribute('open')
  }
}
watch([() => props.open, dialog], sync, { flush: 'post' })
onBeforeUnmount(() => dialog.value?.open && dialog.value.close?.())

function onClick(ev: MouseEvent) {
  // O clique no fundo cai no próprio <dialog>; dentro do painel cai nos filhos.
  if (ev.target === dialog.value) emit('update:open', false)
}
</script>

<template>
  <dialog ref="dialog" class="tl-drawer" :aria-label="label" @close="emit('update:open', false)" @click="onClick">
    <div class="tl-drawer__panel">
      <div class="tl-drawer__head">
        <TlButton icon="close" icon-only variant="ghost" shape="rounded" label="Fechar" @click="emit('update:open', false)" />
      </div>
      <div class="tl-drawer__body">
        <slot v-if="open" />
      </div>
    </div>
  </dialog>
</template>

<style scoped>
.tl-drawer {
  position: fixed;
  inset: 0 0 0 auto;
  margin: 0;
  padding: 0;
  width: min(480px, 100vw);
  max-width: 100vw;
  height: 100dvh;
  max-height: 100dvh;
  border: 0;
  background: transparent;
  color: inherit;
}
.tl-drawer::backdrop {
  background: rgb(21 23 28 / 0.4);
}
.tl-drawer__panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--tl-color-surface);
  border-left: 1px solid var(--tl-color-border);
  box-shadow: -12px 0 32px rgb(21 23 28 / 0.12);
}
.tl-drawer__head {
  display: flex;
  justify-content: flex-end;
  padding: var(--tl-space-2) var(--tl-space-2) 0;
}
.tl-drawer__body {
  flex: 1;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 var(--tl-space-6) var(--tl-space-7);
}
.tl-drawer[open] .tl-drawer__panel {
  animation: tl-drawer-in 180ms ease-out;
}
@keyframes tl-drawer-in {
  from {
    transform: translateX(24px);
    opacity: 0;
  }
}

@media (max-width: 640px) {
  .tl-drawer {
    inset: auto 0 0 0;
    width: 100vw;
    height: auto;
    max-height: 88dvh;
  }
  .tl-drawer__panel {
    max-height: 88dvh;
    border-left: 0;
    border-radius: var(--tl-radius-2xl) var(--tl-radius-2xl) 0 0;
    box-shadow: 0 -12px 32px rgb(21 23 28 / 0.12);
  }
  .tl-drawer__body {
    padding: 0 var(--tl-space-4) var(--tl-space-6);
  }
  .tl-drawer[open] .tl-drawer__panel {
    animation-name: tl-drawer-up;
  }
  @keyframes tl-drawer-up {
    from {
      transform: translateY(24px);
      opacity: 0;
    }
  }
}
@media (prefers-reduced-motion: reduce) {
  .tl-drawer[open] .tl-drawer__panel {
    animation: none;
  }
}
</style>
