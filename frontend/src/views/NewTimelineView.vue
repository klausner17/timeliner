<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { TlButton, TlCard, TlCheckbox, TlField, TlInput, TlSegmented, TlSelect } from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import { createCalendar, createTimeline, currentUser, listCalendars } from '@/data/store'
import { categories, type Calendar, type Category, type Scale } from '@/data/types'
import { scaleOptions } from '@/lib/scale'
import CalendarErasEditor, { eraRow, readEraRows, type EraRow } from '@/components/CalendarErasEditor.vue'

// O endereço (slug) sai do título e não muda depois, por isso o título vem antes.
const router = useRouter()
// Só conta com e-mail confirmado cria conteúdo (o servidor também confere).
const unverified = computed(() => currentUser.value != null && !currentUser.value.emailVerified)
const form = reactive({ title: '', category: 'História' as Category, world: 'real', calendar: '', scale: 'dias' as Scale, cyclic: false })
const fields = ref<Record<string, string>>({})
const error = ref('')
const sending = ref(false)

// ---------- Fixa: sem calendário, contada do início (ciclo de vida do lagarto...) ----------

const fixed = computed(() => form.world === 'fixa')
watch(fixed, (on) => {
  if (on && form.category === 'História') form.category = 'Natureza'
})

// ---------- Mundo ficcional: um calendário existente ou um novo ----------

const NEW = '__novo'
// Os calendários compartilhados crescem com o site: a lista vem do servidor, 50 por vez,
// e a busca pelo nome acha os que não couberam.
const CALENDAR_PAGE = 50
const calendars = ref<Calendar[]>()
const calendarTotal = ref(0)
const calendarError = ref('')
const calendarSearch = ref('')
/** O calendário escolhido continua nas opções mesmo se a busca não o trouxer. */
const picked = ref<Calendar>()
const fictional = computed(() => form.world === 'ficcional')
let calendarSeq = 0
async function loadCalendars(q: string) {
  const id = ++calendarSeq
  try {
    const page = await listCalendars({ q: q || undefined, pageSize: CALENDAR_PAGE })
    if (id !== calendarSeq) return
    calendars.value = page.items
    calendarTotal.value = page.total
    calendarError.value = ''
    if (!form.calendar) form.calendar = page.items[0]?.id ?? NEW
  } catch (e) {
    if (id === calendarSeq) calendarError.value = errorMessage(e, 'Não deu para carregar os calendários.')
  }
}
watch(fictional, (on) => {
  if (on && form.category === 'História') form.category = 'Ficção'
  if (on && !calendars.value) void loadCalendars('')
})
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(calendarSearch, (q) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => loadCalendars(q.trim()), 250)
})
onBeforeUnmount(() => clearTimeout(searchTimer))
const calendarLabel = (c: Calendar) => (c.shared ? `${c.name} (de @${c.owner})` : c.name)
const calendarOptions = computed(() => {
  const list = calendars.value ?? []
  const extra = picked.value && !list.some((c) => c.id === picked.value!.id) ? [picked.value] : []
  return [...extra, ...list].map((c) => ({ value: c.id, label: calendarLabel(c) })).concat({ value: NEW, label: 'Criar um calendário novo…' })
})
const chosen = computed(() => calendars.value?.find((c) => c.id === form.calendar) ?? (picked.value?.id === form.calendar ? picked.value : undefined))
watch(chosen, (c) => {
  if (c) picked.value = c
})
const moreCalendars = computed(() => calendarTotal.value - (calendars.value?.length ?? 0))
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
    const kind = fixed.value ? { scale: form.scale, cyclic: form.cyclic } : { calendarId: cal }
    const t = await createTimeline({ title: form.title.trim(), category: form.category, ...kind, draft: true, visibility: 'privada', contributions: 'fechada' })
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
    <TlCard v-if="unverified" padding="lg" class="page__card">
      <h1 class="page__title">Confirme seu e-mail</h1>
      <p>
        Para criar timelines e sugerir eventos, abra o link que mandamos para <strong>{{ currentUser?.email }}</strong>.
        Se ele não chegou, confira o spam ou use "Mandar de novo" no aviso do topo da página.
      </p>
      <div class="page__actions">
        <TlButton to="/" variant="ghost">Voltar ao início</TlButton>
      </div>
    </TlCard>
    <TlCard v-else as="form" padding="lg" class="page__card" novalidate @submit.prevent="submit">
      <h1 class="page__title">Nova timeline</h1>
      <TlField v-slot="{ id, describedby, invalid }" label="Título" :error="fields.title" hint="Também vira o endereço da timeline.">
        <TlInput :id="id" v-model="form.title" :aria-describedby="describedby" :invalid="invalid" />
      </TlField>
      <TlSegmented
        v-model="form.world"
        label="Tipo de timeline"
        :options="[
          { value: 'real', label: 'Real' },
          { value: 'ficcional', label: 'Ficcional' },
          { value: 'fixa', label: 'Fixa' },
        ]"
      />
      <template v-if="fixed">
        <p class="page__note">
          Sem datas no calendário: cada evento fica a tantos dias, semanas, meses ou anos do início, como no ciclo de vida do lagarto (ovo no dia 0,
          eclosão no dia 60...).
        </p>
        <TlField v-slot="{ id, describedby }" label="Contar em" hint="Timelines fixas na mesma unidade podem ser comparadas. Não muda depois.">
          <TlSelect :id="id" v-model="form.scale" :options="scaleOptions" :aria-describedby="describedby" />
        </TlField>
        <TlCheckbox v-model="form.cyclic" label="Cíclica: depois do fim, recomeça do início" />
      </template>
      <template v-if="fictional">
        <p v-if="calendarError" class="page__error" role="alert">{{ calendarError }}</p>
        <template v-else>
          <TlField v-if="calendarTotal > CALENDAR_PAGE || calendarSearch" v-slot="{ id }" label="Buscar calendário">
            <TlInput :id="id" v-model="calendarSearch" type="search" placeholder="Pelo nome: Terra-média, Westeros…" />
          </TlField>
          <TlField v-slot="{ id, describedby }" label="Calendário" hint="Como os anos são contados nesse mundo. Timelines do mesmo calendário podem ser comparadas.">
            <TlSelect :id="id" v-model="form.calendar" :options="calendarOptions" :aria-describedby="describedby" />
          </TlField>
          <p v-if="moreCalendars > 0" class="page__note">Mostrando {{ calendars?.length }} de {{ calendarTotal }}. Busque pelo nome para achar os outros.</p>
        </template>
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
  flex-wrap: wrap;
  justify-content: end;
  gap: var(--tl-space-2);
}
.page__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
@media (max-width: 480px) {
  .page {
    padding-top: var(--tl-space-6);
    padding-bottom: var(--tl-space-8);
  }
  /* Botão principal em cima, ocupando a largura; Cancelar embaixo. */
  .page__actions {
    flex-direction: column-reverse;
    align-items: stretch;
  }
}
.page__note {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
