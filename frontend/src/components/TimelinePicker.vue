<script setup lang="ts">
/*
 * Seletor de timeline com busca no servidor: em vez de um <select> com todas, mostra
 * a escolhida e, ao abrir, uma busca paginada (20 por vez, "Carregar mais"). Filtra
 * pelo calendário quando a trilha precisa casar com a primeira.
 */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { currentUser, listTimelines } from '@/data/store'
import type { Timeline } from '@/data/types'
import { scaleName } from '@/lib/scale'
import { usePaged } from '@/lib/usePaged'

const props = defineProps<{
  id: string
  /** A timeline escolhida agora. */
  value?: Timeline
  /** Só timelines deste calendário (ver calendarKey); vazio aceita qualquer uma. */
  calendar?: string
  /** Timelines já em outras trilhas: aparecem marcadas. */
  taken?: string[]
}>()
const emit = defineEmits<{ pick: [t: Timeline] }>()

const PAGE = 20

const open = ref(false)
const search = ref('')
const query = ref('')
let debounce: ReturnType<typeof setTimeout> | undefined
watch(search, (s) => {
  clearTimeout(debounce)
  debounce = setTimeout(() => (query.value = s.trim()), 250)
})
onBeforeUnmount(() => clearTimeout(debounce))

// Só busca com o painel aberto, e recomeça quando a busca ou o calendário mudam.
const { items, total, loading, error, hasMore, loadMore } = usePaged(
  () => (open.value ? { q: query.value, calendar: props.calendar, mine: !!currentUser.value } : undefined),
  (k, page) =>
    listTimelines({ q: k.q || undefined, calendar: k.calendar || undefined, includeMine: k.mine || undefined, withEvents: true, page, pageSize: PAGE }),
  (t) => t.slug,
)
const results = computed(() => items.value ?? [])

const detail = (t: Timeline) => (t.calendar ? t.calendar.name : t.scale ? `fixa, em ${scaleName(t.scale)}` : '')
const takenIndex = (t: Timeline) => props.taken?.indexOf(t.slug) ?? -1

const root = ref<HTMLElement>()
const input = ref<HTMLInputElement>()
const active = ref(-1)
watch(results, () => (active.value = results.value.length ? Math.max(0, Math.min(active.value, results.value.length - 1)) : -1))

async function show() {
  open.value = true
  active.value = 0
  await nextTick()
  input.value?.focus()
}
function close(refocus = true) {
  open.value = false
  search.value = ''
  query.value = ''
  if (refocus) root.value?.querySelector<HTMLButtonElement>('.tp__button')?.focus()
}
function choose(t: Timeline) {
  close()
  if (t.slug !== props.value?.slug) emit('pick', t)
}
function onKey(e: KeyboardEvent) {
  const n = results.value.length
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (n) active.value = (active.value + 1) % n
    if (active.value === n - 1 && hasMore.value) void loadMore()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (n) active.value = (active.value - 1 + n) % n
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const t = results.value[active.value]
    if (t) choose(t)
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}
watch(active, async (i) => {
  await nextTick()
  root.value?.querySelector(`#${props.id}-opt-${i}`)?.scrollIntoView({ block: 'nearest' })
})

// Clicar fora fecha.
function onDocPointer(e: PointerEvent) {
  if (open.value && root.value && !root.value.contains(e.target as Node)) close(false)
}
watch(open, (o) => {
  if (o) document.addEventListener('pointerdown', onDocPointer)
  else document.removeEventListener('pointerdown', onDocPointer)
})
onBeforeUnmount(() => document.removeEventListener('pointerdown', onDocPointer))

// Rolar até o fim da lista carrega a página seguinte.
function onScroll(e: Event) {
  const el = e.target as HTMLElement
  if (hasMore.value && !loading.value && el.scrollTop + el.clientHeight >= el.scrollHeight - 40) void loadMore()
}
</script>

<template>
  <div ref="root" class="tp">
    <button
      :id="id"
      type="button"
      class="tp__button"
      aria-haspopup="listbox"
      :aria-expanded="open ? 'true' : 'false'"
      @click="open ? close() : show()"
      @keydown.down.prevent="show"
    >
      <span class="tp__title">{{ value?.title ?? 'Escolher timeline' }}</span>
      <span v-if="value && detail(value)" class="tp__detail">{{ detail(value) }}</span>
    </button>

    <div v-if="open" class="tp__panel">
      <input
        ref="input"
        v-model="search"
        type="search"
        class="tp__search"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded="true"
        :aria-controls="`${id}-list`"
        :aria-activedescendant="active >= 0 ? `${id}-opt-${active}` : undefined"
        aria-label="Buscar timeline"
        placeholder="Buscar pelo título, descrição ou tag…"
        @keydown="onKey"
      />
      <ul :id="`${id}-list`" role="listbox" class="tp__list" :aria-busy="loading ? 'true' : 'false'" @scroll="onScroll">
        <li
          v-for="(t, i) in results"
          :id="`${id}-opt-${i}`"
          :key="t.slug"
          role="option"
          class="tp__opt"
          :class="{ 'tp__opt--active': i === active }"
          :aria-selected="t.slug === value?.slug ? 'true' : 'false'"
          @pointerdown.prevent
          @click="choose(t)"
          @pointermove="active = i"
        >
          <span class="tp__opt-title">{{ t.title }}</span>
          <span class="tp__opt-meta">
            <template v-if="detail(t)">{{ detail(t) }} · </template>@{{ t.owner.handle }} · {{ t.eventCount }} {{ t.eventCount === 1 ? 'evento' : 'eventos' }}
            <template v-if="t.slug === value?.slug"> · escolhida</template>
            <template v-else-if="takenIndex(t) >= 0"> · já na comparação (troca de lugar)</template>
          </span>
        </li>
      </ul>
      <p v-if="error" class="tp__note" role="alert">{{ error }}</p>
      <p v-else-if="items && !results.length && !loading" class="tp__note">Nenhuma timeline encontrada{{ calendar ? ' neste calendário' : '' }}.</p>
      <p v-else-if="loading && !results.length" class="tp__note">Buscando…</p>
      <div v-if="hasMore" class="tp__foot">
        <span>{{ results.length }} de {{ total }}</span>
        <button type="button" class="tp__more" :disabled="loading" @click="loadMore">{{ loading ? 'Carregando…' : 'Carregar mais' }}</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tp {
  position: relative;
  flex: 1;
  min-width: 0;
}
.tp__button {
  width: 100%;
  min-height: 44px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  padding: 4px 32px 4px 0;
  border: 0;
  background: transparent
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='8' viewBox='0 0 12 8' fill='none' stroke='%234E5561' stroke-width='1.8' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M1 1.5l5 5 5-5'/%3E%3C/svg%3E")
    no-repeat right 8px center;
  color: var(--tl-color-text);
  text-align: left;
  cursor: pointer;
}
.tp__button:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 2px;
  border-radius: var(--tl-radius-sm, 6px);
}
.tp__title {
  max-width: 100%;
  font-size: 16px;
  font-weight: var(--tl-weight-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tp__detail {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tp__panel {
  position: absolute;
  z-index: 30;
  top: calc(100% + 10px);
  /* Alinha com a caixa da trilha (bolinha e espaço à esquerda, botão de remover à direita). */
  left: -38px;
  right: -8px;
  min-width: min(340px, calc(100vw - 32px));
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
  padding: var(--tl-space-2);
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border-strong);
  border-radius: 12px;
  box-shadow: 0 8px 28px rgb(21 23 28 / 0.14);
}
.tp__search {
  width: 100%;
  min-height: 42px;
  padding: 0 12px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  font-size: 16px;
}
.tp__search:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 0;
  border-color: var(--tl-color-primary);
}
.tp__list {
  max-height: min(320px, 50vh);
  overflow-y: auto;
  margin: 0;
  padding: 0;
  list-style: none;
}
.tp__list:empty {
  display: none;
}
.tp__opt {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
  border-radius: var(--tl-radius-md);
  cursor: pointer;
}
.tp__opt--active {
  background: var(--tl-color-surface-muted, rgb(21 23 28 / 0.05));
}
.tp__opt[aria-selected='true'] .tp__opt-title {
  color: var(--tl-color-primary);
}
.tp__opt-title {
  font-weight: var(--tl-weight-semibold);
}
.tp__opt-meta,
.tp__note,
.tp__foot {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.tp__note {
  padding: 6px 10px;
}
.tp__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--tl-space-2);
  padding: 2px 4px 0 10px;
}
.tp__more {
  min-height: 36px;
  padding: 0 12px;
  border: 1px solid var(--tl-color-border-strong);
  border-radius: var(--tl-radius-md);
  background: var(--tl-color-surface);
  color: var(--tl-color-text);
  font-weight: var(--tl-weight-semibold);
  cursor: pointer;
}
</style>
