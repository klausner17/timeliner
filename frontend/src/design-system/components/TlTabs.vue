<script setup lang="ts">
import { ref, useId } from 'vue'

export type Tab = { value: string; label: string; count?: number }

const props = defineProps<{ label: string; tabs: Tab[] }>()

const model = defineModel<string>()
const base = useId()
const buttons = ref<HTMLButtonElement[]>([])

const tabId = (v: string) => `${base}-tab-${v}`
const panelId = (v: string) => `${base}-panel-${v}`

function move(delta: number) {
  const i = props.tabs.findIndex((t) => t.value === model.value)
  const next = (i + delta + props.tabs.length) % props.tabs.length
  model.value = props.tabs[next].value
  buttons.value[next]?.focus()
}
</script>

<template>
  <div class="tl-tabs">
    <div class="tl-tabs__list" role="tablist" :aria-label="label" @keydown.right.prevent="move(1)" @keydown.left.prevent="move(-1)">
      <button
        v-for="t in tabs"
        :id="tabId(t.value)"
        ref="buttons"
        :key="t.value"
        type="button"
        role="tab"
        class="tl-tabs__tab"
        :class="{ 'tl-tabs__tab--on': model === t.value }"
        :aria-selected="model === t.value"
        :aria-controls="panelId(t.value)"
        :tabindex="model === t.value ? 0 : -1"
        @click="model = t.value"
      >
        {{ t.label }}<span v-if="t.count !== undefined" class="tl-tabs__count">{{ t.count }}</span>
      </button>
    </div>
    <div v-if="$slots.default" :id="panelId(model ?? '')" role="tabpanel" :aria-labelledby="tabId(model ?? '')">
      <slot :value="model" />
    </div>
  </div>
</template>

<style scoped>
.tl-tabs {
  min-width: 0;
  max-width: 100%;
}
.tl-tabs__list {
  display: flex;
  gap: var(--tl-space-1);
  overflow-x: auto;
}
.tl-tabs__tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 48px;
  padding: 0 16px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--tl-color-text-muted);
  font-weight: var(--tl-weight-semibold);
  font-size: var(--tl-text-base);
  white-space: nowrap;
  cursor: pointer;
}
.tl-tabs__tab:hover {
  color: var(--tl-color-text);
}
.tl-tabs__tab--on {
  color: var(--tl-color-text);
  border-bottom-color: var(--tl-color-text);
}
.tl-tabs__count {
  padding: 1px 7px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-neutral-soft);
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
}
</style>
