<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { TlButton } from '@/design-system'
import AuthCard from '@/components/AuthCard.vue'
import { errorMessage } from '@/data/api'
import { session, verifyEmail } from '@/data/store'

const route = useRoute()
const status = ref<'loading' | 'ok' | 'error'>('loading')
const error = ref('')

onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) {
    status.value = 'error'
    error.value = 'Link incompleto. Abra o link inteiro que mandamos por e-mail.'
    return
  }
  try {
    await verifyEmail(token)
    status.value = 'ok'
  } catch (e) {
    status.value = 'error'
    error.value = errorMessage(e)
  }
})
</script>

<template>
  <AuthCard v-if="status === 'loading'" title="Confirmando seu e-mail…" />
  <AuthCard v-else-if="status === 'ok'" title="E-mail confirmado">
    <p class="auth__ok">Pronto! Se um dia esquecer a senha, é por este e-mail que você recupera a conta.</p>
    <TlButton v-if="session.user" to="/nova" variant="primary">Criar uma timeline</TlButton>
    <TlButton v-else :to="{ name: 'login' }" variant="primary">Entrar</TlButton>
  </AuthCard>
  <AuthCard v-else title="Não deu para confirmar">
    <p class="auth__error" role="alert">{{ error }}</p>
    <p class="auth__note">Você pode pedir um link novo no aviso que aparece no topo do site depois de entrar.</p>
    <TlButton :to="session.user ? '/perfil' : { name: 'login' }">{{ session.user ? 'Ir para o perfil' : 'Entrar' }}</TlButton>
  </AuthCard>
</template>
