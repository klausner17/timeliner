<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import type { TrackColor } from '../tokens'
import TlAvatar from './TlAvatar.vue'
import TlBadge, { type BadgeTone } from './TlBadge.vue'
import TlCard from './TlCard.vue'
import TlFavoriteButton from './TlFavoriteButton.vue'
import TlMiniTrack from './TlMiniTrack.vue'

defineProps<{
  title: string
  description?: string
  color: TrackColor
  points: number[]
  /** Ex.: "1500–2024". */
  range: string
  events: number
  contributors?: number
  /** Visualizações; sem ele não aparece. */
  views?: number
  author?: string
  badge?: { label: string; tone: BadgeTone }
  to?: RouteLocationRaw
  href?: string
  /** Total de estrelas; sem ele o cartão não mostra o botão de favoritar. */
  favorites?: number
  favorited?: boolean
  favoriteBusy?: boolean
}>()
defineEmits<{ favorite: [] }>()
</script>

<!-- O botão de favoritar fica fora do link (botão dentro de <a> não vale) e por cima do rodapé. -->
<template>
  <div class="tl-timeline-card" :class="{ 'tl-timeline-card--fav': favorites !== undefined }">
    <TlCard :to="to" :href="href" class="tl-timeline-card__body">
      <TlMiniTrack :color="color" :points="points" />
      <div class="tl-timeline-card__head">
        <h3 class="tl-timeline-card__title">{{ title }}</h3>
        <TlBadge v-if="badge" :tone="badge.tone">{{ badge.label }}</TlBadge>
      </div>
      <p v-if="description" class="tl-timeline-card__desc">{{ description }}</p>
      <div class="tl-timeline-card__meta">
        <span>{{ range }}</span>
        <span>{{ events }} eventos</span>
        <span v-if="contributors !== undefined">{{ contributors }} contribuidores</span>
        <span v-if="views !== undefined">{{ views.toLocaleString('pt-BR') }} {{ views === 1 ? 'visualização' : 'visualizações' }}</span>
      </div>
      <div v-if="author || favorites !== undefined" class="tl-timeline-card__author">
        <template v-if="author">
          <TlAvatar :name="author" size="xs" decorative />
          <span>{{ author }}</span>
        </template>
      </div>
    </TlCard>
    <TlFavoriteButton
      v-if="favorites !== undefined"
      class="tl-timeline-card__fav"
      size="sm"
      :count="favorites"
      :on="favorited"
      :disabled="favoriteBusy"
      :subject="title"
      @toggle="$emit('favorite')"
    />
  </div>
</template>

<style scoped>
.tl-timeline-card {
  position: relative;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.tl-timeline-card__body {
  flex: 1 1 auto;
}
.tl-timeline-card__fav {
  position: absolute;
  right: var(--tl-space-5);
  bottom: var(--tl-space-5);
}
.tl-timeline-card__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--tl-space-2);
}
.tl-timeline-card__title {
  flex: 1 1 auto;
  font-size: var(--tl-text-xl);
  line-height: var(--tl-leading-snug);
}
.tl-timeline-card__desc {
  margin-top: -6px;
  font-size: var(--tl-text-md);
  line-height: 1.5;
  color: var(--tl-color-text-muted);
}
.tl-timeline-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tl-timeline-card__author {
  display: flex;
  align-items: center;
  gap: var(--tl-space-2);
  padding-top: var(--tl-space-3);
  border-top: 1px solid var(--tl-color-border-subtle);
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
/* Rodapé com a altura do botão e espaço à direita para ele não cobrir o autor. */
.tl-timeline-card--fav .tl-timeline-card__author {
  min-height: 30px;
  margin-top: auto;
  padding-right: 88px;
}
</style>
