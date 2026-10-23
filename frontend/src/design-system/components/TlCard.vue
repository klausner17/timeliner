<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'

const props = withDefaults(
  defineProps<{
    as?: string
    to?: RouteLocationRaw
    href?: string
    padding?: 'sm' | 'md' | 'lg'
    /** `default` branco, `muted` cinza claro, `inverse` escuro, `dashed` para "adicionar". */
    tone?: 'default' | 'muted' | 'inverse' | 'dashed'
    /** Borda azul de destaque (formulário aberto, evento selecionado). */
    highlight?: boolean
  }>(),
  { as: 'div', padding: 'md', tone: 'default' },
)

const tag = computed(() => (props.to ? RouterLink : props.href ? 'a' : props.as))
// Só passa `to`/`href` quando usados: um href undefined anula o do RouterLink.
const linkAttrs = computed(() => (props.to ? { to: props.to } : props.href ? { href: props.href } : {}))
const interactive = computed(() => !!(props.to || props.href))
</script>

<template>
  <component
    :is="tag"
    class="tl-card"
    :class="[`tl-card--${padding}`, `tl-card--${tone}`, { 'tl-card--link': interactive, 'tl-card--highlight': highlight }]"
    v-bind="linkAttrs"
  >
    <slot />
  </component>
</template>

<style scoped>
.tl-card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-xl);
  color: var(--tl-color-text);
  text-decoration: none;
}
.tl-card--sm {
  padding: var(--tl-space-4);
  border-radius: var(--tl-radius-lg);
}
.tl-card--md {
  padding: var(--tl-space-5);
}
.tl-card--lg {
  padding: var(--tl-space-7);
  border-radius: 18px;
}
.tl-card--muted {
  background: var(--tl-color-surface-muted);
}
.tl-card--inverse {
  background: var(--tl-color-surface-inverse);
  border-color: var(--tl-color-surface-inverse);
  color: var(--tl-color-text-inverse);
}
.tl-card--dashed {
  align-items: center;
  justify-content: center;
  background: transparent;
  border: 1.5px dashed var(--tl-color-border-strong);
  color: var(--tl-color-text-body);
  font-weight: var(--tl-weight-semibold);
}
.tl-card--highlight {
  border: 1.5px solid var(--tl-color-primary);
}
.tl-card--link {
  transition:
    border-color var(--tl-duration-fast) var(--tl-ease),
    box-shadow var(--tl-duration-fast) var(--tl-ease);
}
.tl-card--link:hover {
  color: inherit;
  border-color: var(--tl-color-border-strong);
  box-shadow: 0 4px 16px rgb(21 23 28 / 0.06);
}
</style>
