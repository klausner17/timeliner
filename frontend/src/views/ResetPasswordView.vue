<script setup lang="ts">
import { ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { TlButton, TlField, TlInput } from '@/design-system'
import AuthCard from '@/components/AuthCard.vue'
import { ApiError, errorMessage } from '@/data/api'
import { confirmPasswordReset } from '@/data/store'

const route = useRoute()
const router = useRouter()
const token = typeof route.query.token === 'string' ? route.query.token : ''

const password = ref('')
const confirm = ref('')
const fields = ref<Record<string, string>>({})
const error = ref('')
const sending = ref(false)

async function submit() {
  error.value = ''
  fields.value = {}
  if (password.value.length < 8) fields.value.password = 'A senha precisa ter pelo menos 8 caracteres.'
  else if (password.value !== confirm.value) fields.value.confirm = 'As duas senhas não são iguais.'
  if (Object.keys(fields.value).length) return
  sending.value = true
  try {
    await confirmPasswordReset(token, password.value)
    router.replace({ name: 'profile' })
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fields.value = e.fields
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <AuthCard v-if="!token" title="Link incompleto">
    <p class="auth__note">Abra o link inteiro que mandamos por e-mail, ou peça outro.</p>
    <TlButton :to="{ name: 'forgot-password' }" variant="primary">Pedir outro link</TlButton>
  </AuthCard>
  <AuthCard v-else title="Criar senha nova" lead="Depois de salvar, você entra na conta e as outras sessões abertas são encerradas." form @submit="submit">
    <TlField v-slot="{ id, describedby, invalid }" label="Senha nova" hint="Pelo menos 8 caracteres." :error="fields.password">
      <TlInput :id="id" v-model="password" type="password" autocomplete="new-password" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <TlField v-slot="{ id, describedby, invalid }" label="Repita a senha" :error="fields.confirm">
      <TlInput :id="id" v-model="confirm" type="password" autocomplete="new-password" :aria-describedby="describedby" :invalid="invalid" />
    </TlField>
    <p v-if="error" class="auth__error" role="alert">
      {{ error }}
      <RouterLink v-if="!Object.keys(fields).length" :to="{ name: 'forgot-password' }">Pedir outro link</RouterLink>
    </p>
    <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Salvando…' : 'Salvar senha e entrar' }}</TlButton>
  </AuthCard>
</template>
