<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import type { TrackColor } from '../tokens'
import TlAvatar from './TlAvatar.vue'
import TlBadge, { type BadgeTone } from './TlBadge.vue'
import TlCard from './TlCard.vue'
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
  author?: string
  badge?: { label: string; tone: BadgeTone }
  to?: RouteLocationRaw
  href?: string
}>()
</script>

<template>
  <TlCard :to="to" :href="href" class="tl-timeline-card">
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
    </div>
    <div v-if="author" class="tl-timeline-card__author">
      <TlAvatar :name="author" size="xs" decorative />
      <span>{{ author }}</span>
    </div>
  </TlCard>
</template>

<style scoped>
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
</style>
