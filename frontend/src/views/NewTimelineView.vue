<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { TlButton, TlCard, TlCheckbox, TlField, TlInput, TlSegmented, TlSelect } from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import { createCalendar, createTimeline, listCalendars } from '@/data/store'
import { categories, type Calendar, type Category } from '@/data/types'
import CalendarErasEditor, { eraRow, readEraRows, type EraRow } from '@/components/CalendarErasEditor.vue'

// O endereço (slug) sai do título e não muda depois, por isso o título vem antes.
const router = useRouter()
const form = reactive({ title: '', category: 'História' as Category, world: 'real', calendar: '' })
const fields = ref<Record<string, string>>({})
const error = ref('')
const sending = ref(false)

// ---------- Mundo ficcional: um calendário existente ou um novo ----------

const NEW = '__novo'
const calendars = ref<Calendar[]>()
const calendarError = ref('')
const fictional = computed(() => form.world === 'ficcional')
watch(fictional, async (on) => {
  if (on && form.category === 'História') form.category = 'Ficção'
  if (!on || calendars.value) return
  try {
    calendars.value = await listCalendars()
    form.calendar = calendars.value[0]?.id ?? NEW
  } catch (e) {
    calendarError.value = errorMessage(e, 'Não deu para carregar os calendários.')
  }
})
const calendarOptions = computed(() => [
  ...(calendars.value ?? []).map((c) => ({ value: c.id, label: c.shared ? `${c.name} (de @${c.owner})` : c.name })),
  { value: NEW, label: 'Criar um calendário novo…' },
])
const chosen = computed(() => calendars.value?.find((c) => c.id === form.calendar))
const newCalendar = reactive({ name: '', shared: false, eras: [eraRow()] as EraRow[] })

async function calendarId(): Promise<string | undefined> {
  if (!fictional.value) return undefined
  if (form.calendar !== NEW) return form.calendar
  if (!newCalendar.name.trim()) throw new ApiError(400, 'invalid', 'Dê um nome ao calendário.', { calendarName: 'Dê um nome ao calendário.' })
  const read = readEraRows(newCalendar.eras)
  if (read.error) throw new ApiError(400, 'invalid', read.error, { calendarEras: read.error })
  const c = await createCalendar({ name: newCalendar.name.trim(), shared: newCalendar.shared, eras: read.eras })
  // Se a timeline falhar depois, o calendário já criado fica escolhido.
  calendars.value = [...(calendars.value ?? []), c]
  form.calendar = c.id
  return c.id
}

async function submit() {
  fields.value = {}
  error.value = ''
  if (!form.title.trim()) {
    fields.value = { title: 'Dê um título à timeline.' }
    return
  }
  sending.value = true
  try {
    const cal = await calendarId()
    const t = await createTimeline({ title: form.title.trim(), category: form.category, calendarId: cal, draft: true, visibility: 'privada', contributions: 'fechada' })
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
      <TlSegmented
        v-model="form.world"
        label="Tipo de timeline"
        :options="[
          { value: 'real', label: 'Mundo real' },
          { value: 'ficcional', label: 'Mundo ficcional' },
        ]"
      />
      <template v-if="fictional">
        <p v-if="calendarError" class="page__error" role="alert">{{ calendarError }}</p>
        <TlField v-else v-slot="{ id, describedby }" label="Calendário" hint="Como os anos são contados nesse mundo. Timelines do mesmo calendário podem ser comparadas.">
          <TlSelect :id="id" v-model="form.calendar" :options="calendarOptions" :aria-describedby="describedby" />
        </TlField>
        <p v-if="chosen" class="page__note">{{ chosen.eras.map((e) => `${e.name} (${e.abbr})`).join(' · ') }}</p>
        <div v-if="form.calendar === NEW" class="page__calendar">
          <TlField v-slot="{ id, describedby, invalid }" label="Nome do calendário" :error="fields.calendarName">
            <TlInput :id="id" v-model="newCalendar.name" placeholder="Ex.: Terra-média" :aria-describedby="describedby" :invalid="invalid" />
          </TlField>
          <CalendarErasEditor v-model="newCalendar.eras" />
          <TlCheckbox v-model="newCalendar.shared" label="Deixar outras pessoas usarem este calendário" />
        </div>
      </template>
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
  max-width: 640px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.page__calendar {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
  padding: var(--tl-space-4);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-md);
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
