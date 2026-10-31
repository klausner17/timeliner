<script setup lang="ts" generic="C extends string">
import { periodColorNames, trackColorNames, trackColors } from '../tokens'

/**
 * Bolinhas de cor para escolher uma. `palette` diz de onde vêm as cores: trilhas de
 * timeline (a, b, c, d) ou faixas de período (azul, laranja...). Com `none`, aparece uma
 * opção a mais, "sem cor", que deixa o modelo vazio.
 */
const props = withDefaults(
  defineProps<{
    label?: string
    colors?: readonly C[]
    palette?: 'track' | 'period'
    /** Rótulo da opção "sem cor" (ex.: "Automática"). */
    none?: string
  }>(),
  { label: 'Cor da timeline', palette: 'track' },
)

const model = defineModel<C | undefined>()

const list = (): readonly C[] => props.colors ?? (trackColors as readonly string[] as readonly C[])
const names: Record<string, string> = { ...trackColorNames, ...periodColorNames }
const swatch = (c: C) => (props.palette === 'period' ? `var(--tl-period-${c})` : `var(--tl-track-${c})`)
</script>

<template>
  <div class="tl-color-picker" role="radiogroup" :aria-label="label">
    <button
      v-if="none"
      type="button"
      role="radio"
      class="tl-color-picker__swatch tl-color-picker__swatch--none"
      :class="{ 'tl-color-picker__swatch--on': !model }"
      :aria-checked="!model"
      :aria-label="none"
      :title="none"
      @click="model = undefined"
    ></button>
    <button
      v-for="c in list()"
      :key="c"
      type="button"
      role="radio"
      class="tl-color-picker__swatch"
      :class="{ 'tl-color-picker__swatch--on': model === c }"
      :style="{ background: swatch(c) }"
      :aria-checked="model === c"
      :aria-label="names[c] ?? c"
      :title="names[c] ?? c"
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
.tl-color-picker__swatch--none {
  background: repeating-linear-gradient(-45deg, var(--tl-color-surface-muted) 0 5px, var(--tl-color-surface) 5px 10px);
  box-shadow: 0 0 0 1px var(--tl-color-border-strong) inset;
}
.tl-color-picker__swatch--on {
  border-color: var(--tl-color-text);
}
</style>
