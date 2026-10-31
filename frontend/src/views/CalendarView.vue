<script setup lang="ts">
/*
 * Um calendário ficcional: quem é dono edita nome, eras e se outras pessoas podem usar;
 * os outros só veem as eras.
 */
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TlButton, TlCard, TlCheckbox, TlField, TlInput, TlTextarea } from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import { deleteCalendar, getCalendar, session, updateCalendar } from '@/data/store'
import type { Calendar } from '@/data/types'
import CalendarErasEditor, { eraRow, readEraRows, type EraRow } from '@/components/CalendarErasEditor.vue'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const router = useRouter()
const calendar = ref<Calendar>()
const status = ref<'loading' | 'ok' | 'missing' | 'error'>('loading')
const loadError = ref('')
const form = reactive({ name: '', description: '', shared: false, eras: [] as EraRow[] })
const error = ref('')
const savedMsg = ref('')
const sending = ref(false)

const mine = computed(() => !!calendar.value && session.user?.handle === calendar.value.owner)

function fill(c: Calendar) {
  calendar.value = c
  Object.assign(form, { name: c.name, description: c.description, shared: c.shared, eras: c.eras.map(eraRow) })
}

watch(
  () => route.params.id,
  async (id) => {
    status.value = 'loading'
    try {
      fill(await getCalendar(String(id)))
      status.value = 'ok'
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) status.value = 'missing'
      else {
        loadError.value = errorMessage(e, 'Não deu para carregar o calendário.')
        status.value = 'error'
      }
    }
  },
  { immediate: true },
)

async function save() {
  const c = calendar.value
  if (!c || sending.value) return
  error.value = ''
  savedMsg.value = ''
  if (!form.name.trim()) return void (error.value = 'Dê um nome ao calendário.')
  const read = readEraRows(form.eras)
  if (read.error) return void (error.value = read.error)
  sending.value = true
  try {
    fill(await updateCalendar(c.id, { name: form.name.trim(), description: form.description.trim(), shared: form.shared, eras: read.eras }))
    savedMsg.value = 'Calendário salvo. Os eventos das timelines acompanham as eras.'
  } catch (e) {
    const fields = e instanceof ApiError ? Object.values(e.fields) : []
    error.value = fields[0] ?? errorMessage(e, 'Não deu para salvar o calendário.')
  } finally {
    sending.value = false
  }
}

async function remove() {
  const c = calendar.value
  if (!c || !window.confirm(`Apagar o calendário “${c.name}”?`)) return
  try {
    await deleteCalendar(c.id)
    router.push('/perfil')
  } catch (e) {
    error.value = errorMessage(e, 'Não deu para apagar o calendário.')
  }
}

const durationOf = (years?: number) => (years ? `${years.toLocaleString('pt-BR')} anos` : 'sem fim')
</script>

<template>
  <main v-if="status === 'loading'" class="tl-container" aria-busy="true"></main>
  <main v-else-if="status === 'error'" class="tl-container page"><p role="alert">{{ loadError }}</p></main>
  <NotFoundView v-else-if="status === 'missing' || !calendar" />
  <main v-else class="tl-container page">
    <TlCard v-if="mine" as="form" padding="lg" class="page__card" novalidate @submit.prevent="save">
      <h1 class="page__title">Calendário</h1>
      <TlField v-slot="{ id }" label="Nome"><TlInput :id="id" v-model="form.name" /></TlField>
      <TlField v-slot="{ id }" label="Descrição"><TlTextarea :id="id" v-model="form.description" :rows="2" /></TlField>
      <CalendarErasEditor v-model="form.eras" />
      <p class="page__note">
        Mudar a duração de uma era, ou pôr uma era nova antes de outra, move os eventos de todas as timelines deste calendário junto: T.A. 3019 continua T.A. 3019.
      </p>
      <TlCheckbox v-model="form.shared" label="Deixar outras pessoas usarem este calendário" />
      <p v-if="error" class="page__error" role="alert">{{ error }}</p>
      <p v-if="savedMsg" class="page__note" role="status">{{ savedMsg }}</p>
      <div class="page__actions">
        <TlButton variant="ghost" class="danger" @click="remove">Apagar calendário</TlButton>
        <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Salvando…' : 'Salvar' }}</TlButton>
      </div>
    </TlCard>
    <TlCard v-else padding="lg" class="page__card">
      <h1 class="page__title">{{ calendar.name }}</h1>
      <p class="page__note">Calendário de @{{ calendar.owner }}.</p>
      <p v-if="calendar.description">{{ calendar.description }}</p>
      <ol class="eras">
        <li v-for="e in calendar.eras" :key="e.id">
          <strong>{{ e.name }}</strong> ({{ e.abbr }}) · {{ durationOf(e.years) }}
        </li>
      </ol>
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
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.page__title {
  font-size: var(--tl-text-3xl);
}
.page__actions {
  display: flex;
  justify-content: space-between;
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
.danger {
  color: var(--tl-color-warning-strong);
}
.eras {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
  padding-left: var(--tl-space-5);
}
</style>
