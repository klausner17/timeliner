<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { TlButton, TlField, TlInput } from '@/design-system'
import AuthCard from '@/components/AuthCard.vue'
import { ApiError, errorMessage } from '@/data/api'
import { requestPasswordReset } from '@/data/store'

const route = useRoute()
const email = ref(typeof route.query.email === 'string' ? route.query.email : '')
const error = ref('')
const fieldError = ref('')
const sending = ref(false)
const sentTo = ref('')

async function submit() {
  error.value = ''
  fieldError.value = ''
  if (!/^\S+@\S+\.\S+$/.test(email.value.trim())) {
    fieldError.value = 'Informe o e-mail da sua conta.'
    return
  }
  sending.value = true
  try {
    await requestPasswordReset(email.value.trim())
    sentTo.value = email.value.trim()
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fieldError.value = e.fields.email ?? ''
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <AuthCard v-if="sentTo" title="Confira seu e-mail">
    <p class="auth__ok">
      Se houver uma conta com <strong>{{ sentTo }}</strong>, mandamos um link para criar uma senha nova. Ele vale por 1 hora.
    </p>
    <p class="auth__note">Não chegou? Olhe a caixa de spam ou tente de novo em alguns minutos.</p>
    <TlButton :to="{ name: 'login' }">Voltar para Entrar</TlButton>
  </AuthCard>
  <AuthCard v-else title="Esqueci minha senha" lead="Informe o e-mail da conta e mandamos um link para você criar uma senha nova." form @submit="submit">
    <TlField v-slot="{ id, describedby, invalid }" label="E-mail" :error="fieldError">
      <TlInput :id="id" v-model="email" type="email" autocomplete="email" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <p v-if="error" class="auth__error" role="alert">{{ error }}</p>
    <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Enviando…' : 'Mandar link' }}</TlButton>
    <template #after>
      Lembrou? <RouterLink :to="{ name: 'login' }">Entrar</RouterLink>
    </template>
  </AuthCard>
</template>
