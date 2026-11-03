<script setup lang="ts">
import { computed } from "vue";
import TlIcon from "./TlIcon.vue";

/**
 * Estrela com o total de favoritos. `on` é se quem está vendo já deu estrela. O clique só
 * avisa (`toggle`): quem usa decide se precisa de login e atualiza `on` e `count`.
 */
const props = withDefaults(
  defineProps<{
    count: number;
    on?: boolean;
    /** `sm` nos cartões, `md` no topo da timeline. */
    size?: "sm" | "md";
    disabled?: boolean;
    /** Nome da timeline, para o leitor de tela dizer o que está favoritando. */
    subject?: string;
  }>(),
  { size: "md" },
);
defineEmits<{ toggle: [] }>();

const label = computed(() => {
  const what = props.subject ? ` ${props.subject}` : "";
  const total = `${props.count} ${props.count === 1 ? "favorito" : "favoritos"}`;
  return props.on
    ? `Tirar${what} dos favoritos (${total})`
    : `Favoritar${what} (${total})`;
});
</script>

<template>
  <button
    type="button"
    class="tl-fav"
    :class="[`tl-fav--${size}`, { 'tl-fav--on': on }]"
    :aria-pressed="on ? 'true' : 'false'"
    :aria-label="label"
    :title="on ? 'Tirar dos favoritos' : 'Favoritar'"
    :disabled="disabled"
    @click.stop.prevent="$emit('toggle')"
  >
    <TlIcon
      :name="on ? 'star-filled' : 'star'"
      :size="size === 'sm' ? 15 : 17"
      class="tl-fav__icon"
    />
    <span class="tl-fav__count" aria-hidden="true">{{
      count.toLocaleString("pt-BR")
    }}</span>
  </button>
</template>

<style scoped>
.tl-fav {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: var(--tl-control-md);
  padding: 0 14px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-surface);
  color: var(--tl-color-text-body);
  font: inherit;
  font-size: var(--tl-text-md);
  font-weight: var(--tl-weight-semibold);
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition:
    border-color var(--tl-duration-fast) var(--tl-ease),
    background var(--tl-duration-fast) var(--tl-ease),
    color var(--tl-duration-fast) var(--tl-ease);
}
.tl-fav--sm {
  min-height: 30px;
  padding: 0 10px;
  font-size: var(--tl-text-sm);
}
.tl-fav:hover:not(:disabled) {
  border-color: var(--tl-color-favorite);
  color: var(--tl-color-text);
}
.tl-fav:hover:not(:disabled) .tl-fav__icon {
  color: var(--tl-color-favorite);
}
.tl-fav:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 2px;
}
.tl-fav:disabled {
  cursor: default;
  opacity: 0.6;
}
.tl-fav--on {
  border-color: var(--tl-color-favorite);
  background: var(--tl-color-favorite-soft);
  color: var(--tl-color-text);
}
.tl-fav--on .tl-fav__icon {
  color: var(--tl-color-favorite);
}
.tl-fav__icon {
  color: var(--tl-color-text-muted);
}
</style>
