<script setup lang="ts">
import { TlButton, TlCard } from '@/design-system'

withDefaults(defineProps<{ what?: 'página' | 'timeline' }>(), { what: 'página' })
</script>

<template>
  <main class="tl-container not-found">
    <TlCard padding="lg" class="not-found__card">
      <h1 class="not-found__title">{{ what === 'timeline' ? 'Timeline não encontrada' : 'Página não encontrada' }}</h1>
      <p class="not-found__text">
        {{ what === 'timeline' ? 'Ela pode ter sido removida, ser privada ou o link estar errado.' : 'O endereço não existe ou mudou de lugar.' }}
      </p>
      <TlButton to="/" variant="dark">Explorar timelines</TlButton>
    </TlCard>
  </main>
</template>

<style scoped>
.not-found {
  padding-top: var(--tl-space-18);
  padding-bottom: var(--tl-space-18);
}
.not-found__card {
  max-width: 560px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: start;
  gap: var(--tl-space-4);
}
.not-found__title {
  font-size: var(--tl-text-3xl);
}
.not-found__text {
  color: var(--tl-color-text-muted);
}
</style>
