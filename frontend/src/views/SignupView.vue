<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { TlButton, TlField, TlInput } from '@/design-system'
import AuthCard from '@/components/AuthCard.vue'
import { ApiError, errorMessage } from '@/data/api'
import { register, signupOpen } from '@/data/store'
import { nextPath } from '@/lib/nextPath'

const route = useRoute()
const router = useRouter()

const form = reactive({ name: '', handle: '', email: '', password: '', website: '' })
const error = ref('')
const fields = ref<Record<string, string>>({})
const sending = ref(false)
const open = ref(true)
/** O @ segue o nome até a pessoa mexer nele. */
const handleTouched = ref(false)

onMounted(async () => (open.value = await signupOpen()))

/** Sugestão de @ a partir do nome: sem acento, minúsculas, _ no lugar de espaço. */
function slugHandle(name: string) {
  return name
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .slice(0, 30)
}

function onName(value: string | number | undefined) {
  const v = String(value ?? '')
  form.name = v
  if (!handleTouched.value) form.handle = slugHandle(v)
}
function onHandle(value: string | number | undefined) {
  handleTouched.value = true
  form.handle = String(value ?? '').toLowerCase().replace(/^@/, '')
}

const passwordHint = computed(() => (form.password && form.password.length < 8 ? `Faltam ${8 - form.password.length} caracteres.` : 'Pelo menos 8 caracteres.'))

function check(): boolean {
  const f: Record<string, string> = {}
  if (!form.name.trim()) f.name = 'Como você quer aparecer no site?'
  if (!/^[a-z0-9_]{3,30}$/.test(form.handle)) f.handle = 'Use de 3 a 30 letras minúsculas, números ou _.'
  if (!/^\S+@\S+\.\S+$/.test(form.email.trim())) f.email = 'E-mail inválido.'
  if (form.password.length < 8) f.password = 'A senha precisa ter pelo menos 8 caracteres.'
  fields.value = f
  return Object.keys(f).length === 0
}

async function submit() {
  error.value = ''
  if (!check()) return
  sending.value = true
  try {
    await register({ name: form.name.trim(), handle: form.handle, email: form.email.trim(), password: form.password, website: form.website })
    router.replace(nextPath(route.query.next, '/nova'))
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fields.value = e.fields
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <AuthCard v-if="!open" title="Criar conta">
    <p class="auth__note">O cadastro está fechado no momento. Sem conta, você pode explorar e comparar as timelines públicas.</p>
    <TlButton to="/" variant="primary">Explorar timelines</TlButton>
  </AuthCard>
  <AuthCard v-else title="Criar conta" lead="Monte suas timelines, sugira eventos nas dos outros e salve comparações." form @submit="submit">
    <TlField v-slot="{ id, describedby, invalid }" label="Nome" :error="fields.name">
      <TlInput :id="id" :model-value="form.name" autocomplete="name" :maxlength="80" :aria-describedby="describedby" :invalid="invalid" @update:model-value="onName" />
    </TlField>
    <TlField v-slot="{ id, describedby, invalid }" label="Nome de usuário" :hint="form.handle ? `Seu perfil: @${form.handle}` : 'Letras minúsculas, números e _'" :error="fields.handle">
      <TlInput
        :id="id"
        :model-value="form.handle"
        autocomplete="username"
        autocapitalize="none"
        spellcheck="false"
        :maxlength="30"
        :aria-describedby="describedby"
        :invalid="invalid"
        @update:model-value="onHandle"
      />
    </TlField>
    <TlField v-slot="{ id, describedby, invalid }" label="E-mail" hint="Para confirmar a conta e recuperar a senha. Não aparece no seu perfil." :error="fields.email">
      <TlInput :id="id" v-model="form.email" type="email" autocomplete="email" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <TlField v-slot="{ id, describedby, invalid }" label="Senha" :hint="passwordHint" :error="fields.password">
      <TlInput :id="id" v-model="form.password" type="password" autocomplete="new-password" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <div class="auth__trap" aria-hidden="true">
      <label>Site <input v-model="form.website" type="text" name="website" tabindex="-1" autocomplete="off" /></label>
    </div>
    <p v-if="error" class="auth__error" role="alert">{{ error }}</p>
    <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Criando…' : 'Criar conta' }}</TlButton>
    <template #after>
      Já tem conta? <RouterLink :to="{ name: 'login', query: route.query.next ? { next: route.query.next } : {} }">Entrar</RouterLink>
    </template>
  </AuthCard>
</template>
