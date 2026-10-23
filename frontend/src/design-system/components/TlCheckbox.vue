<script setup lang="ts">
defineProps<{ label: string }>()

const model = defineModel<boolean>({ default: false })
</script>

<template>
  <label class="tl-check">
    <input v-model="model" class="tl-check__input" type="checkbox" />
    <span>{{ label }}</span>
  </label>
</template>

<style scoped>
.tl-check {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 36px;
  font-size: var(--tl-text-md);
  color: var(--tl-color-text-body);
  cursor: pointer;
}
.tl-check__input {
  width: 18px;
  height: 18px;
  margin: 0;
  accent-color: var(--tl-color-primary);
}
</style>
