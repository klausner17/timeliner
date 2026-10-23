<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import TlIcon from './TlIcon.vue'
import type { IconName } from './icons'

export type ButtonVariant = 'primary' | 'dark' | 'secondary' | 'ghost' | 'success' | 'inverse' | 'dashed'
export type ButtonSize = 'sm' | 'md' | 'lg'

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariant
    size?: ButtonSize
    /** `pill` nas ações de página, `rounded` dentro de formulários. */
    shape?: 'pill' | 'rounded'
    icon?: IconName
    /** Só ícone: `label` vira o aria-label. */
    iconOnly?: boolean
    label?: string
    href?: string
    to?: RouteLocationRaw
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    block?: boolean
  }>(),
  { variant: 'secondary', size: 'md', shape: 'pill', type: 'button' },
)

const tag = computed(() => (props.to ? RouterLink : props.href ? 'a' : 'button'))
// Só passa `to`/`href` quando usados: um href undefined anula o do RouterLink.
const linkAttrs = computed(() => (props.to ? { to: props.to } : props.href ? { href: props.href } : {}))
const iconSize = computed(() => (props.iconOnly ? (props.size === 'lg' ? 20 : 18) : 16))
</script>

<template>
  <component
    :is="tag"
    class="tl-button"
    :class="[`tl-button--${variant}`, `tl-button--${size}`, `tl-button--${shape}`, { 'tl-button--icon-only': iconOnly, 'tl-button--block': block }]"
    v-bind="linkAttrs"
    :type="tag === 'button' ? type : undefined"
    :disabled="tag === 'button' ? disabled : undefined"
    :aria-disabled="tag !== 'button' && disabled ? 'true' : undefined"
    :aria-label="iconOnly ? label : undefined"
  >
    <TlIcon v-if="icon" :name="icon" :size="iconSize" />
    <template v-if="!iconOnly"><slot>{{ label }}</slot></template>
  </component>
</template>

<style scoped>
.tl-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--tl-space-2);
  min-height: var(--tl-control-md);
  padding: 0 18px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--tl-color-text);
  font-family: var(--tl-font-body);
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-base);
  line-height: 1;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color var(--tl-duration-fast) var(--tl-ease),
    border-color var(--tl-duration-fast) var(--tl-ease);
}
.tl-button:hover {
  color: inherit;
}
.tl-button--pill {
  border-radius: var(--tl-radius-pill);
}
.tl-button--rounded {
  border-radius: var(--tl-radius-md);
}
.tl-button--sm {
  min-height: var(--tl-control-sm);
  padding: 0 14px;
  font-size: var(--tl-text-md);
}
.tl-button--lg {
  min-height: var(--tl-control-lg);
  padding: 0 24px;
  font-size: 16px;
}
.tl-button--icon-only {
  width: var(--tl-control-md);
  padding: 0;
}
.tl-button--icon-only.tl-button--lg {
  width: var(--tl-control-lg);
}
.tl-button--block {
  display: flex;
  width: 100%;
}

.tl-button--primary {
  background: var(--tl-color-primary);
  color: var(--tl-color-text-inverse);
}
.tl-button--primary:hover {
  background: var(--tl-color-primary-hover);
  color: var(--tl-color-text-inverse);
}
.tl-button--dark {
  background: var(--tl-color-surface-inverse);
  color: var(--tl-color-text-inverse);
}
.tl-button--dark:hover {
  background: var(--tl-color-text-strong);
  color: var(--tl-color-text-inverse);
}
.tl-button--secondary {
  background: var(--tl-color-surface);
  border-color: var(--tl-color-border-strong);
}
.tl-button--secondary:hover {
  background: var(--tl-color-surface-muted);
  border-color: var(--tl-color-text-subtle);
}
.tl-button--ghost:hover {
  background: var(--tl-color-neutral-soft);
}
.tl-button--success {
  background: var(--tl-color-success);
  color: var(--tl-color-text-inverse);
}
.tl-button--success:hover {
  background: var(--tl-color-success-strong);
  color: var(--tl-color-text-inverse);
}
.tl-button--inverse {
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
}
.tl-button--dashed {
  border: 1.5px dashed var(--tl-color-border-strong);
  color: var(--tl-color-text-body);
}
.tl-button--dashed:hover {
  border-color: var(--tl-color-text-subtle);
}

.tl-button:disabled,
.tl-button[aria-disabled='true'] {
  opacity: 0.5;
  cursor: not-allowed;
  pointer-events: none;
}
</style>
