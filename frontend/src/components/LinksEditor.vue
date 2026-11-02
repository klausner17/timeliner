<script setup lang="ts">
import { TlButton, TlInput } from '@/design-system'

/** Linhas de link (endereço e nome opcional) no formulário de evento. */
export type LinkRow = { url: string; label: string }

const props = withDefaults(defineProps<{ max?: number; errors?: Record<number, string> }>(), { max: 10, errors: () => ({}) })
const rows = defineModel<LinkRow[]>({ required: true })

const urlPlaceholder = 'https://'
const add = () => (rows.value = [...rows.value, { url: '', label: '' }])
const remove = (i: number) => (rows.value = rows.value.filter((_, j) => j !== i))
</script>

<template>
  <fieldset class="links">
    <legend class="links__legend">Mais links <span class="links__hint">(opcional)</span></legend>
    <p class="links__help">Matérias, documentos, vídeos. Quem abrir o evento vê a miniatura e o título de cada página.</p>
    <div v-for="(r, i) in rows" :key="i" class="links__row">
      <TlInput
        v-model="r.url"
        type="url"
        :placeholder="urlPlaceholder"
        :aria-label="`Endereço do link ${i + 1}`"
        :invalid="!!props.errors[i]"
        class="links__url"
      />
      <TlInput v-model="r.label" :aria-label="`Nome do link ${i + 1} (opcional)`" placeholder="Nome (opcional)" maxlength="120" class="links__label" />
      <TlButton icon-only icon="trash" variant="ghost" shape="rounded" :label="`Remover link ${i + 1}`" @click="remove(i)" />
      <p v-if="props.errors[i]" class="links__error" role="alert">{{ props.errors[i] }}</p>
    </div>
    <TlButton v-if="rows.length < max" size="sm" icon="plus" variant="ghost" @click="add">Adicionar link</TlButton>
  </fieldset>
</template>

<style scoped>
.links {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: var(--tl-space-2);
  margin: 0;
  padding: 0;
  border: 0;
  min-width: 0;
}
.links__legend {
  padding: 0;
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
}
.links__hint,
.links__help {
  font-weight: normal;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.links__row {
  width: 100%;
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(0, 2fr) auto;
  gap: var(--tl-space-2);
  align-items: center;
}
.links__error {
  grid-column: 1 / -1;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-warning-strong);
}
@media (max-width: 560px) {
  .links__row {
    grid-template-columns: minmax(0, 1fr) auto;
  }
  .links__label {
    grid-row: 2;
  }
}
</style>
