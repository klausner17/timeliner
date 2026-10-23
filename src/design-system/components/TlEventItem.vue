<script setup lang="ts">
import type { TrackColor } from '../tokens'

/** Um evento na timeline vertical: ano, trilho com ponto e cartão. */
withDefaults(
  defineProps<{
    year: string | number
    title: string
    date?: string
    description?: string
    by?: string
    sourceUrl?: string
    color?: TrackColor
    /** Sugestão ainda não aprovada: ponto e borda tracejados. */
    suggestion?: boolean
    compact?: boolean
  }>(),
  { color: 'a' },
)
</script>

<template>
  <article
    class="tl-event"
    :class="{ 'tl-event--suggestion': suggestion, 'tl-event--compact': compact }"
    :style="{ '--track': `var(--tl-track-${color})` }"
  >
    <div class="tl-event__year">{{ year }}</div>
    <div class="tl-event__rail" aria-hidden="true"><span class="tl-event__dot"></span></div>
    <div class="tl-event__card">
      <div v-if="date || suggestion" class="tl-event__date">
        {{ date }}<template v-if="suggestion">{{ date ? ' · ' : '' }}sugestão</template>
      </div>
      <h3 class="tl-event__title">{{ title }}</h3>
      <p v-if="description" class="tl-event__desc">{{ description }}</p>
      <div v-if="sourceUrl || by || $slots.actions" class="tl-event__meta">
        <a v-if="sourceUrl" :href="sourceUrl" target="_blank" rel="noopener">Fonte</a>
        <span v-if="by">adicionado por {{ by }}</span>
        <slot name="actions" />
      </div>
    </div>
  </article>
</template>

<style scoped>
.tl-event {
  display: grid;
  grid-template-columns: 96px 28px minmax(0, 1fr);
  column-gap: var(--tl-space-2);
}
.tl-event--compact {
  grid-template-columns: 56px 24px minmax(0, 1fr);
  column-gap: 6px;
}
.tl-event__year {
  padding-top: 18px;
  text-align: right;
  font-family: var(--tl-font-display);
  font-weight: var(--tl-weight-semibold);
  font-size: 18px;
}
.tl-event--compact .tl-event__year {
  font-size: 16px;
}
.tl-event__rail {
  position: relative;
  display: flex;
  justify-content: center;
}
.tl-event__rail::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  width: 2px;
  background: var(--tl-color-rail);
}
.tl-event__dot {
  position: relative;
  margin-top: 22px;
  width: 14px;
  height: 14px;
  border-radius: var(--tl-radius-pill);
  background: var(--track);
  box-shadow: 0 0 0 4px var(--tl-color-bg);
}
.tl-event__card {
  margin: 6px 0 10px;
  padding: var(--tl-space-4) 18px;
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-lg);
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
  min-width: 0;
}
.tl-event--suggestion .tl-event__dot {
  background: var(--tl-color-surface);
  border: 2px dashed var(--track);
  box-shadow: none;
}
.tl-event--suggestion .tl-event__card {
  border: 1.5px dashed var(--track);
}
.tl-event__date {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tl-event__title {
  font-family: var(--tl-font-body);
  font-size: var(--tl-text-lg);
  font-weight: var(--tl-weight-semibold);
  letter-spacing: 0;
  line-height: var(--tl-leading-snug);
}
.tl-event--compact .tl-event__title {
  font-size: var(--tl-text-base);
}
.tl-event__desc {
  font-size: var(--tl-text-base);
  line-height: var(--tl-leading-normal);
  color: var(--tl-color-text-body);
}
.tl-event__meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 16px;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}

@media (max-width: 560px) {
  .tl-event {
    grid-template-columns: 52px 22px minmax(0, 1fr);
    column-gap: 6px;
  }
  .tl-event__year {
    font-size: 15px;
  }
}
</style>
