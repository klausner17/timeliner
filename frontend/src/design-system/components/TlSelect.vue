<script setup lang="ts">
export type SelectOption = string | { value: string; label: string }

defineProps<{ options: SelectOption[]; invalid?: boolean }>()

const model = defineModel<string>()

const valueOf = (o: SelectOption) => (typeof o === 'string' ? o : o.value)
const labelOf = (o: SelectOption) => (typeof o === 'string' ? o : o.label)
</script>

<template>
  <select v-model="model" class="tl-select" :aria-invalid="invalid ? 'true' : undefined">
    <option v-for="o in options" :key="valueOf(o)" :value="valueOf(o)">{{ labelOf(o) }}</option>
  </select>
</template>

<style scoped>
.tl-select {
  width: 100%;
  min-height: 46px;
  padding: 0 36px 0 12px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface)
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='8' viewBox='0 0 12 8' fill='none' stroke='%234E5561' stroke-width='1.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M1 1.5l5 5 5-5'/%3E%3C/svg%3E")
    no-repeat right 14px center;
  color: var(--tl-color-text);
  font-size: var(--tl-text-base);
  appearance: none;
  cursor: pointer;
}
.tl-select:hover {
  border-color: var(--tl-color-text-subtle);
}
.tl-select:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 0;
  border-color: var(--tl-color-primary);
}
</style>
