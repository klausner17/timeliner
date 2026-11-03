<script setup lang="ts">
import { ref } from 'vue'
import { TlButton } from '@/design-system'
import { errorMessage } from '@/data/api'
import { resendVerification } from '@/data/store'

/** Aviso de e-mail ainda não confirmado, com o botão de mandar o link de novo. */
defineProps<{ email: string }>()

const state = ref<'idle' | 'sending' | 'sent'>('idle')
const error = ref('')

async function resend() {
  state.value = 'sending'
  error.value = ''
  try {
    await resendVerification()
    state.value = 'sent'
  } catch (e) {
    state.value = 'idle'
    error.value = errorMessage(e)
  }
}
</script>

<template>
  <div class="email-banner" role="status">
    <div class="tl-container email-banner__inner">
      <p class="email-banner__text">
        <strong>Confirme seu e-mail.</strong>
        <template v-if="state === 'sent'"> Mandamos um link novo para {{ email }}.</template>
        <template v-else> Mandamos um link para {{ email }}. Sem ele, você não cria timelines nem sugere eventos, e não consegue recuperar a senha.</template>
        <span v-if="error" class="email-banner__error"> {{ error }}</span>
      </p>
      <TlButton v-if="state !== 'sent'" size="sm" :disabled="state === 'sending'" @click="resend">{{ state === 'sending' ? 'Enviando…' : 'Mandar de novo' }}</TlButton>
    </div>
  </div>
</template>

<style scoped>
.email-banner {
  background: var(--tl-color-warning-soft);
  border-bottom: 1px solid var(--tl-color-border);
}
.email-banner__inner {
  padding-top: var(--tl-space-3);
  padding-bottom: var(--tl-space-3);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-2) var(--tl-space-4);
}
.email-banner__text {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text);
  flex: 1 1 280px;
}
.email-banner__error {
  color: var(--tl-color-warning-strong);
}
</style>
