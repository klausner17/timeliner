<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
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
import { addEvent, canEdit, deleteTimeline, findTimeline, publishTimeline, removeEvent, sortedEvents, updateEvent } from '@/data/store'
import { categories, type TimelineEvent } from '@/data/types'
import { yearLabel } from '@/lib/format'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const router = useRouter()
const timeline = computed(() => {
  const t = findTimeline(String(route.params.slug))
  return t && canEdit(t) ? t : undefined
})
const isNew = computed(() => route.query.nova === '1')

const urlPlaceholder = 'https://'
const events = computed(() => (timeline.value ? sortedEvents(timeline.value) : []))
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
  get: () => timeline.value?.tags.join(', ') ?? '',
  set: (v: string) => {
    if (timeline.value) timeline.value.tags = v.split(',').map((s) => s.trim()).filter(Boolean)
  },
})

const inviteMsg = ref('')

// ---------- Formulário de evento ----------

type Form = { year: string; endYear: string; date: string; era: string; title: string; description: string; source: string }
const blank = (): Form => ({ year: '', endYear: '', date: '', era: '', title: '', description: '', source: '' })
const form = reactive<Form>(blank())
const formOpen = ref(false)
const editing = ref<string>()
const errors = ref<Partial<Record<keyof Form, string>>>({})
const justAdded = ref<string>()

function openNew() {
  Object.assign(form, blank())
  editing.value = undefined
  errors.value = {}
  formOpen.value = true
}

function openEdit(e: TimelineEvent) {
  Object.assign(form, {
    year: String(e.year),
    endYear: e.endYear ? String(e.endYear) : '',
    date: e.date ?? '',
    era: e.era ?? '',
    title: e.title,
    description: e.description,
    source: e.sourceUrl ?? '',
  })
  editing.value = e.id
  errors.value = {}
  formOpen.value = true
}

function closeForm() {
  formOpen.value = false
  editing.value = undefined
}

function submit() {
  const t = timeline.value
  if (!t) return
  const year = Number(form.year)
  const endYear = form.endYear ? Number(form.endYear) : undefined
  const e: typeof errors.value = {}
  if (!form.year || !Number.isInteger(year)) e.year = 'Informe o ano.'
  if (endYear !== undefined && (!Number.isInteger(endYear) || endYear < year)) e.endYear = 'O fim precisa ser depois do começo.'
  if (!form.title.trim()) e.title = 'Dê um título ao evento.'
  errors.value = e
  if (Object.keys(e).length) return

  const data = {
    year,
    endYear,
    date: form.date.trim() || undefined,
    era: form.era || undefined,
    title: form.title.trim(),
    description: form.description.trim(),
    sourceUrl: form.source.trim() || undefined,
  }
  if (editing.value) {
    updateEvent(t, editing.value, data)
    justAdded.value = editing.value
  } else {
    justAdded.value = addEvent(t, data).id
  }
  closeForm()
}

function remove(e: TimelineEvent) {
  if (timeline.value && window.confirm(`Remover “${e.title}”?`)) removeEvent(timeline.value, e.id)
}

// ---------- Importar planilha (CSV: ano; título; descrição; fonte) ----------

const importInput = ref<HTMLInputElement>()
const importMsg = ref('')

async function importFile(ev: Event) {
  const t = timeline.value
  const file = (ev.target as HTMLInputElement).files?.[0]
  if (!t || !file) return
  const lines = (await file.text()).split(/\r?\n/).filter((l) => l.trim())
  let count = 0
  for (const line of lines) {
    const sep = line.includes(';') ? ';' : line.includes('\t') ? '\t' : ','
    const [y, title, description = '', source = ''] = line.split(sep).map((c) => c.trim().replace(/^"|"$/g, ''))
    const year = Number(y)
    if (!Number.isInteger(year) || !title) continue // pula cabeçalho e linhas incompletas
    addEvent(t, { year, title, description, sourceUrl: source || undefined })
    count++
  }
  importMsg.value = count ? `${count} ${count === 1 ? 'evento importado' : 'eventos importados'}.` : 'Nenhuma linha válida. Use: ano; título; descrição; fonte.'
  ;(ev.target as HTMLInputElement).value = ''
}

// ---------- Publicar e excluir ----------

function publish() {
  if (!timeline.value) return
  if (timeline.value.visibility === 'privada') timeline.value.visibility = 'publica'
  publishTimeline(timeline.value)
  router.push(`/t/${timeline.value.slug}`)
}

function destroy() {
  const t = timeline.value
  if (t && window.confirm(`Excluir a timeline “${t.title}”? Isso não pode ser desfeito.`)) {
    deleteTimeline(t.slug)
    router.push('/perfil')
  }
}
</script>

<template>
  <NotFoundView v-if="!timeline" what="timeline" />
  <div v-else class="tl-container page">
    <div class="bar">
      <div class="bar__title">
        <RouterLink to="/perfil" class="back"><TlIcon name="back" :size="14" />Minhas timelines</RouterLink>
        <div class="bar__line">
          <h1 class="bar__h1">{{ isNew && timeline.title === 'Nova timeline' ? 'Nova timeline' : timeline.title }}</h1>
          <TlBadge v-if="timeline.draft" tone="warning">Rascunho</TlBadge>
        </div>
        <p class="bar__hint">As mudanças são salvas sozinhas.</p>
      </div>
      <div class="bar__actions">
        <TlButton v-if="!timeline.draft" :to="`/t/${timeline.slug}`">Ver timeline</TlButton>
        <TlButton v-else variant="primary" :disabled="!timeline.events.length || !timeline.title.trim()" @click="publish">Publicar</TlButton>
      </div>
    </div>

    <div class="layout">
      <aside class="side">
        <TlCard as="section" class="box">
          <h2 class="box__title">Sobre a timeline</h2>
          <TlField v-slot="{ id }" label="Título"><TlInput :id="id" v-model="timeline.title" /></TlField>
          <TlField v-slot="{ id }" label="Descrição"><TlTextarea :id="id" v-model="timeline.description" :rows="3" /></TlField>
          <TlField v-slot="{ id }" label="Categoria"><TlSelect :id="id" v-model="timeline.category" :options="categoryOptions" /></TlField>
          <TlField v-slot="{ id, describedby }" label="Tags" hint="Separe por vírgula.">
            <TlInput :id="id" v-model="tagsText" :aria-describedby="describedby" />
          </TlField>
          <div class="box__group">
            <span class="box__label">Cor</span>
            <TlColorPicker v-model="timeline.color" />
          </div>
        </TlCard>

        <TlCard as="section" class="box">
          <h2 class="box__title">Quem vê e quem contribui</h2>
          <TlRadioGroup v-model="timeline.visibility" legend="Visibilidade" :options="visibilityOptions" />
          <TlRadioGroup v-model="timeline.contributions" legend="Contribuições" :options="contributionOptions" class="box__sep" />
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
            <TlButton @click="importInput?.click()">Importar planilha</TlButton>
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
            <TlField v-slot="{ id }" label="Data por extenso (opcional)">
              <TlInput :id="id" v-model="form.date" placeholder="Ex.: 13 de maio de 1888" />
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
          <div class="form__actions">
            <TlButton variant="ghost" @click="closeForm">Cancelar</TlButton>
            <TlButton type="submit" variant="primary">{{ editing ? 'Salvar evento' : 'Adicionar evento' }}</TlButton>
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
            <TlBadge v-if="e.id === justAdded" tone="info">{{ editing ? 'Editado' : 'Novo' }}</TlBadge>
            <TlButton icon-only icon="edit" variant="ghost" shape="rounded" :label="`Editar ${e.title}`" @click="openEdit(e)" />
            <TlButton icon-only icon="trash" variant="ghost" shape="rounded" :label="`Remover ${e.title}`" @click="remove(e)" />
          </div>
        </TlCard>
        <TlCard v-else-if="!formOpen" tone="dashed" class="empty">
          <p>Comece pelo primeiro evento. Você pode importar uma planilha com as colunas ano; título; descrição; fonte.</p>
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
