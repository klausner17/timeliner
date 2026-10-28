<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { TlButton, TlCard, TlField, TlInput, TlSelect } from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import { createTimeline } from '@/data/store'
import { categories, type Category } from '@/data/types'

// O endereço (slug) sai do título e não muda depois, por isso o título vem antes.
const router = useRouter()
const form = reactive({ title: '', category: 'História' as Category })
const fields = ref<Record<string, string>>({})
const error = ref('')
const sending = ref(false)

async function submit() {
  fields.value = {}
  error.value = ''
  if (!form.title.trim()) {
    fields.value = { title: 'Dê um título à timeline.' }
    return
  }
  sending.value = true
  try {
    const t = await createTimeline({ title: form.title.trim(), category: form.category, draft: true, visibility: 'privada', contributions: 'fechada' })
    router.replace({ path: `/t/${t.slug}/editar`, query: { nova: '1' } })
  } catch (e) {
    error.value = errorMessage(e)
    if (e instanceof ApiError) fields.value = e.fields
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <main class="tl-container page">
    <TlCard as="form" padding="lg" class="page__card" novalidate @submit.prevent="submit">
      <h1 class="page__title">Nova timeline</h1>
      <TlField v-slot="{ id, describedby, invalid }" label="Título" :error="fields.title" hint="Também vira o endereço da timeline.">
        <TlInput :id="id" v-model="form.title" :aria-describedby="describedby" :invalid="invalid" />
      </TlField>
      <TlField v-slot="{ id }" label="Categoria">
        <TlSelect :id="id" v-model="form.category" :options="[...categories]" />
      </TlField>
      <p v-if="error" class="page__error" role="alert">{{ error }}</p>
      <div class="page__actions">
        <TlButton to="/perfil" variant="ghost">Cancelar</TlButton>
        <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Criando…' : 'Criar e adicionar eventos' }}</TlButton>
      </div>
      <p class="page__note">Ela começa como rascunho privado. Você publica quando quiser.</p>
    </TlCard>
  </main>
</template>

<style scoped>
.page {
  padding-top: var(--tl-space-14);
  padding-bottom: var(--tl-space-14);
  display: flex;
  justify-content: center;
}
.page__card {
  width: 100%;
  max-width: 520px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.page__title {
  font-size: var(--tl-text-3xl);
}
.page__actions {
  display: flex;
  justify-content: end;
  gap: var(--tl-space-2);
}
.page__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
.page__note {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
