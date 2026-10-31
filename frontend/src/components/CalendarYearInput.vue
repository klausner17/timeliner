<script setup lang="ts">
/*
 * Ano num calendário ficcional: a era e o ano dentro dela ("Terceira Era · 3019").
 * O v-model é o ano contínuo como texto, igual a um campo de ano comum: vazio sem ano e
 * "?" quando o ano não cabe na era, para o formulário recusar.
 */
import { computed, ref, watch } from 'vue'
import { TlInput, TlSelect } from '@/design-system'
import { fromEraYear, toEraYear } from '@/lib/calendar'
import type { TimelineCalendar } from '@/data/types'

const props = defineProps<{ calendar: TimelineCalendar; id?: string; invalid?: boolean; describedby?: string; placeholder?: string }>()
const model = defineModel<string>({ required: true })

const eraId = ref(props.calendar.eras[0]?.id ?? '')
const year = ref('')
const eraOptions = computed(() => props.calendar.eras.map((e) => ({ value: e.id, label: `${e.name} (${e.abbr})` })))
const era = computed(() => props.calendar.eras.find((e) => e.id === eraId.value))
const tooLong = computed(() => model.value === '?' && era.value?.years !== undefined)

// De fora para dentro: o ano contínuo vira era + ano.
watch(
  model,
  (v) => {
    if (v === '?') return
    const parts = v.trim() ? toEraYear(Number(v), props.calendar) : undefined
    if (parts) [eraId.value, year.value] = [parts.era.id, String(parts.year)]
    else if (!v.trim()) year.value = ''
  },
  { immediate: true },
)

// De dentro para fora.
watch([eraId, year], ([id, y]) => {
  if (!y.trim()) return void (model.value = '')
  const abs = fromEraYear(id, Number(y), props.calendar)
  model.value = abs === undefined ? '?' : String(abs)
})
</script>

<template>
  <div class="era-year">
    <TlSelect v-model="eraId" :options="eraOptions" :aria-label="'Era'" :invalid="invalid" />
    <TlInput :id="id" v-model="year" inputmode="numeric" :placeholder="placeholder ?? 'Ano'" :aria-describedby="describedby" :invalid="invalid" />
    <span v-if="tooLong" class="era-year__hint" role="status">A {{ era?.name }} tem {{ era?.years }} anos.</span>
  </div>
</template>

<style scoped>
.era-year {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr);
  gap: var(--tl-space-2);
}
.era-year__hint {
  grid-column: 1 / -1;
  color: var(--tl-color-warning-strong);
  font-size: var(--tl-text-sm);
}
</style>
