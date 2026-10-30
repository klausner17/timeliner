<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
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
import { errorMessage } from '@/data/api'
import {
  currentUser,
  followUser,
  getProfile,
  getTimeline,
  handleLabel,
  listContributions,
  listFollowing,
  listSavedComparisons,
  listTimelines,
  logout,
  rangeLabel,
  updateMe,
} from '@/data/store'
import { suggestionKindLabel, type Profile, type SavedComparison, type Suggestion, type SuggestionStatus, type Timeline, type User, type UserSummary } from '@/data/types'
import { copyLink, monthYear, timeAgo } from '@/lib/format'
import { formatYearRange } from '@/design-system/years'

const route = useRoute()
const router = useRouter()

// A rota exige login, então currentUser existe aqui.
const me = computed(() => currentUser.value as User)

const status = ref<'loading' | 'ok' | 'error'>('loading')
const statusError = ref('')
const profile = shallowRef<Profile>()
const mine = shallowRef<Timeline[]>([])
const contributions = shallowRef<Suggestion[]>([])
const comparisonsRaw = shallowRef<SavedComparison[]>([])
const titles = shallowRef(new Map<string, Timeline>())
const followedTimelines = shallowRef<Timeline[]>([])
// Quem a pessoa segue, mais quem ela deixou de seguir nesta visita (para poder voltar atrás).
const followList = ref<UserSummary[]>([])
const followingNow = ref(new Set<string>())

watch(
  () => currentUser.value?.handle,
  async (handle) => {
    if (!handle) return
    status.value = 'loading'
    try {
      const [p, own, contrib, saved, following] = await Promise.all([
        getProfile(handle),
        listTimelines({ owner: 'me', pageSize: 100, sort: 'recent' }),
        listContributions(handle),
        listSavedComparisons(),
        listFollowing(handle),
      ])
      profile.value = p
      mine.value = own.items
      contributions.value = contrib
      comparisonsRaw.value = saved
      followedTimelines.value = following.timelines
      followList.value = following.users
      followingNow.value = new Set(following.users.map((u) => u.handle))
      status.value = 'ok'
      titles.value = await timelinesFor(saved, [...own.items, ...following.timelines])
    } catch (e) {
      statusError.value = errorMessage(e, 'Não deu para carregar o perfil.')
      status.value = 'error'
    }
  },
  { immediate: true },
)

/** Busca as timelines das comparações salvas que ainda não vieram nas listas. */
async function timelinesFor(saved: SavedComparison[], known: Timeline[]) {
  const out = new Map(known.map((t) => [t.slug, t]))
  const missing = [...new Set(saved.flatMap((c) => c.timelines))].filter((s) => !out.has(s))
  const found = await Promise.all(missing.map((s) => getTimeline(s).catch(() => undefined)))
  for (const t of found) if (t) out.set(t.slug, t)
  return out
}

const tabs = computed<Tab[]>(() => [
  { value: 'timelines', label: 'Timelines', count: mine.value.length },
  { value: 'contribuicoes', label: 'Contribuições', count: contributions.value.length },
  { value: 'comparacoes', label: 'Comparações salvas', count: comparisonsRaw.value.length },
  { value: 'seguindo', label: 'Seguindo', count: followingNow.value.size + followedTimelines.value.length },
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
  comparisonsRaw.value.map((c) => ({ ...c, items: c.timelines.map((s) => titles.value.get(s)).filter((t): t is Timeline => !!t) })).filter((c) => c.items.length > 1),
)
const compareLink = (c: { timelines: string[]; start: number; end: number }) => ({
  path: '/comparar',
  query: { ...Object.fromEntries(c.timelines.map((s, i) => ['abcd'[i], s])), de: String(c.start), ate: String(c.end) },
})

async function toggleFollowUser(handle: string) {
  const on = !followingNow.value.has(handle)
  try {
    await followUser(handle, on)
    const next = new Set(followingNow.value)
    if (on) next.add(handle)
    else next.delete(handle)
    followingNow.value = next
  } catch (e) {
    shareMsg.value = errorMessage(e)
  }
}

// ---------- Bio ----------

const editingBio = ref(false)
const bioDraft = ref('')
const bioError = ref('')
const savingBio = ref(false)
function editBio() {
  bioDraft.value = me.value.bio
  bioError.value = ''
  editingBio.value = true
}
async function saveBio() {
  savingBio.value = true
  try {
    await updateMe({ bio: bioDraft.value.trim() })
    editingBio.value = false
  } catch (e) {
    bioError.value = errorMessage(e, 'Não deu para salvar a bio.')
  } finally {
    savingBio.value = false
  }
}

function signOut() {
  logout()
  router.replace('/')
}

const shareMsg = ref('')
async function share() {
  shareMsg.value = (await copyLink(window.location.href)) ? 'Link copiado' : 'Não deu para copiar o link'
  setTimeout(() => (shareMsg.value = ''), 2500)
}
</script>

<template>
  <div v-if="me">
    <section class="top">
      <div class="tl-container top__inner">
        <div class="who">
          <TlAvatar :name="me.handle" :initial="me.name.charAt(0)" :tint="me.tint" size="xl" decorative />
          <div class="who__text">
            <h1 class="who__name">{{ me.name }}</h1>
            <div class="who__meta">
              <span>{{ handleLabel(me.handle) }}</span>
              <span v-if="me.memberSince">Membro desde {{ monthYear(me.memberSince) }}</span>
            </div>
            <form v-if="editingBio" class="bio" @submit.prevent="saveBio">
              <TlField v-slot="{ id }" label="Bio">
                <TlTextarea :id="id" v-model="bioDraft" :rows="2" :maxlength="160" />
              </TlField>
              <p v-if="bioError" class="who__error" role="alert">{{ bioError }}</p>
              <div class="bio__actions">
                <TlButton size="sm" variant="ghost" @click="editingBio = false">Cancelar</TlButton>
                <TlButton size="sm" type="submit" variant="primary" :disabled="savingBio">Salvar</TlButton>
              </div>
            </form>
            <p v-else class="who__bio" :class="{ 'who__bio--empty': !me.bio }">{{ me.bio || 'Conte do que você gosta de fazer timelines.' }}</p>
          </div>
          <div class="who__actions">
            <TlButton v-if="!editingBio" @click="editBio">Editar perfil</TlButton>
            <TlButton icon="share" icon-only label="Compartilhar perfil" @click="share" />
            <TlButton variant="ghost" @click="signOut">Sair</TlButton>
            <span class="who__toast" role="status">{{ shareMsg }}</span>
          </div>
        </div>

        <dl class="stats">
          <TlStat label="Timelines" :value="mine.length" />
          <TlStat label="Eventos criados" :value="profile?.stats.events ?? 0" />
          <TlStat label="Contribuições aprovadas" :value="profile?.stats.approvedContributions ?? 0" />
          <TlStat label="Seguidores" :value="profile?.stats.followers ?? 0" />
        </dl>
      </div>
    </section>

    <main v-if="status === 'loading'" class="tl-container body" aria-busy="true"></main>
    <main v-else-if="status === 'error'" class="tl-container body">
      <p class="who__error" role="alert">{{ statusError }}</p>
    </main>
    <main v-else class="tl-container body">
      <TlTabs v-model="tab" label="Seções do perfil" :tabs="tabs">
        <div class="panel">
          <div v-if="tab === 'timelines'" class="grid">
            <TlTimelineCard
              v-for="t in mine"
              :key="t.slug"
              :to="t.draft ? `/t/${t.slug}/editar` : `/t/${t.slug}`"
              :title="t.title"
              :color="t.color"
              :points="t.points"
              :range="rangeLabel(t)"
              :events="t.eventCount"
              :contributors="t.visibility === 'privada' ? undefined : t.contributorCount"
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
                    {{ suggestionKindLabel[c.kind] }} em <RouterLink :to="`/t/${c.timeline.slug}`">{{ c.timeline.title }}</RouterLink> · {{ timeAgo(c.createdAt) }}
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
                <span class="cmp__meta">{{ formatYearRange(c.start, c.end) }} · salva {{ timeAgo(c.createdAt) }}</span>
              </TlCard>
            </div>
            <TlCard v-else tone="muted">Nenhuma comparação salva. Monte uma em Comparar e use “Salvar comparação”.</TlCard>
          </template>

          <template v-else>
            <div v-if="followedTimelines.length" class="grid">
              <TlTimelineCard
                v-for="t in followedTimelines"
                :key="t.slug"
                :to="`/t/${t.slug}`"
                :title="t.title"
                :color="t.color"
                :points="t.points"
                :range="rangeLabel(t)"
                :events="t.eventCount"
                :contributors="t.contributorCount"
              />
            </div>
            <div v-if="followList.length" class="grid grid--people">
              <TlCard v-for="u in followList" :key="u.handle" padding="sm" class="person">
                <TlAvatar :name="u.handle" :initial="u.name.charAt(0)" :tint="u.tint" size="lg" decorative />
                <div class="person__text">
                  <span class="person__name">{{ u.name }}</span>
                  <span class="person__info">{{ handleLabel(u.handle) }}</span>
                </div>
                <TlButton
                  size="sm"
                  :aria-pressed="followingNow.has(u.handle)"
                  :variant="followingNow.has(u.handle) ? 'secondary' : 'dark'"
                  @click="toggleFollowUser(u.handle)"
                >
                  {{ followingNow.has(u.handle) ? 'Seguindo' : 'Seguir' }}
                </TlButton>
              </TlCard>
            </div>
            <TlCard v-if="!followedTimelines.length && !followList.length" tone="muted">Você ainda não segue ninguém nem nenhuma timeline.</TlCard>
          </template>
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
.who__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
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
