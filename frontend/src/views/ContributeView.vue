<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  TlAvatar,
  TlBadge,
  TlButton,
  TlCard,
  TlCheckbox,
  TlEventItem,
  TlField,
  TlIcon,
  TlInput,
  TlSelect,
  TlTextarea,
  type BadgeTone,
} from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import {
  canEdit,
  currentUser,
  getTimeline,
  handleLabel,
  listContributions,
  listSuggestions,
  reopenSuggestion,
  reviewSuggestion,
  suggest,
} from '@/data/store'
import { suggestionKindLabel, type EventInput, type Suggestion, type SuggestionStatus, type Timeline } from '@/data/types'
import { dateLabel, monthOptions, readMoment, timeAgo, yearLabel } from '@/lib/format'
import CalendarYearInput from '@/components/CalendarYearInput.vue'
import { reckoningOf, unitLabel } from '@/lib/scale'
import NotFoundView from './NotFoundView.vue'

const route = useRoute()
const status = ref<'loading' | 'ok' | 'missing' | 'error'>('loading')
const statusError = ref('')
const timeline = ref<Timeline>()
const isOwner = computed(() => !!timeline.value && canEdit(timeline.value))
const open = computed(() => timeline.value?.contributions === 'sugestoes')
const loginPath = computed(() => `/entrar?next=${encodeURIComponent(route.fullPath)}`)

/** Dono: a fila de sugestões. Outros: as próprias sugestões nesta timeline. */
const suggestions = ref<Suggestion[]>([])

watch(
  [() => String(route.params.slug), () => currentUser.value?.handle],
  async ([slug, me]) => {
    status.value = 'loading'
    try {
      const t = await getTimeline(slug)
      if (!t) {
        status.value = 'missing'
        return
      }
      timeline.value = t
      if (canEdit(t)) suggestions.value = await listSuggestions(slug)
      else suggestions.value = me ? (await listContributions(me)).filter((s) => s.timeline.slug === slug) : []
      status.value = 'ok'
    } catch (e) {
      statusError.value = errorMessage(e, 'Não deu para carregar as sugestões.')
      status.value = 'error'
    }
  },
  { immediate: true },
)

// ---------- Formulário de sugestão ----------

const urlPlaceholder = 'https://'
const MAX_DESC = 280

const blank = () => ({
  day: '',
  month: '',
  time: '',
  year: '',
  approximate: false,
  period: false,
  endYear: '',
  title: '',
  description: '',
  era: '',
  source: '',
  note: '',
})
const form = reactive(blank())
const errors = ref<Record<string, string>>({})
const sendError = ref('')
const sending = ref(false)
const sent = ref<Suggestion>()

/**
 * Calendário ficcional (anos por era) ou escala de uma timeline fixa (dias desde o início):
 * só o ano, como numa data aproximada.
 */
const cal = computed(() => reckoningOf(timeline.value))
const scale = computed(() => timeline.value?.scale)
const posLabel = computed(() => (scale.value ? unitLabel(scale.value) : 'Ano'))
const yearOnly = computed(() => form.approximate || !!cal.value)

const eraOptions = computed(() => [{ value: '', label: 'Escolher pelo ano' }, ...(timeline.value?.eras ?? []).map((e) => ({ value: e.name, label: e.name }))])

function eraFor(year: number) {
  const eras = timeline.value?.eras ?? []
  return [...eras].reverse().find((e) => year >= e.start)?.name
}

const draft = computed<EventInput | undefined>(() => {
  const year = Number(form.year)
  if (!form.year || !Number.isInteger(year)) return undefined
  const endYear = form.period && form.endYear ? Number(form.endYear) : undefined
  const moment = readMoment(year, form)
  return {
    year,
    ...(yearOnly.value || Object.keys(moment.errors).length ? {} : moment.value),
    endYear: endYear && Number.isInteger(endYear) ? endYear : undefined,
    title: form.title.trim(),
    description: form.description.trim(),
    sourceUrl: form.source.trim() || undefined,
    era: form.era || eraFor(year),
  }
})

function validate() {
  const e: Record<string, string> = {}
  const year = Number(form.year)
  if (!form.year) e.year = scale.value ? `Informe o ${posLabel.value.toLowerCase()}.` : 'Informe o ano.'
  else if (scale.value) {
    if (!Number.isInteger(year) || year < 0) e.year = 'Use um número inteiro a partir do 0.'
  } else if (cal.value) {
    if (!Number.isInteger(year)) e.year = 'Use um ano que exista na era.'
  } else if (!Number.isInteger(year) || year < -10000 || year > 2100) e.year = 'Use um ano válido, como 1992.'
  if (!yearOnly.value) Object.assign(e, readMoment(year, form).errors)
  if (form.period) {
    const end = Number(form.endYear)
    if (!form.endYear) e.endYear = 'Informe o ano de fim.'
    else if (!Number.isInteger(end) || end < year) e.endYear = 'O fim precisa ser depois do começo.'
  }
  if (!form.title.trim()) e.title = 'Dê um título ao evento.'
  if (!form.description.trim()) e.description = 'Conte o essencial em poucas linhas.'
  try {
    const u = new URL(form.source.trim())
    if (!/^https?:$/.test(u.protocol)) throw new Error()
  } catch {
    e.source = 'Informe um link para a fonte, começando com https://'
  }
  errors.value = e
  return !Object.keys(e).length
}

function focusFirstError() {
  const first = Object.keys(errors.value)[0]
  if (first) document.querySelector<HTMLElement>(`[data-field="${first}"] input, [data-field="${first}"] textarea`)?.focus()
}

async function submit() {
  sendError.value = ''
  if (!timeline.value || !validate() || !draft.value) {
    focusFirstError()
    return
  }
  sending.value = true
  try {
    const s = await suggest(timeline.value.slug, { kind: 'new_event', event: draft.value, note: form.note.trim() || undefined })
    suggestions.value = [s, ...suggestions.value]
    sent.value = s
    Object.assign(form, blank())
  } catch (e) {
    sendError.value = errorMessage(e, 'Não deu para enviar a sugestão.')
    if (e instanceof ApiError && Object.keys(e.fields).length) {
      errors.value = e.fields
      focusFirstError()
    }
  } finally {
    sending.value = false
  }
}

// ---------- Revisão (dono) ----------

const pendingCount = computed(() => suggestions.value.filter((s) => s.status === 'pendente').length)

const adjusting = ref<string>()
const feedback = ref('')
const busy = ref<string>()
const reviewError = ref('')

/** Troca a sugestão na lista pela versão que a API devolveu. */
async function act(id: string, call: () => Promise<Suggestion>) {
  busy.value = id
  reviewError.value = ''
  try {
    const s = await call()
    suggestions.value = suggestions.value.map((x) => (x.id === id ? s : x))
    return true
  } catch (e) {
    reviewError.value = errorMessage(e, 'Não deu para registrar a decisão.')
    return false
  } finally {
    busy.value = undefined
  }
}

const decide = (id: string, decision: 'approve' | 'reject') => act(id, () => reviewSuggestion(id, decision))
const undo = (id: string) => act(id, () => reopenSuggestion(id))

async function askAdjust(id: string) {
  if (await act(id, () => reviewSuggestion(id, 'request_changes', feedback.value.trim() || undefined))) {
    adjusting.value = undefined
    feedback.value = ''
  }
}

const statusBadge: Record<SuggestionStatus, { label: string; tone: BadgeTone }> = {
  pendente: { label: 'Pendente', tone: 'info' },
  aprovada: { label: 'Aprovada', tone: 'success' },
  recusada: { label: 'Recusada', tone: 'neutral' },
  ajuste: { label: 'Ajuste pedido', tone: 'warning' },
}
</script>

<template>
  <main v-if="status === 'loading'" class="tl-container" aria-busy="true"></main>
  <main v-else-if="status === 'error'" class="tl-container page">
    <p class="page__lead" role="alert">{{ statusError }}</p>
  </main>
  <NotFoundView v-else-if="status === 'missing' || !timeline" what="timeline" />
  <main v-else class="tl-container page">
    <div class="page__head">
      <RouterLink :to="`/t/${timeline.slug}`" class="back"><TlIcon name="back" :size="14" />{{ timeline.title }}</RouterLink>
      <template v-if="isOwner">
        <h1 class="page__title">Revisar sugestões</h1>
        <p class="page__lead">Aprovadas, as sugestões entram na timeline com o nome de quem enviou. Você pode desfazer uma decisão.</p>
      </template>
      <template v-else>
        <h1 class="page__title">Sugerir um evento</h1>
        <p class="page__lead">
          Sua sugestão vai para o dono da timeline, que pode aprovar, pedir ajustes ou recusar. Eventos com fonte confiável são aprovados mais
          rápido.
        </p>
      </template>
    </div>

    <!-- Quem não é dono: formulário de sugestão -->
    <template v-if="!isOwner">
      <TlCard v-if="!open" tone="muted" padding="lg" class="closed">
        <p>Esta timeline não aceita sugestões no momento. Só {{ timeline.contributions === 'convidados' ? 'pessoas convidadas' : 'o dono' }} pode editar.</p>
        <TlButton :to="`/t/${timeline.slug}`">Voltar para a timeline</TlButton>
      </TlCard>

      <TlCard v-else-if="!currentUser" tone="muted" padding="lg" class="closed">
        <p>Entre na sua conta para sugerir eventos.</p>
        <TlButton variant="primary" :to="loginPath">Entrar</TlButton>
      </TlCard>

      <div v-else class="split">
        <TlCard v-if="sent" as="section" padding="lg" class="form sent" role="status">
          <h2 class="sent__title">Sugestão enviada</h2>
          <p>“{{ sent.event.title }}” foi para {{ handleLabel(timeline.owner.handle) }}. Você acompanha a resposta no seu perfil, em Contribuições.</p>
          <div class="sent__actions">
            <TlButton variant="primary" @click="sent = undefined">Sugerir outro evento</TlButton>
            <TlButton :to="`/t/${timeline.slug}`">Voltar para a timeline</TlButton>
          </div>
        </TlCard>

        <TlCard v-else as="form" padding="lg" class="form" novalidate @submit.prevent="submit">
          <fieldset class="when">
            <legend class="when__legend">Quando aconteceu</legend>
            <TlField v-if="cal" v-slot="{ id, describedby, invalid }" :label="posLabel" tone="muted" :error="errors.year" data-field="year">
              <CalendarYearInput :id="id" v-model="form.year" :calendar="cal" :describedby="describedby" :invalid="invalid" />
            </TlField>
            <div v-else class="when__grid" :class="{ 'when__grid--year': form.approximate }">
              <template v-if="!form.approximate">
                <TlField v-slot="{ id, describedby, invalid }" label="Dia" tone="muted" :error="errors.day" data-field="day">
                  <TlInput :id="id" v-model="form.day" inputmode="numeric" maxlength="2" :aria-describedby="describedby" :invalid="invalid" />
                </TlField>
                <TlField v-slot="{ id }" label="Mês" tone="muted">
                  <TlSelect :id="id" v-model="form.month" :options="monthOptions" />
                </TlField>
              </template>
              <TlField v-slot="{ id, describedby, invalid }" label="Ano" tone="muted" :error="errors.year" data-field="year">
                <TlInput :id="id" v-model="form.year" inputmode="numeric" :aria-describedby="describedby" :invalid="invalid" />
              </TlField>
              <TlField v-if="!form.approximate" v-slot="{ id, describedby, invalid }" label="Hora" tone="muted" :error="errors.time" data-field="time">
                <TlInput :id="id" v-model="form.time" type="time" :aria-describedby="describedby" :invalid="invalid" />
              </TlField>
            </div>
            <TlCheckbox v-if="!cal" v-model="form.approximate" label="Data aproximada (mostrar só o ano)" />
            <TlCheckbox v-model="form.period" label="É um período (tem data de fim)" />
            <TlField v-if="form.period" v-slot="{ id, describedby, invalid }" :label="scale ? 'Fim' : 'Ano de fim'" tone="muted" :error="errors.endYear" data-field="endYear" class="when__end">
              <CalendarYearInput v-if="cal" :id="id" v-model="form.endYear" :calendar="cal" :describedby="describedby" :invalid="invalid" />
              <TlInput v-else :id="id" v-model="form.endYear" inputmode="numeric" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
          </fieldset>

          <TlField v-slot="{ id, describedby, invalid }" label="Título" :error="errors.title" data-field="title">
            <TlInput :id="id" v-model="form.title" :aria-describedby="describedby" :invalid="invalid" />
          </TlField>

          <TlField v-slot="{ id, describedby, invalid }" label="Descrição" :error="errors.description" data-field="description">
            <TlTextarea :id="id" v-model="form.description" :maxlength="MAX_DESC" :aria-describedby="describedby" :invalid="invalid" />
          </TlField>

          <div class="form__row">
            <TlField v-if="timeline.eras.length" v-slot="{ id }" label="Período da timeline">
              <TlSelect :id="id" v-model="form.era" :options="eraOptions" />
            </TlField>
            <TlField v-slot="{ id, describedby, invalid }" label="Fonte (obrigatória)" :error="errors.source" data-field="source">
              <TlInput :id="id" v-model="form.source" type="url" :placeholder="urlPlaceholder" :aria-describedby="describedby" :invalid="invalid" />
            </TlField>
          </div>


          <TlField v-slot="{ id }" label="Recado para o dono (opcional)">
            <TlInput :id="id" v-model="form.note" placeholder="Por que esse evento deveria entrar?" />
          </TlField>

          <p v-if="sendError" class="form__error" role="alert">{{ sendError }}</p>
          <div class="form__actions">
            <TlButton :to="`/t/${timeline.slug}`" variant="ghost">Cancelar</TlButton>
            <TlButton type="submit" variant="primary" :disabled="sending">{{ sending ? 'Enviando…' : 'Enviar sugestão' }}</TlButton>
          </div>
        </TlCard>

        <aside class="aside">
          <section v-if="!sent" class="preview">
            <h2 class="preview__title">Prévia na timeline</h2>
            <TlEventItem
              :year="draft ? yearLabel(draft, cal) : '—'"
              :date="draft ? dateLabel(draft, cal) : undefined"
              :title="form.title || 'Título do evento'"
              :description="form.description || 'A descrição aparece aqui.'"
              :color="timeline.color"
              suggestion
              compact
            />
          </section>
          <TlCard class="how">
            <h2 class="how__title">Como funciona</h2>
            <ol class="how__list">
              <li><span aria-hidden="true">1</span>Você envia o evento com uma fonte.</li>
              <li><span aria-hidden="true">2</span>O dono revisa e pode pedir ajustes.</li>
              <li><span aria-hidden="true">3</span>Aprovado, o evento entra com seu nome como contribuidor.</li>
            </ol>
          </TlCard>
          <TlCard v-if="suggestions.length" class="how">
            <h2 class="how__title">Suas sugestões aqui</h2>
            <ul class="mine">
              <li v-for="s in suggestions" :key="s.id">
                <span>{{ s.event.title }}</span>
                <TlBadge :tone="statusBadge[s.status].tone">{{ statusBadge[s.status].label }}</TlBadge>
              </li>
            </ul>
          </TlCard>
        </aside>
      </div>
    </template>

    <!-- Dono: fila de sugestões -->
    <section v-else id="revisar" class="review" aria-labelledby="revisar-titulo">
      <div class="review__head">
        <h2 id="revisar-titulo" class="review__title">Sugestões</h2>
        <span class="review__count">{{ pendingCount }} {{ pendingCount === 1 ? 'pendente' : 'pendentes' }}</span>
      </div>
      <p v-if="reviewError" class="form__error" role="alert">{{ reviewError }}</p>
      <TlCard v-if="!suggestions.length" tone="muted">Ninguém sugeriu eventos ainda.</TlCard>
      <TlCard v-for="s in suggestions" :key="s.id" as="article" class="sug">
        <TlAvatar :name="s.by" tint="neutral" size="lg" decorative />
        <div class="sug__body">
          <div class="sug__line">
            <strong class="sug__title">{{ s.event.title }}</strong>
            <span class="sug__muted">{{ dateLabel(s.event, cal) }}</span>
          </div>
          <span class="sug__muted">{{ suggestionKindLabel[s.kind] }} por {{ handleLabel(s.by) }} · {{ timeAgo(s.createdAt) }}</span>
          <p class="sug__desc">{{ s.event.description }}</p>
          <p v-if="s.note" class="sug__note">“{{ s.note }}”</p>
          <a v-if="s.event.sourceUrl" :href="s.event.sourceUrl" target="_blank" rel="noopener" class="sug__source">Fonte</a>
          <form v-if="adjusting === s.id" class="adjust" @submit.prevent="askAdjust(s.id)">
            <TlField v-slot="{ id }" label="O que precisa mudar?">
              <TlInput :id="id" v-model="feedback" />
            </TlField>
            <div class="adjust__actions">
              <TlButton size="sm" variant="ghost" @click="adjusting = undefined">Cancelar</TlButton>
              <TlButton size="sm" type="submit" variant="primary" :disabled="busy === s.id">Enviar pedido</TlButton>
            </div>
          </form>
        </div>
        <div v-if="s.status === 'pendente' && adjusting !== s.id" class="sug__actions">
          <TlButton :disabled="busy === s.id" @click="decide(s.id, 'reject')">Recusar</TlButton>
          <TlButton :disabled="busy === s.id" @click="adjusting = s.id">Pedir ajuste</TlButton>
          <TlButton variant="success" icon="check" :disabled="busy === s.id" @click="decide(s.id, 'approve')">Aprovar</TlButton>
        </div>
        <div v-else-if="s.status !== 'pendente'" class="sug__actions">
          <TlBadge size="md" :tone="statusBadge[s.status].tone">{{ statusBadge[s.status].label }}</TlBadge>
          <TlButton variant="ghost" :disabled="busy === s.id" @click="undo(s.id)">Desfazer</TlButton>
        </div>
      </TlCard>
    </section>
  </main>
</template>

<style scoped>
.page {
  padding-top: var(--tl-space-7);
  padding-bottom: calc(var(--tl-space-8) * 2);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-7);
}
.page__head {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
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
.page__title {
  font-size: var(--tl-text-4xl);
  letter-spacing: var(--tl-tracking-tight);
}
.page__lead {
  max-width: 640px;
  font-size: 16px;
  color: var(--tl-color-text-muted);
}
.form__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
.closed {
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: var(--tl-space-4);
}
.split {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-7);
  align-items: start;
}
.form {
  flex: 999 1 560px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 22px;
}
.when {
  margin: 0;
  padding: 0;
  border: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.when__legend {
  padding: 0;
  margin-bottom: var(--tl-space-2);
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
}
.when__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 10px;
}
.when__grid--year {
  grid-template-columns: minmax(0, 1fr);
  max-width: 200px;
}
.when__end {
  max-width: 200px;
}
.form__row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--tl-space-4);
}
.form__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: end;
  gap: 10px;
  padding-top: 6px;
  border-top: 1px solid var(--tl-color-border-subtle);
}
.sent {
  gap: var(--tl-space-3);
}
.sent__title {
  font-size: var(--tl-text-2xl);
}
.sent__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
  margin-top: var(--tl-space-2);
}
.aside {
  flex: 1 1 320px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.preview {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.preview__title {
  font-family: var(--tl-font-body);
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-muted);
  letter-spacing: 0;
}
.how {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.how__title {
  font-family: var(--tl-font-body);
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  letter-spacing: 0;
}
.how__list {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-body);
}
.how__list li {
  display: flex;
  gap: var(--tl-space-3);
}
.how__list li span {
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-primary-soft);
  color: var(--tl-color-primary);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-sm);
}
.mine {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 10px;
  font-size: var(--tl-text-md);
}
.mine li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--tl-space-3);
}

.review {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
}
.review__head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--tl-space-3);
}
.review__title {
  font-size: 26px;
}
.review__count {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.sug {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-items: start;
  gap: var(--tl-space-4) var(--tl-space-6);
}
.sug__body {
  flex: 1 1 360px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
}
.sug__line {
  display: flex;
  flex-wrap: wrap;
  gap: 6px var(--tl-space-3);
  align-items: baseline;
}
.sug__title {
  font-size: 16px;
}
.sug__muted {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.sug__desc {
  margin-top: var(--tl-space-1);
  font-size: var(--tl-text-base);
  color: var(--tl-color-text-body);
}
.sug__note {
  font-size: var(--tl-text-md);
  font-style: italic;
  color: var(--tl-color-text-muted);
}
.sug__source {
  font-size: var(--tl-text-sm);
  width: fit-content;
}
.sug__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
}
.adjust {
  margin-top: var(--tl-space-3);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.adjust__actions {
  display: flex;
  gap: var(--tl-space-2);
  justify-content: end;
}
</style>
