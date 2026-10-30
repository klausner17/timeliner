<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import {
  TlBadge,
  TlButton,
  TlCard,
  TlColorPicker,
  TlField,
  TlIcon,
  TlInput,
  TlRadioGroup,
  TlSelect,
  TlTextarea,
} from '@/design-system'
import { errorMessage } from '@/data/api'
import { canEdit, createEvent, deleteEvent, deleteTimeline, getTimeline, loadAllEvents, updateEvent, updateTimeline } from '@/data/store'
import { categories, type EventInput, type Timeline, type TimelineEvent, type TimelinePatch } from '@/data/types'
import { chronological } from '@/data/windowing'
import { momentFields, monthOptions, readMoment, yearLabel } from '@/lib/format'
import { parseSheet } from '@/lib/importSheet'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const router = useRouter()
const slug = computed(() => String(route.params.slug))
const isNew = computed(() => route.query.nova === '1')

const status = ref<'loading' | 'ok' | 'missing' | 'error'>('loading')
const statusError = ref('')
const timeline = ref<Timeline>()
const events = ref<TimelineEvent[]>([])

// Cópia editável dos dados da timeline; cada mudança vai para a API depois de uma pausa.
type Meta = Pick<Timeline, 'title' | 'description' | 'category' | 'color' | 'visibility' | 'contributions' | 'tags'>
const meta = reactive<Meta>({ title: '', description: '', category: 'História', color: 'a', visibility: 'privada', contributions: 'fechada', tags: [] })
let saved: Meta = { ...meta }
const saveState = ref<'idle' | 'pending' | 'saving' | 'saved' | 'error'>('idle')
const saveError = ref('')
let saveTimer: ReturnType<typeof setTimeout> | undefined
let saving: Promise<void> = Promise.resolve()

watch(
  slug,
  async (s) => {
    status.value = 'loading'
    try {
      const t = await getTimeline(s)
      if (!t || !canEdit(t)) {
        status.value = 'missing'
        return
      }
      events.value = t.eventsNextCursor ? await loadAllEvents(s) : t.events
      timeline.value = t
      Object.assign(meta, pickMeta(t))
      saved = pickMeta(t)
      saveState.value = 'idle'
      status.value = 'ok'
    } catch (e) {
      statusError.value = errorMessage(e, 'Não deu para carregar a timeline.')
      status.value = 'error'
    }
  },
  { immediate: true },
)

function pickMeta(t: Meta): Meta {
  return { title: t.title, description: t.description, category: t.category, color: t.color, visibility: t.visibility, contributions: t.contributions, tags: [...t.tags] }
}

function changes(): TimelinePatch {
  const out: Record<string, unknown> = {}
  for (const k of Object.keys(meta) as (keyof Meta)[]) {
    if (JSON.stringify(meta[k]) !== JSON.stringify(saved[k])) out[k] = k === 'tags' ? [...meta.tags] : meta[k]
  }
  return out as TimelinePatch
}

watch(meta, () => {
  if (status.value !== 'ok' || !Object.keys(changes()).length) return
  saveState.value = 'pending'
  clearTimeout(saveTimer)
  saveTimer = setTimeout(flush, 800)
})

/** Manda o que mudou agora (antes de publicar ou de sair da tela). */
function flush(): Promise<void> {
  clearTimeout(saveTimer)
  saving = saving.then(async () => {
    const patch = changes()
    if (!Object.keys(patch).length || !timeline.value) return
    if (patch.title !== undefined && !patch.title.trim()) {
      saveState.value = 'error'
      saveError.value = 'O título não pode ficar vazio.'
      return
    }
    saveState.value = 'saving'
    try {
      const t = await updateTimeline(timeline.value.slug, patch)
      timeline.value = t
      saved = { ...saved, ...pickMeta({ ...saved, ...patch } as Meta) }
      saveState.value = Object.keys(changes()).length ? 'pending' : 'saved'
    } catch (e) {
      saveState.value = 'error'
      saveError.value = errorMessage(e, 'Não deu para salvar.')
    }
  })
  return saving
}

onBeforeRouteLeave(() => flush())
onBeforeUnmount(() => clearTimeout(saveTimer))

const saveLabel = computed(() => {
  switch (saveState.value) {
    case 'pending':
    case 'saving':
      return 'Salvando…'
    case 'saved':
      return 'Tudo salvo.'
    case 'error':
      return saveError.value
    default:
      return 'As mudanças são salvas sozinhas.'
  }
})

const urlPlaceholder = 'https://'
const categoryOptions = categories.map((c) => c)
const eraOptions = computed(() => [{ value: '', label: 'Sem período' }, ...(timeline.value?.eras ?? []).map((e) => ({ value: e.name, label: e.name }))])

const visibilityOptions = [
  { value: 'publica', label: 'Pública' },
  { value: 'link', label: 'Só com o link' },
  { value: 'privada', label: 'Privada' },
]
const contributionOptions = [
  { value: 'sugestoes', label: 'Qualquer pessoa sugere, eu aprovo' },
  { value: 'convidados', label: 'Só pessoas convidadas editam' },
  { value: 'fechada', label: 'Fechada, só eu edito' },
]

const tagsText = computed({
  get: () => meta.tags.join(', '),
  set: (v: string) => {
    meta.tags = v.split(',').map((s) => s.trim()).filter(Boolean)
  },
})

const inviteMsg = ref('')

// ---------- Formulário de evento ----------

type Form = { year: string; month: string; day: string; time: string; endYear: string; date: string; era: string; title: string; description: string; source: string }
const blank = (): Form => ({ year: '', month: '', day: '', time: '', endYear: '', date: '', era: '', title: '', description: '', source: '' })
const form = reactive<Form>(blank())
const formOpen = ref(false)
const editing = ref<string>()
const errors = ref<Partial<Record<keyof Form, string>>>({})
const formError = ref('')
const sending = ref(false)
const justAdded = ref<string>()
const justEdited = ref(false)

function openNew() {
  Object.assign(form, blank())
  editing.value = undefined
  errors.value = {}
  formError.value = ''
  formOpen.value = true
}

function openEdit(e: TimelineEvent) {
  Object.assign(form, {
    year: String(e.year),
    ...momentFields(e),
    endYear: e.endYear ? String(e.endYear) : '',
    date: e.date ?? '',
    era: e.era ?? '',
    title: e.title,
    description: e.description,
    source: e.sourceUrl ?? '',
  })
  editing.value = e.id
  errors.value = {}
  formError.value = ''
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = undefined
}

function place(e: TimelineEvent) {
  events.value = chronological([...events.value.filter((x) => x.id !== e.id), e])
}

async function submit() {
  const t = timeline.value
  if (!t || sending.value) return
  const year = Number(form.year)
  const endYear = form.endYear ? Number(form.endYear) : undefined
  const moment = readMoment(year, form)
  const e: typeof errors.value = { ...moment.errors }
  if (!form.year || !Number.isInteger(year)) e.year = 'Informe o ano.'
  if (endYear !== undefined && (!Number.isInteger(endYear) || endYear < year)) e.endYear = 'O fim precisa ser depois do começo.'
  if (!form.title.trim()) e.title = 'Dê um título ao evento.'
  errors.value = e
  formError.value = ''
  if (Object.keys(e).length) return

  const data: EventInput = {
    year,
    ...moment.value,
    endYear,
    date: form.date.trim() || undefined,
    era: form.era || undefined,
    title: form.title.trim(),
    description: form.description.trim(),
    sourceUrl: form.source.trim() || undefined,
  }
  sending.value = true
  try {
    // Na edição, campo apagado vai como null para a API limpar.
    const result = editing.value
      ? await updateEvent(t.slug, editing.value, { ...data, month: data.month ?? null, day: data.day ?? null, time: data.time ?? null, endYear: data.endYear ?? null, date: data.date ?? null, era: data.era ?? null, sourceUrl: data.sourceUrl ?? null })
      : await createEvent(t.slug, data)
    justEdited.value = !!editing.value
    justAdded.value = result.id
    place(result)
    closeForm()
  } catch (err) {
    formError.value = errorMessage(err, 'Não deu para salvar o evento.')
  } finally {
    sending.value = false
  }
}

async function remove(e: TimelineEvent) {
  if (!timeline.value || !window.confirm(`Remover “${e.title}”?`)) return
  try {
    await deleteEvent(timeline.value.slug, e.id)
    events.value = events.value.filter((x) => x.id !== e.id)
  } catch (err) {
    window.alert(errorMessage(err, 'Não deu para remover o evento.'))
  }
}

// ---------- Importar planilha (CSV: ano ou data; título; descrição; fonte) ----------

const importInput = ref<HTMLInputElement>()
const importMsg = ref('')
const importing = ref(false)

async function importFile(ev: Event) {
  const t = timeline.value
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!t || !file) return
  const rows = parseSheet(await file.text())
  if (!rows.length) {
    importMsg.value = 'Nenhuma linha válida. Use: ano ou data; título; descrição; fonte.'
    return
  }
  importing.value = true
  let done = 0
  try {
    for (const row of rows) {
      importMsg.value = `Importando ${done + 1} de ${rows.length}…`
      place(await createEvent(t.slug, row))
      done++
    }
    importMsg.value = `${done} ${done === 1 ? 'evento importado' : 'eventos importados'}.`
  } catch (err) {
    importMsg.value = `${done} de ${rows.length} importados. Parou em: ${errorMessage(err)}`
  } finally {
    importing.value = false
  }
}

// ---------- Publicar e excluir ----------

const publishing = ref(false)

async function publish() {
  const t = timeline.value
  if (!t) return
  publishing.value = true
  try {
    await flush()
    if (saveState.value === 'error') return
    if (meta.visibility === 'privada') {
      meta.visibility = 'publica'
      saved.visibility = 'publica'
    }
    timeline.value = await updateTimeline(t.slug, { draft: false, visibility: meta.visibility })
    router.push(`/t/${t.slug}`)
  } catch (err) {
    saveState.value = 'error'
    saveError.value = errorMessage(err, 'Não deu para publicar.')
  } finally {
    publishing.value = false
  }
}

async function destroy() {
  const t = timeline.value
  if (!t || !window.confirm(`Excluir a timeline “${t.title}”? Isso não pode ser desfeito.`)) return
  try {
    clearTimeout(saveTimer)
    await deleteTimeline(t.slug)
    status.value = 'missing'
    router.push('/perfil')
  } catch (err) {
    window.alert(errorMessage(err, 'Não deu para excluir.'))
  }
}
</script>


<template>
  <main v-if="status === 'loading'" class="tl-container" aria-busy="true"></main>
  <main v-else-if="status === 'error'" class="tl-container page">
    <p class="main__msg" role="alert">{{ statusError }}</p>
  </main>
  <NotFoundView v-else-if="status === 'missing' || !timeline" what="timeline" />
  <div v-else class="tl-container page">
    <div class="bar">
      <div class="bar__title">
        <RouterLink to="/perfil" class="back"><TlIcon name="back" :size="14" />Minhas timelines</RouterLink>
        <div class="bar__line">
          <h1 class="bar__h1">{{ meta.title.trim() || (isNew ? 'Nova timeline' : timeline.title) }}</h1>
          <TlBadge v-if="timeline.draft" tone="warning">Rascunho</TlBadge>
        </div>
        <p class="bar__hint" :class="{ 'bar__hint--error': saveState === 'error' }" role="status">{{ saveLabel }}</p>
      </div>
      <div class="bar__actions">
        <TlButton v-if="!timeline.draft" :to="`/t/${timeline.slug}`">Ver timeline</TlButton>
        <TlButton v-else variant="primary" :disabled="publishing || !events.length || !meta.title.trim()" @click="publish">{{ publishing ? 'Publicando…' : 'Publicar' }}</TlButton>
      </div>
    </div>

    <div class="layout">
      <aside class="side">
        <TlCard as="section" class="box">
          <h2 class="box__title">Sobre a timeline</h2>
          <TlField v-slot="{ id }" label="Título"><TlInput :id="id" v-model="meta.title" /></TlField>
          <TlField v-slot="{ id }" label="Descrição"><TlTextarea :id="id" v-model="meta.description" :rows="3" /></TlField>
          <TlField v-slot="{ id }" label="Categoria"><TlSelect :id="id" v-model="meta.category" :options="categoryOptions" /></TlField>
          <TlField v-slot="{ id, describedby }" label="Tags" hint="Separe por vírgula.">
            <TlInput :id="id" v-model="tagsText" :aria-describedby="describedby" />
          </TlField>
          <div class="box__group">
            <span class="box__label">Cor</span>
            <TlColorPicker v-model="meta.color" />
          </div>
        </TlCard>

        <TlCard as="section" class="box">
          <h2 class="box__title">Quem vê e quem contribui</h2>
          <TlRadioGroup v-model="meta.visibility" legend="Visibilidade" :options="visibilityOptions" />
          <TlRadioGroup v-model="meta.contributions" legend="Contribuições" :options="contributionOptions" class="box__sep" />
          <TlButton
            v-if="timeline.contributions === 'convidados'"
            shape="rounded"
            block
            @click="inviteMsg = 'Os convites chegam junto com as contas de usuário.'"
          >
            Convidar editores
          </TlButton>
          <p v-if="inviteMsg" class="box__hint" role="status">{{ inviteMsg }}</p>
        </TlCard>

        <TlButton variant="ghost" class="danger" @click="destroy">Excluir timeline</TlButton>
      </aside>

      <main class="main">
        <div class="main__head">
          <h2 class="main__title">Eventos <span class="main__count">({{ events.length }})</span></h2>
          <div class="main__actions">
            <TlButton :disabled="importing" @click="importInput?.click()">Importar planilha</TlButton>
            <input ref="importInput" type="file" accept=".csv,.tsv,.txt" class="tl-visually-hidden" tabindex="-1" aria-hidden="true" @change="importFile" />
            <TlButton variant="dark" icon="plus" @click="openNew">Novo evento</TlButton>
          </div>
        </div>
        <p v-if="importMsg" class="main__msg" role="status">{{ importMsg }}</p>

        <TlCard v-if="formOpen" as="form" highlight class="form" @submit.prevent="submit">
          <h3 class="form__title">{{ editing ? 'Editar evento' : 'Novo evento' }}</h3>
          <div class="form__grid">
            <TlField v-slot="{ id, describedby, invalid }" label="Ano" :error="errors.year">
              <TlInput :id="id" v-model="form.year" inputmode="numeric" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Até (opcional)" :error="errors.endYear">
              <TlInput :id="id" v-model="form.endYear" inputmode="numeric" placeholder="Ano de fim" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Mês (opcional)" :error="errors.month">
              <TlSelect :id="id" v-model="form.month" :options="monthOptions" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Dia (opcional)" :error="errors.day">
              <TlInput :id="id" v-model="form.day" inputmode="numeric" maxlength="2" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Hora (opcional)" :error="errors.time">
              <TlInput :id="id" v-model="form.time" type="time" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
            <TlField v-slot="{ id, describedby }" label="Data por extenso (opcional)" hint="Substitui a data montada com mês, dia e hora.">
              <TlInput :id="id" v-model="form.date" placeholder="Ex.: verão de 1888" :aria-describedby="describedby" />
            </TlField>
            <TlField v-if="timeline.eras.length" v-slot="{ id }" label="Período">
              <TlSelect :id="id" v-model="form.era" :options="eraOptions" />
            </TlField>
          </div>
          <TlField v-slot="{ id, describedby, invalid }" label="Título" :error="errors.title">
            <TlInput :id="id" v-model="form.title" :aria-describedby="describedby" :invalid="invalid" />
          </TlField>
          <TlField v-slot="{ id }" label="Descrição"><TlTextarea :id="id" v-model="form.description" :rows="2" /></TlField>
          <TlField v-slot="{ id }" label="Fonte"><TlInput :id="id" v-model="form.source" type="url" :placeholder="urlPlaceholder" /></TlField>
          <p v-if="formError" class="form__error" role="alert">{{ formError }}</p>
          <div class="form__actions">
            <TlButton variant="ghost" @click="closeForm">Cancelar</TlButton>
            <TlButton type="submit" variant="primary" :disabled="sending">{{ editing ? 'Salvar evento' : 'Adicionar evento' }}</TlButton>
          </div>
        </TlCard>

        <TlCard v-if="events.length" as="section" padding="sm" class="list" aria-label="Lista de eventos">
          <div v-for="e in events" :key="e.id" class="row" :class="{ 'row--new': e.id === justAdded }">
            <span class="row__dot" :style="{ background: `var(--tl-track-${timeline.color})` }" aria-hidden="true"></span>
            <span class="row__year">{{ yearLabel(e) }}</span>
            <div class="row__text">
              <span class="row__title">{{ e.title }}</span>
              <span class="row__desc">{{ e.description }}</span>
            </div>
            <TlBadge v-if="e.id === justAdded" tone="info">{{ justEdited ? 'Editado' : 'Novo' }}</TlBadge>
            <TlButton icon-only icon="edit" variant="ghost" shape="rounded" :label="`Editar ${e.title}`" @click="openEdit(e)" />
            <TlButton icon-only icon="trash" variant="ghost" shape="rounded" :label="`Remover ${e.title}`" @click="remove(e)" />
          </div>
        </TlCard>
        <TlCard v-else-if="!formOpen" tone="dashed" class="empty">
          <p>Comece pelo primeiro evento. Você pode importar uma planilha com as colunas ano ou data; título; descrição; fonte. A data pode ser 13/05/1888, 05/1888 ou 1888-05-13, com hora opcional, como 13/05/1888 14:30.</p>
          <TlButton variant="dark" icon="plus" @click="openNew">Novo evento</TlButton>
        </TlCard>
        <p class="main__note">Os eventos são ordenados pela data automaticamente.</p>
      </main>
    </div>
  </div>
</template>

<style scoped>
.page {
  padding-top: var(--tl-space-7);
  padding-bottom: calc(var(--tl-space-8) * 2);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-6);
}
.bar {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  justify-content: space-between;
  gap: var(--tl-space-4);
}
.bar__title {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
}
.back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: fit-content;
  min-height: var(--tl-control-md);
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
}
.bar__line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-3);
}
.bar__h1 {
  font-size: var(--tl-text-4xl);
  letter-spacing: var(--tl-tracking-tight);
}
.bar__hint {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.bar__hint--error,
.form__error {
  color: var(--tl-color-warning-strong);
}
.form__error {
  font-size: var(--tl-text-md);
}
.bar__actions {
  display: flex;
  gap: var(--tl-space-2);
}
.layout {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-7);
  align-items: start;
}
.side {
  flex: 1 1 320px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.box {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.box__title {
  font-family: var(--tl-font-body);
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  letter-spacing: 0;
}
.box__group {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.box__label {
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
}
.box__sep {
  padding-top: var(--tl-space-3);
  border-top: 1px solid var(--tl-color-border-subtle);
}
.box__hint {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.danger {
  align-self: start;
  color: var(--tl-color-warning-strong);
}
.main {
  flex: 999 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.main__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-3);
}
.main__title {
  font-size: var(--tl-text-3xl);
}
.main__count {
  font-weight: var(--tl-weight-medium);
  color: var(--tl-color-text-muted);
}
.main__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
}
.main__msg,
.main__note {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.form__title {
  font-family: var(--tl-font-body);
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  letter-spacing: 0;
}
.form__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--tl-space-3);
}
.form__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: end;
  gap: var(--tl-space-2);
}
.list {
  padding: 0;
  overflow: hidden;
}
.row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: var(--tl-space-2) var(--tl-space-3) var(--tl-space-2) var(--tl-space-4);
  border-bottom: 1px solid var(--tl-color-border-subtle);
}
.row:last-child {
  border-bottom: 0;
}
.row--new {
  background: var(--tl-color-primary-soft);
}
.row__dot {
  flex: none;
  width: 12px;
  height: 12px;
  border-radius: var(--tl-radius-pill);
}
.row__year {
  flex: none;
  width: 64px;
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-semibold);
  font-size: 16px;
}
.row__text {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.row__title {
  font-weight: var(--tl-weight-semibold);
}
.row__desc {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.empty {
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: var(--tl-space-4);
}
</style>
