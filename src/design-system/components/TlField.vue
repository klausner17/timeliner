<script setup lang="ts">
import { useId } from 'vue'

withDefaults(
  defineProps<{
    label: string
    hint?: string
    error?: string
    /** `strong` = rótulo em negrito (formulários), `muted` = rótulo cinza (subcampos como Dia/Mês/Ano). */
    tone?: 'strong' | 'muted'
  }>(),
  { tone: 'strong' },
)

const id = useId()
const hintId = `${id}-hint`
</script>

<template>
  <div class="tl-field">
    <label class="tl-field__label" :class="`tl-field__label--${tone}`" :for="id">{{ label }}</label>
    <slot :id="id" :describedby="hint || error ? hintId : undefined" :invalid="!!error" />
    <span v-if="error" :id="hintId" class="tl-field__hint tl-field__hint--error">{{ error }}</span>
    <span v-else-if="hint" :id="hintId" class="tl-field__hint">{{ hint }}</span>
  </div>
</template>

<style scoped>
.tl-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.tl-field__label {
  font-size: var(--tl-text-md);
}
.tl-field__label--strong {
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text);
}
.tl-field__label--muted {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tl-field__hint {
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
.tl-field__hint--error {
  color: var(--tl-color-warning-strong);
}
</style>
