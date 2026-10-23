<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TlButton, TlCard, TlChip, TlInput, TlSegmented, TlTimelineCard, TlTrackLabel } from '@/design-system'
import { featuredPairs } from '@/data/mock'
import { contributors, findTimeline, publicTimelines, rangeLabel, trackPoints } from '@/data/store'
import { categories, type Timeline } from '@/data/types'

const route = useRoute()
const router = useRouter()

const query = computed(() => (typeof route.query.q === 'string' ? route.query.q : ''))
const category = computed(() => (typeof route.query.cat === 'string' ? route.query.cat : ''))
const search = ref(query.value)
watch(query, (q) => (search.value = q))

const order = ref('populares')
const orders = [
  { value: 'populares', label: 'Populares' },
  { value: 'recentes', label: 'Recentes' },
  { value: 'abertas', label: 'Abertas a contribuição' },
]

const normalize = (s: string) => s.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase()

const results = computed(() => {
  const words = normalize(query.value).split(/\s+/).filter(Boolean)
  let list = publicTimelines.value.filter((t) => {
    if (category.value && t.category !== category.value) return false
    const text = normalize([t.title, t.description, ...t.tags].join(' '))
    return words.every((w) => text.includes(w))
  })
  if (order.value === 'abertas') list = list.filter((t) => t.contributions === 'sugestoes')
  const by = order.value === 'recentes' ? (t: Timeline) => -t.createdAt : (t: Timeline) => -t.followers
  return [...list].sort((a, b) => by(a) - by(b))
})

const filtering = computed(() => !!(query.value || category.value))

function submit() {
  router.push({ query: { ...route.query, q: search.value.trim() || undefined } })
}

function categoryLink(c: string) {
  return { query: { ...route.query, cat: category.value === c ? undefined : c } }
}

const pairs = computed(() =>
  featuredPairs
    .map((p) => ({ ...p, items: p.timelines.map(findTimeline).filter((t): t is Timeline => !!t) }))
    .filter((p) => p.items.length === p.timelines.length),
)

const compareLink = (slugs: string[]) => ({ path: '/comparar', query: { a: slugs[0], b: slugs[1] } })

// Faixa decorativa do topo: duas trilhas com um ano em comum marcado.
const heroA = [8, 26, 47, 71, 90]
const heroB = [17, 47, 62, 84]
</script>

<template>
  <main>
    <section class="hero">
      <div class="tl-container hero__inner">
        <p class="hero__eyebrow">Timelines colaborativas</p>
        <h1 class="hero__title">Toda história fica mais clara quando vira linha do tempo.</h1>
        <p class="hero__lead">
          Crie suas timelines, contribua com as de outras pessoas e coloque duas histórias lado a lado para ver o que aconteceu ao mesmo
          tempo.
        </p>
        <form role="search" class="hero__search" @submit.prevent="submit">
          <label for="busca" class="tl-visually-hidden">Buscar timelines</label>
          <TlInput
            id="busca"
            v-model="search"
            type="search"
            size="lg"
            shape="pill"
            class="hero__input"
            placeholder="Buscar: Brasil Império, Copa do Mundo, internet…"
          />
          <TlButton type="submit" variant="primary" size="lg">Buscar</TlButton>
        </form>
        <div class="hero__chips" role="group" aria-label="Categorias">
          <TlChip v-for="c in categories" :key="c" kind="tag" :to="categoryLink(c)" :class="{ 'hero__chip--on': category === c }" :aria-current="category === c ? 'true' : undefined">
            {{ c }}
          </TlChip>
        </div>
        <div class="hero__tracks" aria-hidden="true">
          <div class="hero__line hero__line--a"></div>
          <div class="hero__line hero__line--b"></div>
          <span v-for="x in heroA" :key="`a${x}`" class="hero__dot hero__dot--a" :style="{ left: `${x}%` }"></span>
          <span v-for="x in heroB" :key="`b${x}`" class="hero__dot hero__dot--b" :style="{ left: `${x}%` }"></span>
          <span class="hero__now" :style="{ left: 'calc(47% + 7px)' }"></span>
        </div>
      </div>
    </section>

    <div class="tl-container home">
      <section class="home__section" aria-labelledby="destaque">
        <div class="home__head">
          <h2 id="destaque" class="home__h2">
            <template v-if="filtering">
              {{ results.length }} {{ results.length === 1 ? 'timeline' : 'timelines' }}
              <template v-if="query"> para “{{ query }}”</template>
              <template v-if="category"> em {{ category }}</template>
            </template>
            <template v-else>Em destaque</template>
          </h2>
          <TlSegmented v-model="order" label="Ordenar timelines" :options="orders" />
        </div>
        <div v-if="results.length" class="home__grid">
          <TlTimelineCard
            v-for="t in results"
            :key="t.slug"
            :to="`/t/${t.slug}`"
            :title="t.title"
            :description="t.description"
            :color="t.color"
            :points="trackPoints(t)"
            :range="rangeLabel(t)"
            :events="t.events.length"
            :contributors="contributors(t).length"
            :author="t.owner"
          />
        </div>
        <TlCard v-else tone="muted" class="home__empty">
          <p>Nenhuma timeline encontrada. Tente outra busca ou crie a sua.</p>
          <div class="home__empty-actions">
            <TlButton :to="{ query: {} }">Limpar busca</TlButton>
            <TlButton to="/nova" variant="dark" icon="plus">Criar timeline</TlButton>
          </div>
        </TlCard>
      </section>

      <section class="home__section" aria-labelledby="comparacoes">
        <div class="home__head">
          <h2 id="comparacoes" class="home__h2">Comparações populares</h2>
          <RouterLink to="/comparar" class="home__more">Montar uma comparação</RouterLink>
        </div>
        <div class="home__grid home__grid--wide">
          <TlCard v-for="p in pairs" :key="p.timelines.join()" :to="compareLink(p.timelines)" class="pair">
            <div class="pair__names">
              <TlTrackLabel color="a">{{ p.items[0].title }}</TlTrackLabel>
              <TlTrackLabel color="b">{{ p.items[1].title }}</TlTrackLabel>
            </div>
            <p class="pair__note">{{ p.note }}</p>
          </TlCard>
        </div>
      </section>

      <TlCard tone="inverse" padding="lg" class="cta">
        <div class="cta__text">
          <h2 class="home__h2">Tem uma história para contar?</h2>
          <p>Comece com poucos eventos. Você decide se a timeline aceita contribuições e quem aprova.</p>
        </div>
        <TlButton to="/nova" variant="inverse" size="lg">Criar minha timeline</TlButton>
      </TlCard>
    </div>
  </main>
</template>

<style scoped>
.hero {
  background: var(--tl-color-surface);
  border-bottom: 1px solid var(--tl-color-border);
}
.hero__inner {
  padding-top: var(--tl-space-18);
  padding-bottom: var(--tl-space-14);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: var(--tl-space-5);
}
.hero__eyebrow {
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
  letter-spacing: var(--tl-tracking-eyebrow);
  text-transform: uppercase;
  color: var(--tl-color-primary);
}
.hero__title {
  max-width: 820px;
  font-size: var(--tl-text-6xl);
  line-height: var(--tl-leading-tight);
  letter-spacing: var(--tl-tracking-tight);
}
.hero__lead {
  max-width: 620px;
  font-size: 19px;
  color: var(--tl-color-text-muted);
}
.hero__search {
  width: 100%;
  max-width: 640px;
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
  margin-top: var(--tl-space-2);
}
.hero__input {
  flex: 1 1 320px;
}
.hero__chips {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: var(--tl-space-2);
}
.hero__chip--on {
  background: var(--tl-color-surface-inverse);
  color: var(--tl-color-text-inverse);
}
.hero__tracks {
  width: 100%;
  max-width: 900px;
  margin-top: var(--tl-space-7);
  position: relative;
  height: 96px;
}
.hero__line {
  position: absolute;
  left: 0;
  right: 0;
  height: 2px;
}
.hero__line--a {
  top: 30px;
  background: var(--tl-track-a);
}
.hero__line--b {
  top: 66px;
  background: var(--tl-track-b);
}
.hero__dot {
  position: absolute;
  width: 16px;
  height: 16px;
  border-radius: var(--tl-radius-pill);
}
.hero__dot--a {
  top: 23px;
  background: var(--tl-track-a);
}
.hero__dot--b {
  top: 59px;
  background: var(--tl-track-b);
}
.hero__now {
  position: absolute;
  top: 8px;
  width: 2px;
  height: 82px;
  background: var(--tl-color-text);
  opacity: 0.25;
}

.home {
  padding-top: var(--tl-space-12);
  padding-bottom: calc(var(--tl-space-8) * 2);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-14);
}
.home__section {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-5);
}
.home__head {
  display: flex;
  flex-wrap: wrap;
  align-items: end;
  justify-content: space-between;
  gap: var(--tl-space-3);
}
.home__h2 {
  font-size: var(--tl-text-3xl);
}
.home__more {
  font-weight: var(--tl-weight-semibold);
  padding: 10px 0;
}
.home__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(300px, 100%), 1fr));
  gap: var(--tl-space-5);
}
.home__grid--wide {
  grid-template-columns: repeat(auto-fill, minmax(min(340px, 100%), 1fr));
}
.home__empty {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
  align-items: start;
}
.home__empty-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
}
.pair {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-4);
}
.pair__names {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.pair__note {
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-muted);
}
.cta {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-6);
}
.cta__text {
  flex: 1 1 420px;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.cta__text p {
  color: var(--tl-color-text-inverse-muted);
}

@media (max-width: 640px) {
  .hero__inner {
    padding-top: var(--tl-space-12);
  }
  .hero__title {
    font-size: var(--tl-text-4xl);
  }
}
</style>
