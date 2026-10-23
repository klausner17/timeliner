<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  TlAvatar,
  TlBadge,
  TlButton,
  TlCard,
  TlField,
  TlIcon,
  TlStat,
  TlTabs,
  TlTextarea,
  TlTimelineCard,
  TlTrackLabel,
  trackColors,
  type BadgeTone,
  type Tab,
} from '@/design-system'
import {
  contributors,
  currentUser,
  findTimeline,
  findUser,
  isFollowingUser,
  rangeLabel,
  state,
  suggestionsBy,
  timelinesOf,
  toggleFollowUser,
  trackPoints,
} from '@/data/store'
import type { SuggestionStatus, Timeline } from '@/data/types'
import { copyLink } from '@/lib/format'

const route = useRoute()
const router = useRouter()

const me = currentUser
const mine = computed(() => timelinesOf(me.value.handle))
const contributions = computed(() => suggestionsBy(me.value.handle))
const eventsCreated = computed(() => state.timelines.reduce((n, t) => n + t.events.filter((e) => e.by === me.value.handle).length, 0))
const followers = computed(() => mine.value.reduce((n, t) => n + t.followers, 0))
// Quem a pessoa segue, mais quem ela deixou de seguir nesta visita (para poder voltar atrás).
const followList = ref([...state.following])

const tabs = computed<Tab[]>(() => [
  { value: 'timelines', label: 'Timelines', count: mine.value.length },
  { value: 'contribuicoes', label: 'Contribuições', count: contributions.value.length },
  { value: 'comparacoes', label: 'Comparações salvas', count: state.savedComparisons.length },
  { value: 'seguindo', label: 'Seguindo', count: state.following.length },
])
const tab = computed({
  get: () => (tabs.value.some((t) => t.value === route.query.aba) ? String(route.query.aba) : 'timelines'),
  set: (v: string) => router.replace({ query: { ...route.query, aba: v === 'timelines' ? undefined : v } }),
})

function badgeOf(t: Timeline): { label: string; tone: BadgeTone } {
  if (t.draft) return { label: 'Rascunho', tone: 'warning' }
  if (t.visibility === 'privada') return { label: 'Privada', tone: 'neutral' }
  if (t.visibility === 'link') return { label: 'Só com o link', tone: 'info' }
  return { label: 'Pública', tone: 'success' }
}

const statusBadge: Record<SuggestionStatus, { label: string; tone: BadgeTone }> = {
  pendente: { label: 'Pendente', tone: 'info' },
  aprovada: { label: 'Aprovada', tone: 'success' },
  recusada: { label: 'Recusada', tone: 'neutral' },
  ajuste: { label: 'Ajuste pedido', tone: 'warning' },
}

const comparisons = computed(() =>
  state.savedComparisons.map((c) => ({ ...c, items: c.timelines.map(findTimeline).filter((t): t is Timeline => !!t) })).filter((c) => c.items.length > 1),
)
const compareLink = (c: { timelines: string[]; start: number; end: number }) => ({
  path: '/comparar',
  query: { ...Object.fromEntries(c.timelines.map((s, i) => ['abcd'[i], s])), de: String(c.start), ate: String(c.end) },
})

const followCount = (handle: string) => timelinesOf(handle).filter((t) => t.visibility === 'publica' && !t.draft).length

// ---------- Bio ----------

const editingBio = ref(false)
const bioDraft = ref('')
function editBio() {
  bioDraft.value = state.bio
  editingBio.value = true
}
function saveBio() {
  state.bio = bioDraft.value.trim()
  editingBio.value = false
}

const shareMsg = ref('')
async function share() {
  shareMsg.value = (await copyLink(window.location.href)) ? 'Link copiado' : 'Não deu para copiar o link'
  setTimeout(() => (shareMsg.value = ''), 2500)
}
</script>

<template>
  <div>
    <section class="top">
      <div class="tl-container top__inner">
        <div class="who">
          <TlAvatar :name="me.handle" :initial="me.name.charAt(0)" :tint="me.tint" size="xl" decorative />
          <div class="who__text">
            <h1 class="who__name">{{ me.name }}</h1>
            <div class="who__meta">
              <span>{{ me.handle }}</span>
              <span v-if="me.memberSince">Membro desde {{ me.memberSince }}</span>
            </div>
            <form v-if="editingBio" class="bio" @submit.prevent="saveBio">
              <TlField v-slot="{ id }" label="Bio">
                <TlTextarea :id="id" v-model="bioDraft" :rows="2" :maxlength="160" />
              </TlField>
              <div class="bio__actions">
                <TlButton size="sm" variant="ghost" @click="editingBio = false">Cancelar</TlButton>
                <TlButton size="sm" type="submit" variant="primary">Salvar</TlButton>
              </div>
            </form>
            <p v-else class="who__bio" :class="{ 'who__bio--empty': !state.bio }">{{ state.bio || 'Conte do que você gosta de fazer timelines.' }}</p>
          </div>
          <div class="who__actions">
            <TlButton v-if="!editingBio" @click="editBio">Editar perfil</TlButton>
            <TlButton icon="share" icon-only label="Compartilhar perfil" @click="share" />
            <span class="who__toast" role="status">{{ shareMsg }}</span>
          </div>
        </div>

        <dl class="stats">
          <TlStat label="Timelines" :value="mine.length" />
          <TlStat label="Eventos criados" :value="eventsCreated" />
          <TlStat label="Contribuições aprovadas" :value="contributions.filter((c) => c.status === 'aprovada').length" />
          <TlStat label="Seguidores" :value="followers" />
        </dl>
      </div>
    </section>

    <main class="tl-container body">
      <TlTabs v-model="tab" label="Seções do perfil" :tabs="tabs">
        <div class="panel">
          <div v-if="tab === 'timelines'" class="grid">
            <TlTimelineCard
              v-for="t in mine"
              :key="t.slug"
              :to="t.draft ? `/t/${t.slug}/editar` : `/t/${t.slug}`"
              :title="t.title"
              :color="t.color"
              :points="trackPoints(t)"
              :range="rangeLabel(t)"
              :events="t.events.length"
              :contributors="t.visibility === 'privada' ? undefined : contributors(t).length"
              :badge="badgeOf(t)"
            />
            <TlCard to="/nova" tone="dashed" class="new">
              <TlIcon name="plus" :size="24" />
              Nova timeline
            </TlCard>
          </div>

          <template v-else-if="tab === 'contribuicoes'">
            <TlCard v-if="contributions.length" padding="sm" class="rows">
              <div v-for="c in contributions" :key="c.id" class="contrib">
                <div class="contrib__text">
                  <span class="contrib__title">{{ c.event.title }}</span>
                  <span class="contrib__meta">
                    {{ c.kind }} em <RouterLink :to="`/t/${c.timeline}`">{{ findTimeline(c.timeline)?.title ?? c.timeline }}</RouterLink> · {{ c.when }}
                  </span>
                  <span v-if="c.feedback" class="contrib__meta">Pedido do dono: “{{ c.feedback }}”</span>
                </div>
                <TlBadge size="md" :tone="statusBadge[c.status].tone">{{ statusBadge[c.status].label }}</TlBadge>
              </div>
            </TlCard>
            <TlCard v-else tone="muted">Você ainda não sugeriu eventos. Abra uma timeline aberta a contribuições e use “Sugerir evento”.</TlCard>
          </template>

          <template v-else-if="tab === 'comparacoes'">
            <div v-if="comparisons.length" class="grid grid--wide">
              <TlCard v-for="c in comparisons" :key="c.id" :to="compareLink(c)" class="cmp">
                <TlTrackLabel v-for="(t, i) in c.items" :key="t.slug" :color="trackColors[i]">{{ t.title }}</TlTrackLabel>
                <span class="cmp__meta">{{ c.start }}–{{ c.end }} · salva {{ c.when }}</span>
              </TlCard>
            </div>
            <TlCard v-else tone="muted">Nenhuma comparação salva. Monte uma em Comparar e use “Salvar comparação”.</TlCard>
          </template>

          <div v-else class="grid grid--people">
            <TlCard v-for="h in followList" :key="h" padding="sm" class="person">
              <TlAvatar :name="h" :tint="findUser(h).tint" size="lg" decorative />
              <div class="person__text">
                <span class="person__name">{{ h }}</span>
                <span class="person__info">{{ followCount(h) }} {{ followCount(h) === 1 ? 'timeline' : 'timelines' }}</span>
              </div>
              <TlButton size="sm" :aria-pressed="isFollowingUser(h)" :variant="isFollowingUser(h) ? 'secondary' : 'dark'" @click="toggleFollowUser(h)">
                {{ isFollowingUser(h) ? 'Seguindo' : 'Seguir' }}
              </TlButton>
            </TlCard>
          </div>
        </div>
      </TlTabs>
    </main>
  </div>
</template>

<style scoped>
.top {
  background: var(--tl-color-surface);
  border-bottom: 1px solid var(--tl-color-border);
}
.top__inner {
  padding-top: 36px;
  padding-bottom: var(--tl-space-6);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-6);
}
.who {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-6);
}
.who__text {
  flex: 1 1 360px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.who__name {
  font-size: var(--tl-text-4xl);
  letter-spacing: var(--tl-tracking-tight);
}
.who__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px var(--tl-space-4);
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.who__bio {
  max-width: 620px;
  font-size: 16px;
  color: var(--tl-color-text-body);
}
.who__bio--empty {
  color: var(--tl-color-text-muted);
  font-style: italic;
}
.who__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
}
.who__toast {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.who__toast:empty {
  display: none;
}
.bio {
  max-width: 620px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.bio__actions {
  display: flex;
  justify-content: end;
  gap: var(--tl-space-2);
}
.stats {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: var(--tl-space-3);
}
.body {
  padding-top: var(--tl-space-6);
  padding-bottom: calc(var(--tl-space-8) * 2);
}
.panel {
  padding-top: var(--tl-space-6);
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(300px, 100%), 1fr));
  gap: var(--tl-space-5);
}
.grid--wide {
  grid-template-columns: repeat(auto-fill, minmax(min(340px, 100%), 1fr));
}
.grid--people {
  grid-template-columns: repeat(auto-fill, minmax(min(260px, 100%), 1fr));
  gap: var(--tl-space-3);
}
.new {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 200px;
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-body);
  text-decoration: none;
}
.rows {
  padding: 0;
  overflow: hidden;
}
.contrib {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px var(--tl-space-5);
  padding: var(--tl-space-4) var(--tl-space-5);
  border-bottom: 1px solid var(--tl-color-border-subtle);
}
.contrib:last-child {
  border-bottom: 0;
}
.contrib__text {
  flex: 1 1 360px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
}
.contrib__title {
  font-weight: var(--tl-weight-semibold);
}
.contrib__meta {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.cmp {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.cmp__meta {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.person {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: var(--tl-space-3);
}
.person__text {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.person__name {
  font-weight: var(--tl-weight-semibold);
}
.person__info {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
