<script setup lang="ts">
import { useId } from 'vue'

defineProps<{ legend: string; options: { value: string; label: string }[] }>()

const model = defineModel<string>()
const name = useId()
</script>

<template>
  <fieldset class="tl-radio-group">
    <legend class="tl-radio-group__legend">{{ legend }}</legend>
    <label v-for="o in options" :key="o.value" class="tl-radio">
      <input v-model="model" class="tl-radio__input" type="radio" :name="name" :value="o.value" />
      <span>{{ o.label }}</span>
    </label>
  </fieldset>
</template>

<style scoped>
.tl-radio-group {
  margin: 0;
  padding: 0;
  border: 0;
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-1);
  min-width: 0;
}
.tl-radio-group__legend {
  padding: 0;
  margin-bottom: 6px;
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
}
.tl-radio {
  display: flex;
  align-items: start;
  gap: 10px;
  min-height: 40px;
  font-size: var(--tl-text-md);
  line-height: 1.4;
  cursor: pointer;
}
.tl-radio__input {
  flex: none;
  width: 18px;
  height: 18px;
  margin: 1px 0 0;
  accent-color: var(--tl-color-primary);
}
</style>
