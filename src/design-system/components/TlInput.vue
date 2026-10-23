<script setup lang="ts">
withDefaults(defineProps<{ size?: 'md' | 'lg'; shape?: 'rounded' | 'pill'; invalid?: boolean }>(), {
  size: 'md',
  shape: 'rounded',
})

const model = defineModel<string | number>()
</script>

<template>
  <input
    v-model="model"
    class="tl-input"
    :class="[`tl-input--${size}`, `tl-input--${shape}`]"
    :aria-invalid="invalid ? 'true' : undefined"
  />
</template>

<style scoped>
.tl-input {
  width: 100%;
  min-height: 46px;
  padding: 0 14px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  font-size: var(--tl-text-base);
}
.tl-input::placeholder {
  color: var(--tl-color-text-muted);
}
.tl-input:hover {
  border-color: var(--tl-color-text-subtle);
}
.tl-input:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 0;
  border-color: var(--tl-color-primary);
}
.tl-input--lg {
  min-height: var(--tl-control-lg);
  padding: 0 20px;
  font-size: 16px;
}
.tl-input--pill {
  border-radius: var(--tl-radius-pill);
}
.tl-input[aria-invalid='true'] {
  border-color: var(--tl-color-warning);
}
</style>
