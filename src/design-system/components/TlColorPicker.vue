<script setup lang="ts">
import { trackColorNames, trackColors, type TrackColor } from '../tokens'

withDefaults(defineProps<{ label?: string; colors?: readonly TrackColor[] }>(), {
  label: 'Cor da timeline',
  colors: () => trackColors,
})

const model = defineModel<TrackColor>()
</script>

<template>
  <div class="tl-color-picker" role="radiogroup" :aria-label="label">
    <button
      v-for="c in colors"
      :key="c"
      type="button"
      role="radio"
      class="tl-color-picker__swatch"
      :class="{ 'tl-color-picker__swatch--on': model === c }"
      :style="{ background: `var(--tl-track-${c})` }"
      :aria-checked="model === c"
      :aria-label="trackColorNames[c]"
      @click="model = c"
    ></button>
  </div>
</template>

<style scoped>
.tl-color-picker {
  display: flex;
  flex-wrap: wrap;
  gap: var(--tl-space-2);
}
.tl-color-picker__swatch {
  width: 36px;
  height: 36px;
  border-radius: var(--tl-radius-pill);
  border: 3px solid transparent;
  box-shadow: 0 0 0 2px var(--tl-color-surface) inset;
  cursor: pointer;
}
.tl-color-picker__swatch--on {
  border-color: var(--tl-color-text);
}
</style>
