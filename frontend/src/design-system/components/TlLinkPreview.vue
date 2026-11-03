<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import TlIcon from './TlIcon.vue'

/**
 * Cartão de um link, como os apps de conversa mostram: miniatura, site, título e resumo.
 * A miniatura é carregada direto do site de origem; se não abrir, o cartão fica sem ela.
 */
const props = defineProps<{
  url: string
  /** Texto escolhido por quem escreveu o evento; vence o título da página. */
  label?: string
  title?: string
  description?: string
  image?: string
  siteName?: string
  /** Prévia ainda chegando: mostra o esqueleto no lugar do título e do resumo. */
  loading?: boolean
}>()

const broken = ref(false)
watch(
  () => props.image,
  () => (broken.value = false),
)

/** Endereço curto para quando a página não tem título: "pt.wikipedia.org/wiki/Lei_Áurea". */
const shortUrl = computed(() => {
  try {
    const u = new URL(props.url)
    const path = decodeURIComponent(u.pathname).replace(/\/$/, '')
    return u.hostname.replace(/^www\./, '') + path
  } catch {
    return props.url
  }
})
const site = computed(() => props.siteName || shortUrl.value.split('/')[0])
const heading = computed(() => props.label || props.title || shortUrl.value)
const showImage = computed(() => !!props.image && !broken.value)
</script>

<template>
  <a class="tl-link" :class="{ 'tl-link--loading': loading }" :href="url" target="_blank" rel="noopener noreferrer" :aria-busy="loading || undefined">
    <span class="tl-link__thumb" aria-hidden="true">
      <img v-if="showImage" :src="image" alt="" loading="lazy" referrerpolicy="no-referrer" @error="broken = true" />
      <TlIcon v-else name="link" :size="22" />
    </span>
    <span class="tl-link__text">
      <span class="tl-link__site">{{ site }}</span>
      <span class="tl-link__title">{{ heading }}</span>
      <span v-if="loading" class="tl-link__skeleton"></span>
      <span v-else-if="label && title && title !== label" class="tl-link__desc">{{ title }}</span>
      <span v-else-if="description" class="tl-link__desc">{{ description }}</span>
    </span>
  </a>
</template>

<style scoped>
.tl-link {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  gap: var(--tl-space-3);
  align-items: center;
  padding: var(--tl-space-2);
  padding-right: var(--tl-space-3);
  background: var(--tl-color-surface);
  border: 1px solid var(--tl-color-border);
  border-radius: var(--tl-radius-md);
  color: inherit;
  text-decoration: none;
  transition: border-color 120ms ease, background 120ms ease;
}
.tl-link:hover {
  border-color: var(--tl-color-border-strong);
  background: var(--tl-color-surface-muted);
}
.tl-link:focus-visible {
  outline: 2px solid var(--tl-color-focus);
  outline-offset: 2px;
}
.tl-link__thumb {
  width: 88px;
  height: 66px;
  border-radius: var(--tl-radius-sm);
  background: var(--tl-color-surface-sunken);
  color: var(--tl-color-text-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.tl-link__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.tl-link__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.tl-link__site {
  font-size: var(--tl-text-xs);
  color: var(--tl-color-text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.tl-link__title {
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text);
  line-height: var(--tl-leading-snug);
  overflow-wrap: anywhere;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.tl-link__desc {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-body);
  line-height: var(--tl-leading-snug);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.tl-link__skeleton {
  height: 12px;
  width: 70%;
  margin-top: 4px;
  border-radius: var(--tl-radius-pill);
  background: var(--tl-color-surface-sunken);
  animation: tl-link-pulse 1.2s ease-in-out infinite;
}
@keyframes tl-link-pulse {
  50% {
    opacity: 0.5;
  }
}
@media (prefers-reduced-motion: reduce) {
  .tl-link__skeleton {
    animation: none;
  }
}
@media (max-width: 420px) {
  .tl-link {
    grid-template-columns: 64px minmax(0, 1fr);
  }
  .tl-link__thumb {
    width: 64px;
    height: 56px;
  }
}
</style>
