<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { TlButton, TlCard, TlChip, TlInput, TlSegmented, TlTimelineCard } from '@/design-system'
import { handleLabel, listTimelines, rangeLabel, type TimelineQuery } from '@/data/store'
import { categories, type TimelinePage } from '@/data/types'
import { useLatest } from '@/lib/useLatest'

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

// A busca (título, descrição e tags), a categoria e a ordem vão para a API.
const params = computed<TimelineQuery>(() => ({
  q: query.value || undefined,
  cat: category.value || undefined,
  sort: order.value === 'recentes' ? 'recent' : 'popular',
  openToContributions: order.value === 'abertas' || undefined,
  pageSize: 48,
}))
const { data: page, error } = useLatest(() => listTimelines(params.value), undefined as TimelinePage | undefined)
const results = computed(() => page.value?.items ?? [])
const total = computed(() => page.value?.total ?? 0)

const filtering = computed(() => !!(query.value || category.value))

function submit() {
  router.push({ query: { ...route.query, q: search.value.trim() || undefined } })
}

function categoryLink(c: string) {
  return { query: { ...route.query, cat: category.value === c ? undefined : c } }
}

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
              {{ total }} {{ total === 1 ? 'timeline' : 'timelines' }}
              <template v-if="query"> para “{{ query }}”</template>
              <template v-if="category"> em {{ category }}</template>
            </template>
            <template v-else>Em destaque</template>
          </h2>
          <TlSegmented v-model="order" label="Ordenar timelines" :options="orders" />
        </div>
        <p v-if="error" class="home__error" role="alert">{{ error }}</p>
        <div v-if="!page" class="home__grid" aria-busy="true"></div>
        <div v-else-if="results.length" class="home__grid">
          <TlTimelineCard
            v-for="t in results"
            :key="t.slug"
            :to="`/t/${t.slug}`"
            :title="t.title"
            :description="t.description"
            :color="t.color"
            :points="t.points"
            :range="rangeLabel(t)"
            :events="t.eventCount"
            :contributors="t.contributorCount"
            :author="handleLabel(t.owner.handle)"
          />
        </div>
        <TlCard v-else tone="muted" class="home__empty">
          <p v-if="filtering">Nenhuma timeline encontrada. Tente outra busca ou crie a sua.</p>
          <p v-else>Ainda não há timelines públicas. Que tal criar a primeira?</p>
          <div class="home__empty-actions">
            <TlButton v-if="filtering" :to="{ query: {} }">Limpar busca</TlButton>
            <TlButton to="/nova" variant="dark" icon="plus">Criar timeline</TlButton>
          </div>
        </TlCard>
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
.home__error {
  font-size: var(--tl-text-md);
  color: var(--tl-color-warning-strong);
}
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
.home__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(300px, 100%), 1fr));
  gap: var(--tl-space-5);
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
