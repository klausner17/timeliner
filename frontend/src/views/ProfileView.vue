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
  TlInput,
  TlStat,
  TlTabs,
  TlTextarea,
  TlTimelineCard,
  TlTrackLabel,
  trackColors,
  type BadgeTone,
  type Tab,
} from '@/design-system'
import { ApiError, errorMessage } from '@/data/api'
import {
  changePassword,
  currentUser,
  followUser,
  getProfile,
  handleLabel,
  listContributions,
  listFollowing,
  listSavedComparisons,
  listTimelines,
  logout,
  rangeLabel,
  updateMe,
} from '@/data/store'
import { suggestionKindLabel, type Profile, type SavedComparison, type SuggestionStatus, type Timeline, type User } from '@/data/types'
import { copyLink, monthYear, timeAgo } from '@/lib/format'
import { useFavorite } from '@/lib/useFavorite'
import { usePages } from '@/lib/usePaged'

const route = useRoute()
const router = useRouter()

// A rota exige login, então currentUser existe aqui.
const me = computed(() => currentUser.value as User)

const status = ref<'loading' | 'ok' | 'error'>('loading')
const statusError = ref('')
const profile = shallowRef<Profile>()
// Cada aba vem do servidor uma página por vez, com "Carregar mais".
const PAGE = 24
const handle = () => currentUser.value?.handle ?? ''
const mineP = usePages((page) => listTimelines({ owner: 'me', pageSize: PAGE, sort: 'recent', page }), (t) => t.slug)
const contribP = usePages((page) => listContributions(handle(), { page, pageSize: PAGE }), (s) => s.id)
const savedP = usePages((page) => listSavedComparisons({ page, pageSize: PAGE }), (c) => c.id)
const favP = usePages((page) => listTimelines({ favoritedBy: handle(), pageSize: PAGE, page }), (t) => t.slug)
const favorites = favP.items
// Tirar a estrela aqui não some com o cartão: dá para voltar atrás.
const { toggle: toggleFavorite, busy: favBusy } = useFavorite((msg) => (statusError.value = msg))
const followingPage = (page: number) => listFollowing(handle(), { page, pageSize: PAGE })
const followedTimelinesP = usePages(
  (page) => followingPage(page).then((f) => ({ items: f.timelines, total: f.timelinesTotal, page: f.page, pageSize: f.pageSize })),
  (t) => t.slug,
)
const followUsersP = usePages(
  (page) => followingPage(page).then((f) => ({ items: f.users, total: f.usersTotal, page: f.page, pageSize: f.pageSize })),
  (u) => u.handle,
)
const mine = mineP.items
const mineTotal = mineP.total
const contributions = contribP.items
const comparisonsRaw = savedP.items
const titles = shallowRef(new Map<string, Timeline>())
const followedTimelines = followedTimelinesP.items
// Quem a pessoa segue, mais quem ela deixou de seguir nesta visita (para poder voltar atrás).
const followList = followUsersP.items
const followingNow = ref(new Set<string>())
watch(followList, (us) => {
  const next = new Set(followingNow.value)
  for (const u of us) if (!unfollowed.has(u.handle)) next.add(u.handle)
  followingNow.value = next
})
const unfollowed = new Set<string>()

async function more(load: () => Promise<void>) {
  try {
    await load()
  } catch (e) {
    statusError.value = errorMessage(e, 'Não deu para carregar mais.')
  }
}

watch(
  () => currentUser.value?.handle,
  async (handle) => {
    if (!handle) return
    status.value = 'loading'
    try {
      const [p, own, contrib, saved, following, favs] = await Promise.all([
        getProfile(handle),
        listTimelines({ owner: 'me', pageSize: PAGE, sort: 'recent' }),
        listContributions(handle, { pageSize: PAGE }),
        listSavedComparisons({ pageSize: PAGE }),
        listFollowing(handle, { pageSize: PAGE }),
        listTimelines({ favoritedBy: handle, pageSize: PAGE }),
      ])
      profile.value = p
      mineP.reset(own)
      contribP.reset(contrib)
      savedP.reset(saved)
      favP.reset(favs)
      followedTimelinesP.reset({ items: following.timelines, total: following.timelinesTotal, page: 1, pageSize: PAGE })
      unfollowed.clear()
      followingNow.value = new Set()
      followUsersP.reset({ items: following.users, total: following.usersTotal, page: 1, pageSize: PAGE })
      status.value = 'ok'
    } catch (e) {
      statusError.value = errorMessage(e, 'Não deu para carregar o perfil.')
      status.value = 'error'
    }
  },
  { immediate: true },
)

/** Busca, de uma vez só, as timelines das comparações salvas que ainda não vieram nas listas. */
async function timelinesFor(saved: SavedComparison[], known: Map<string, Timeline>) {
  const out = new Map(known)
  const missing = [...new Set(saved.flatMap((c) => c.timelines))].filter((s) => !out.has(s))
  for (let i = 0; i < missing.length; i += 20) {
    const chunk = missing.slice(i, i + 20)
    const page = await listTimelines({ slugs: chunk.join(','), pageSize: chunk.length }).catch(() => undefined)
    for (const t of page?.items ?? []) out.set(t.slug, t)
  }
  return out
}

// As timelines das comparações salvas: as que já vieram nas outras listas, e o resto numa chamada.
watch(comparisonsRaw, async (saved) => {
  const known = new Map(titles.value)
  for (const t of [...mine.value, ...followedTimelines.value, ...favorites.value]) known.set(t.slug, t)
  titles.value = await timelinesFor(saved, known)
})

const tabs = computed<Tab[]>(() => [
  { value: 'timelines', label: 'Timelines', count: mineTotal.value },
  { value: 'contribuicoes', label: 'Contribuições', count: contribP.total.value },
  { value: 'comparacoes', label: 'Comparações salvas', count: savedP.total.value },
  { value: 'favoritas', label: 'Favoritas', count: favP.total.value },
  { value: 'seguindo', label: 'Seguindo', count: followUsersP.total.value - unfollowedCount.value + followedTimelinesP.total.value },
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

const unfollowedCount = ref(0)
async function toggleFollowUser(handle: string) {
  const on = !followingNow.value.has(handle)
  try {
    await followUser(handle, on)
    const next = new Set(followingNow.value)
    if (on) next.add(handle)
    else next.delete(handle)
    followingNow.value = next
    if (on) unfollowed.delete(handle)
    else unfollowed.add(handle)
    unfollowedCount.value = unfollowed.size
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

// ---------- Senha ----------

const editingPassword = ref(false)
const pw = ref({ current: '', next: '' })
const pwFields = ref<Record<string, string>>({})
const pwError = ref('')
const savingPassword = ref(false)
function editPassword() {
  pw.value = { current: '', next: '' }
  pwFields.value = {}
  pwError.value = ''
  editingBio.value = false
  editingPassword.value = true
}
async function savePassword() {
  pwError.value = ''
  pwFields.value = {}
  if (!pw.value.current) pwFields.value.currentPassword = 'Informe a senha atual.'
  if (pw.value.next.length < 8) pwFields.value.newPassword = 'A senha precisa ter pelo menos 8 caracteres.'
  if (Object.keys(pwFields.value).length) return
  savingPassword.value = true
  try {
    await changePassword(pw.value.current, pw.value.next)
    editingPassword.value = false
    shareMsg.value = 'Senha trocada. As outras sessões foram encerradas.'
    setTimeout(() => (shareMsg.value = ''), 4000)
  } catch (e) {
    pwError.value = errorMessage(e, 'Não deu para trocar a senha.')
    if (e instanceof ApiError) pwFields.value = e.fields
  } finally {
    savingPassword.value = false
  }
}

async function signOut() {
  await logout()
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
            <form v-else-if="editingPassword" class="bio" novalidate @submit.prevent="savePassword">
              <TlField v-slot="{ id, describedby, invalid }" label="Senha atual" :error="pwFields.currentPassword">
                <TlInput :id="id" v-model="pw.current" type="password" autocomplete="current-password" :aria-describedby="describedby" :invalid="invalid" />
              </TlField>
              <TlField v-slot="{ id, describedby, invalid }" label="Senha nova" hint="Pelo menos 8 caracteres. As outras sessões abertas serão encerradas." :error="pwFields.newPassword">
                <TlInput :id="id" v-model="pw.next" type="password" autocomplete="new-password" :aria-describedby="describedby" :invalid="invalid" />
              </TlField>
              <p v-if="pwError" class="who__error" role="alert">{{ pwError }}</p>
              <div class="bio__actions">
                <TlButton size="sm" variant="ghost" @click="editingPassword = false">Cancelar</TlButton>
                <TlButton size="sm" type="submit" variant="primary" :disabled="savingPassword">Trocar senha</TlButton>
              </div>
            </form>
            <p v-else class="who__bio" :class="{ 'who__bio--empty': !me.bio }">{{ me.bio || 'Conte do que você gosta de fazer timelines.' }}</p>
          </div>
          <div class="who__actions">
            <TlButton v-if="!editingBio && !editingPassword" @click="editBio">Editar perfil</TlButton>
            <TlButton v-if="!editingBio && !editingPassword" variant="ghost" @click="editPassword">Trocar senha</TlButton>
            <TlButton icon="share" icon-only label="Compartilhar perfil" @click="share" />
            <TlButton variant="ghost" @click="signOut">Sair</TlButton>
            <span class="who__toast" role="status">{{ shareMsg }}</span>
          </div>
        </div>

        <dl class="stats">
          <TlStat label="Timelines" :value="mineTotal" />
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
            <div v-if="mineP.hasMore.value" class="more">
              <span class="more__count">Mostrando {{ mine.length }} de {{ mineTotal }}</span>
              <TlButton :disabled="mineP.loading.value" @click="more(mineP.loadMore)">{{ mineP.loading.value ? 'Carregando…' : 'Carregar mais' }}</TlButton>
            </div>
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
            <div v-if="contribP.hasMore.value" class="more">
              <span class="more__count">Mostrando {{ contributions.length }} de {{ contribP.total.value }}</span>
              <TlButton :disabled="contribP.loading.value" @click="more(contribP.loadMore)">{{ contribP.loading.value ? 'Carregando…' : 'Carregar mais' }}</TlButton>
            </div>
            <TlCard v-if="!contributions.length" tone="muted">Você ainda não sugeriu eventos. Abra uma timeline aberta a contribuições e use “Sugerir evento”.</TlCard>
          </template>

          <template v-else-if="tab === 'comparacoes'">
            <div v-if="comparisons.length" class="grid grid--wide">
              <TlCard v-for="c in comparisons" :key="c.id" :to="compareLink(c)" class="cmp">
                <TlTrackLabel v-for="(t, i) in c.items" :key="t.slug" :color="trackColors[i]">{{ t.title }}</TlTrackLabel>
                <span class="cmp__meta">{{ c.start }}–{{ c.end }} · salva {{ timeAgo(c.createdAt) }}</span>
              </TlCard>
            </div>
            <div v-if="savedP.hasMore.value" class="more">
              <span class="more__count">Mostrando {{ comparisonsRaw.length }} de {{ savedP.total.value }}</span>
              <TlButton :disabled="savedP.loading.value" @click="more(savedP.loadMore)">{{ savedP.loading.value ? 'Carregando…' : 'Carregar mais' }}</TlButton>
            </div>
            <TlCard v-if="!comparisons.length && !savedP.hasMore.value" tone="muted">Nenhuma comparação salva. Monte uma em Comparar e use “Salvar comparação”.</TlCard>
          </template>

          <template v-else-if="tab === 'favoritas'">
            <div v-if="favorites.length" class="grid">
              <TlTimelineCard
                v-for="t in favorites"
                :key="t.slug"
                :to="`/t/${t.slug}`"
                :title="t.title"
                :color="t.color"
                :points="t.points"
                :range="rangeLabel(t)"
                :events="t.eventCount"
                :contributors="t.contributorCount"
                :author="handleLabel(t.owner.handle)"
                :views="t.views"
                :favorites="t.favorites"
                :favorited="t.favorited"
                :favorite-busy="favBusy.has(t.slug)"
                @favorite="toggleFavorite(t)"
              />
              <div v-if="favP.hasMore.value" class="more">
                <span class="more__count">Mostrando {{ favorites.length }} de {{ favP.total.value }}</span>
                <TlButton :disabled="favP.loading.value" @click="more(favP.loadMore)">{{ favP.loading.value ? 'Carregando…' : 'Carregar mais' }}</TlButton>
              </div>
            </div>
            <TlCard v-else tone="muted">Nenhuma favorita ainda. Toque na estrela de uma timeline para guardá-la aqui e ajudar ela a subir em Populares.</TlCard>
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
              <div v-if="followedTimelinesP.hasMore.value" class="more">
                <span class="more__count">Mostrando {{ followedTimelines.length }} de {{ followedTimelinesP.total.value }} timelines</span>
                <TlButton :disabled="followedTimelinesP.loading.value" @click="more(followedTimelinesP.loadMore)">Carregar mais timelines</TlButton>
              </div>
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
              <div v-if="followUsersP.hasMore.value" class="more">
                <span class="more__count">Mostrando {{ followList.length }} de {{ followUsersP.total.value }} pessoas</span>
                <TlButton :disabled="followUsersP.loading.value" @click="more(followUsersP.loadMore)">Carregar mais pessoas</TlButton>
              </div>
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
  grid-template-columns: repeat(auto-fit, minmax(min(130px, calc(50% - var(--tl-space-3))), 1fr));
  gap: var(--tl-space-3);
}
.body {
  padding-top: var(--tl-space-6);
  padding-bottom: calc(var(--tl-space-8) * 2);
}
.panel {
  padding-top: var(--tl-space-6);
}
.more {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--tl-space-3);
}
.more__count {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
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
