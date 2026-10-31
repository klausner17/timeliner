<script setup lang="ts">
/*
 * Ano num calendário ficcional: a era e o ano dentro dela ("Terceira Era · 3019").
 * O v-model é o ano contínuo como texto, igual a um campo de ano comum: vazio sem ano e
 * "?" quando o ano não cabe na era, para o formulário recusar.
 *
 * Numa timeline fixa (calendar é a escala, ex.: "dias"), é só a posição a partir do
 * início, com a unidade ao lado ("Dia [60]").
 */
import { computed, ref, watch } from 'vue'
import { TlInput, TlSelect } from '@/design-system'
import { fromEraYear, toEraYear } from '@/lib/calendar'
import { isScale, unitLabel, type Reckoning } from '@/lib/scale'

const props = defineProps<{ calendar: Reckoning; id?: string; invalid?: boolean; describedby?: string; placeholder?: string }>()
const model = defineModel<string>({ required: true })

const cal = computed(() => (isScale(props.calendar) ? undefined : props.calendar))
const unit = computed(() => (isScale(props.calendar) ? unitLabel(props.calendar) : ''))
const eraId = ref(cal.value?.eras[0]?.id ?? '')
const year = ref('')
const eraOptions = computed(() => (cal.value?.eras ?? []).map((e) => ({ value: e.id, label: `${e.name} (${e.abbr})` })))
const era = computed(() => cal.value?.eras.find((e) => e.id === eraId.value))
const tooLong = computed(() => model.value === '?' && era.value?.years !== undefined)

// De fora para dentro: o ano contínuo vira era + ano.
watch(
  model,
  (v) => {
    if (v === '?' || !cal.value) return
    const parts = v.trim() ? toEraYear(Number(v), cal.value) : undefined
    if (parts) [eraId.value, year.value] = [parts.era.id, String(parts.year)]
    else if (!v.trim()) year.value = ''
  },
  { immediate: true },
)

// De dentro para fora.
watch([eraId, year], ([id, y]) => {
  if (!cal.value) return
  if (!y.trim()) return void (model.value = '')
  const abs = fromEraYear(id, Number(y), cal.value)
  model.value = abs === undefined ? '?' : String(abs)
})
</script>

<template>
  <div v-if="unit" class="scale-pos">
    <span class="scale-pos__unit" aria-hidden="true">{{ unit }}</span>
    <TlInput :id="id" v-model="model" inputmode="numeric" :placeholder="placeholder ?? '0'" :aria-describedby="describedby" :invalid="invalid" />
  </div>
  <div v-else class="era-year">
    <TlSelect v-model="eraId" :options="eraOptions" :aria-label="'Era'" :invalid="invalid" />
    <TlInput :id="id" v-model="year" inputmode="numeric" :placeholder="placeholder ?? 'Ano'" :aria-describedby="describedby" :invalid="invalid" />
    <span v-if="tooLong" class="era-year__hint" role="status">A {{ era?.name }} tem {{ era?.years }} anos.</span>
  </div>
</template>

<style scoped>
.scale-pos {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: var(--tl-space-2);
}
.scale-pos__unit {
  color: var(--tl-color-text-muted);
  font-size: var(--tl-text-md);
}
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
