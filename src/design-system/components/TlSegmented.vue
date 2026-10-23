<script setup lang="ts">
withDefaults(defineProps<{ label: string; options: { value: string; label: string }[]; size?: 'sm' | 'md' }>(), {
  size: 'md',
})

const model = defineModel<string>()
</script>

<template>
  <div class="tl-segmented" :class="`tl-segmented--${size}`" role="group" :aria-label="label">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      class="tl-segmented__item"
      :class="{ 'tl-segmented__item--on': model === o.value }"
      :aria-pressed="model === o.value"
      @click="model = o.value"
    >
      {{ o.label }}
    </button>
  </div>
</template>

<style scoped>
.tl-segmented {
  display: inline-flex;
  flex-wrap: wrap;
  gap: var(--tl-space-1);
  padding: var(--tl-space-1);
  background: var(--tl-color-surface-sunken);
  border-radius: var(--tl-radius-pill);
}
.tl-segmented__item {
  min-height: 40px;
  padding: 0 16px;
  border: 0;
  border-radius: var(--tl-radius-pill);
  background: transparent;
  color: var(--tl-color-text-muted);
  font-size: var(--tl-text-md);
  cursor: pointer;
}
.tl-segmented--sm .tl-segmented__item {
  min-height: var(--tl-control-sm);
  padding: 0 14px;
}
.tl-segmented__item:hover {
  color: var(--tl-color-text);
}
.tl-segmented__item--on {
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  font-weight: var(--tl-weight-semibold);
  box-shadow: 0 1px 2px rgb(21 23 28 / 0.06);
}
</style>
