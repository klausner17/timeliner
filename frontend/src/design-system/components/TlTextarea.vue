<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(defineProps<{ rows?: number; maxlength?: number; invalid?: boolean }>(), { rows: 4 })

const model = defineModel<string>({ default: '' })
const counter = computed(() => (props.maxlength ? `${model.value.length} de ${props.maxlength} caracteres` : ''))
</script>

<template>
  <div class="tl-textarea">
    <textarea
      v-bind="$attrs"
      v-model="model"
      class="tl-textarea__control"
      :rows="rows"
      :maxlength="maxlength"
      :aria-invalid="invalid ? 'true' : undefined"
    ></textarea>
    <span v-if="counter" class="tl-textarea__counter" aria-live="polite">{{ counter }}</span>
  </div>
</template>

<style scoped>
.tl-textarea {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.tl-textarea__control {
  width: 100%;
  padding: 12px 14px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  font-size: var(--tl-text-base);
  line-height: 1.5;
  resize: vertical;
}
.tl-textarea__control:hover {
  border-color: var(--tl-color-text-subtle);
}
.tl-textarea__control:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 0;
  border-color: var(--tl-color-primary);
}
.tl-textarea__control[aria-invalid='true'] {
  border-color: var(--tl-color-warning);
}
.tl-textarea__counter {
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
</style>
