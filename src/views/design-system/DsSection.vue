<script setup lang="ts">
defineProps<{ id: string; title: string; description?: string; usage?: string }>()
</script>

<template>
  <section :id="id" class="ds-section">
    <header class="ds-section__head">
      <h2 class="ds-section__title">{{ title }}</h2>
      <p v-if="description" class="ds-section__desc">{{ description }}</p>
    </header>
    <slot />
    <pre v-if="usage" class="ds-section__code"><code>{{ usage }}</code></pre>
  </section>
</template>

<style scoped>
.ds-section {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-5);
  padding-bottom: var(--tl-space-12);
  border-bottom: 1px solid var(--tl-color-border);
  scroll-margin-top: var(--tl-space-6);
}
.ds-section__head {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.ds-section__title {
  font-size: var(--tl-text-3xl);
}
.ds-section__desc {
  max-width: 680px;
  color: var(--tl-color-text-muted);
}
.ds-section__code {
  margin: 0;
  padding: var(--tl-space-4);
  border-radius: var(--tl-radius-lg);
  background: var(--tl-color-surface-inverse);
  color: #e4e6eb;
  font-size: var(--tl-text-sm);
  line-height: 1.6;
  overflow-x: auto;
}
</style>
