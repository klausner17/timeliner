<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TlButton, TlCard, TlField, TlInput } from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import { login } from '@/data/store'

const route = useRoute()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const error = ref('')
const fields = ref<Record<string, string>>({})
const sending = ref(false)

/** Só volta para caminhos do próprio app. */
function nextPath() {
  const next = route.query.next
  return typeof next === 'string' && next.startsWith('/') && !next.startsWith('//') ? next : '/perfil'
}

async function submit() {
  error.value = ''
  fields.value = {}
  if (!form.email.trim() || !form.password) {
    error.value = 'Informe e-mail e senha.'
    return
  }
  sending.value = true
  try {
    await login(form.email.trim(), form.password)
    router.replace(nextPath())
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fields.value = e.fields
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <main class="tl-container login">
    <TlCard as="form" padding="lg" class="login__card" novalidate @submit.prevent="submit">
      <h1 class="login__title">Entrar</h1>
      <TlField v-slot="{ id, describedby, invalid }" label="E-mail" :error="fields.email">
        <TlInput :id="id" v-model="form.email" type="email" autocomplete="username" :aria-describedby="describedby" :invalid="invalid" />
      </TlField>
      <TlField v-slot="{ id, describedby, invalid }" label="Senha" :error="fields.password">
        <TlInput :id="id" v-model="form.password" type="password" autocomplete="current-password" :aria-describedby="describedby" :invalid="invalid" />
      </TlField>
      <p v-if="error" class="login__error" role="alert">{{ error }}</p>
      <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Entrando…' : 'Entrar' }}</TlButton>
      <p class="login__note">O cadastro ainda está fechado. Sem conta, você pode explorar e comparar as timelines públicas.</p>
    </TlCard>
  </main>
</template>

<style scoped>
.login {
  padding-top: var(--tl-space-14);
  padding-bottom: var(--tl-space-14);
  display: flex;
  justify-content: center;
}
.login__card {
  width: 100%;
  max-width: 420px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.login__title {
  font-size: var(--tl-text-3xl);
}
.login__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
.login__note {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
