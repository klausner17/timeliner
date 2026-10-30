<script setup lang="ts">
import { epochOptions, type Epoch } from '../years'
import TlInput from './TlInput.vue'
import TlSelect from './TlSelect.vue'

defineOptions({ inheritAttrs: false })
defineProps<{ invalid?: boolean }>()

const model = defineModel<string>({ default: '' })
const epoch = defineModel<Epoch>('epoch', { default: 'dC' })
</script>

<template>
  <div class="tl-year-input">
    <TlInput v-bind="$attrs" v-model="model" inputmode="numeric" :invalid="invalid" />
    <TlSelect v-model="epoch" :options="epochOptions" class="tl-year-input__epoch" aria-label="Antes ou depois de Cristo" />
  </div>
</template>

<style scoped>
.tl-year-input {
  display: flex;
  gap: var(--tl-space-2);
}
.tl-year-input__epoch {
  flex: 0 0 88px;
}
</style>
