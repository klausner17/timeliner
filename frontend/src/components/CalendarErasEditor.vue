<script lang="ts">
import type { CalendarEra, CalendarEraInput } from '@/data/types'

/** Uma era no formulário: a duração é texto, vazia só na última. */
export type EraRow = { id?: string; name: string; abbr: string; years: string; key: number }

let nextKey = 0
export const eraRow = (e?: CalendarEra): EraRow => ({ id: e?.id, name: e?.name ?? '', abbr: e?.abbr ?? '', years: e?.years ? String(e.years) : '', key: nextKey++ })

/** Confere as eras do formulário; devolve o que vai para a API ou a primeira mensagem de erro. */
export function readEraRows(rows: EraRow[]): { eras: CalendarEraInput[]; error?: undefined } | { eras?: undefined; error: string } {
  if (!rows.length) return { error: 'O calendário precisa de pelo menos uma era.' }
  const names = new Set<string>()
  const eras: CalendarEraInput[] = []
  for (const [i, r] of rows.entries()) {
    const name = r.name.trim()
    const abbr = r.abbr.trim()
    const last = i === rows.length - 1
    if (!name) return { error: `Dê um nome à ${i + 1}ª era.` }
    if (names.has(name.toLowerCase())) return { error: `Há duas eras chamadas “${name}”.` }
    names.add(name.toLowerCase())
    if (!abbr) return { error: `Dê uma sigla à ${name} (ex.: T.A.).` }
    const years = r.years.trim() ? Number(r.years) : null
    if (years === null && !last) return { error: `Informe quantos anos durou a ${name}. Só a última era pode ficar sem fim.` }
    if (years !== null && (!Number.isInteger(years) || years < 1)) return { error: `A duração da ${name} precisa ser um número de anos.` }
    eras.push({ ...(r.id ? { id: r.id } : {}), name, abbr, years })
  }
  return { eras }
}
</script>

<script setup lang="ts">
import { TlButton, TlInput } from '@/design-system'

const rows = defineModel<EraRow[]>({ required: true })

const add = () => (rows.value = [...rows.value, eraRow()])
const remove = (i: number) => (rows.value = rows.value.filter((_, j) => j !== i))
/** Só eras novas mudam de lugar: as que já existem guardam a ordem dos eventos. */
function move(i: number, by: number) {
  const j = i + by
  if (j < 0 || j >= rows.value.length) return
  const next = [...rows.value]
  ;[next[i], next[j]] = [next[j], next[i]]
  rows.value = next
}
</script>

<template>
  <div class="eras">
    <div class="eras__head" aria-hidden="true">
      <span>Era</span>
      <span>Sigla</span>
      <span>Duração (anos)</span>
    </div>
    <div v-for="(r, i) in rows" :key="r.key" class="eras__row">
      <TlInput v-model="r.name" :aria-label="`Nome da ${i + 1}ª era`" placeholder="Ex.: Terceira Era" />
      <TlInput v-model="r.abbr" :aria-label="`Sigla da ${i + 1}ª era`" placeholder="T.A." maxlength="12" />
      <TlInput
        v-model="r.years"
        inputmode="numeric"
        :aria-label="`Duração da ${i + 1}ª era em anos`"
        :placeholder="i === rows.length - 1 ? 'Sem fim' : 'Anos'"
      />
      <div class="eras__actions">
        <template v-if="!r.id">
          <TlButton icon-only icon="up" size="sm" variant="ghost" :label="`Subir ${r.name || 'era'}`" :disabled="i === 0" @click="move(i, -1)" />
          <TlButton icon-only icon="down" size="sm" variant="ghost" :label="`Descer ${r.name || 'era'}`" :disabled="i === rows.length - 1" @click="move(i, 1)" />
        </template>
        <TlButton icon-only icon="trash" size="sm" variant="ghost" :label="`Remover ${r.name || 'era'}`" @click="remove(i)" />
      </div>
    </div>
    <TlButton variant="dashed" shape="rounded" icon="plus" @click="add">Adicionar era</TlButton>
    <p class="eras__hint">Os anos recomeçam do 1 em cada era. Só a última pode ficar sem duração.</p>
  </div>
</template>

<style scoped>
.eras {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
}
.eras__head,
.eras__row {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: var(--tl-space-2);
  align-items: center;
}
.eras__head {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.eras__actions {
  display: flex;
  gap: 2px;
  min-width: 36px;
}
/* No celular, o nome ocupa a linha de cima e a sigla e a duração dividem a de baixo. */
@media (max-width: 560px) {
  .eras__head {
    display: none;
  }
  .eras__row {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
    padding-bottom: var(--tl-space-2);
    border-bottom: 1px solid var(--tl-color-border-subtle);
  }
  .eras__row > :first-child {
    grid-column: 1 / 3;
  }
  .eras__actions {
    grid-column: 3;
    grid-row: 1 / 3;
    flex-direction: column;
  }
}
.eras__hint {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
