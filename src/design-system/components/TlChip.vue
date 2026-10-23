<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'

/**
 * `filter`: botão liga/desliga (filtro por período).
 * `tag`: rótulo estático ou link de categoria.
 */
const props = withDefaults(
  defineProps<{ kind?: 'filter' | 'tag'; pressed?: boolean; to?: RouteLocationRaw; href?: string; size?: 'sm' | 'md' }>(),
  { kind: 'filter', size: 'md' },
)

const tag = computed(() => {
  if (props.to) return RouterLink
  if (props.href) return 'a'
  return props.kind === 'filter' ? 'button' : 'span'
})
// Só passa `to`/`href` quando usados: um href undefined anula o do RouterLink.
const linkAttrs = computed(() => (props.to ? { to: props.to } : props.href ? { href: props.href } : {}))
</script>

<template>
  <component
    :is="tag"
    class="tl-chip"
    :class="[`tl-chip--${kind}`, `tl-chip--${size}`, { 'tl-chip--on': pressed }]"
    v-bind="linkAttrs"
    :type="tag === 'button' ? 'button' : undefined"
    :aria-pressed="tag === 'button' ? !!pressed : undefined"
  >
    <slot />
  </component>
</template>

<style scoped>
.tl-chip {
  display: inline-flex;
  align-items: center;
  border-radius: var(--tl-radius-pill);
  font-size: var(--tl-text-md);
  text-decoration: none;
  white-space: nowrap;
  color: var(--tl-color-text-strong);
}
.tl-chip--filter {
  min-height: 40px;
  padding: 0 14px;
  border: 1px solid var(--tl-color-border-strong);
  background: var(--tl-color-surface);
  font-weight: var(--tl-weight-medium);
  cursor: pointer;
}
.tl-chip--filter:hover {
  border-color: var(--tl-color-text-subtle);
}
.tl-chip--filter.tl-chip--on {
  background: var(--tl-color-surface-inverse);
  border-color: var(--tl-color-surface-inverse);
  color: var(--tl-color-text-inverse);
}
.tl-chip--tag {
  padding: 10px 14px;
  border: 0;
  background: var(--tl-color-neutral-soft);
}
.tl-chip--tag.tl-chip--sm {
  padding: 6px 10px;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text);
}
a.tl-chip--tag:hover {
  background: var(--tl-color-surface-sunken);
  color: var(--tl-color-text);
}
</style>
