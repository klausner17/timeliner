<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { TlButton, TlField, TlInput } from '@/design-system'
import AuthCard from '@/components/AuthCard.vue'
import { ApiError, errorMessage } from '@/data/api'
import { login, signupOpen } from '@/data/store'
import { nextPath } from '@/lib/nextPath'

const route = useRoute()
const router = useRouter()

const form = reactive({ email: '', password: '' })
const error = ref('')
const fields = ref<Record<string, string>>({})
const sending = ref(false)
const canSignUp = ref(true)

onMounted(async () => (canSignUp.value = await signupOpen()))

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
    router.replace(nextPath(route.query.next))
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fields.value = e.fields
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <AuthCard title="Entrar" form @submit="submit">
    <TlField v-slot="{ id, describedby, invalid }" label="E-mail" :error="fields.email">
      <TlInput :id="id" v-model="form.email" type="email" autocomplete="username" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <TlField v-slot="{ id, describedby, invalid }" label="Senha" :error="fields.password">
      <TlInput :id="id" v-model="form.password" type="password" autocomplete="current-password" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <div class="auth__row">
      <RouterLink :to="{ name: 'forgot-password', query: form.email ? { email: form.email.trim() } : {} }">Esqueci minha senha</RouterLink>
    </div>
    <p v-if="error" class="auth__error" role="alert">{{ error }}</p>
    <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Entrando…' : 'Entrar' }}</TlButton>
    <p v-if="!canSignUp" class="auth__note">O cadastro está fechado no momento. Sem conta, você pode explorar e comparar as timelines públicas.</p>
    <template v-if="canSignUp" #after>
      Ainda não tem conta? <RouterLink :to="{ name: 'signup', query: route.query.next ? { next: route.query.next } : {} }">Criar conta</RouterLink>
    </template>
  </AuthCard>
</template>
