<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { TlLinkPreview } from '@/design-system'
import { loadLinkPreviews } from '@/data/store'
import type { LinkPreview, TimelineEvent } from '@/data/types'

/**
 * Tudo de um evento: data, título, descrição longa em parágrafos e a prévia de cada link
 * (fonte primeiro). Os links aparecem na hora com o endereço e ganham título e miniatura
 * quando a prévia chega.
 */
const props = defineProps<{
  slug: string
  event: TimelineEvent
  /** Data por extenso, já no calendário da timeline. */
  date: string
  /** Título menor, para o cartão ao lado das trilhas. */
  compact?: boolean
}>()

type Item = { url: string; label?: string; preview?: LinkPreview }

/** A fonte e os links, sem repetir endereço, na mesma ordem da API. */
const linkList = computed(() => {
  const out: { url: string; label?: string }[] = []
  const seen = new Set<string>()
  const add = (url: string | undefined, label?: string) => {
    if (url && !seen.has(url)) {
      seen.add(url)
      out.push({ url, label: label || undefined })
    }
  }
  add(props.event.sourceUrl, 'Fonte')
  for (const l of props.event.links ?? []) add(l.url, l.label)
  return out
})

const previews = ref<Map<string, LinkPreview>>(new Map())
const loading = ref(false)
let seq = 0

watch(
  () => [props.slug, props.event.id, linkList.value.map((l) => l.url).join(' ')],
  async () => {
    const id = ++seq
    previews.value = new Map()
    if (!linkList.value.length || !props.event.id) return
    loading.value = true
    try {
      const items = await loadLinkPreviews(props.slug, props.event)
      if (id === seq) previews.value = new Map(items.map((p) => [p.url, p]))
    } catch {
      // Sem prévia: os links ficam só com o endereço.
    } finally {
      if (id === seq) loading.value = false
    }
  },
  { immediate: true },
)

const items = computed<Item[]>(() => linkList.value.map((l) => ({ ...l, preview: previews.value.get(l.url) })))

/** "Fonte" é só o rótulo padrão: com a prévia, o título da página diz mais. */
const labelOf = (it: Item) => (it.label === 'Fonte' && it.preview?.title ? undefined : it.label)

/** Parágrafos separados por linha em branco; quebras simples ficam dentro do parágrafo. */
const paragraphs = computed(() =>
  props.event.description
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean),
)
</script>

<template>
  <article class="ev" :class="{ 'ev--compact': compact }">
    <slot name="meta">
      <span class="ev__date">{{ date }}</span>
    </slot>
    <h2 class="ev__title">{{ event.title }}</h2>
    <div v-if="paragraphs.length" class="ev__desc">
      <p v-for="(p, i) in paragraphs" :key="i">{{ p }}</p>
    </div>
    <section v-if="items.length" class="ev__links" aria-label="Fontes e links">
      <h3 class="ev__links-title">{{ items.length === 1 ? 'Fonte' : 'Fontes e links' }}</h3>
      <TlLinkPreview
        v-for="it in items"
        :key="it.url"
        :url="it.url"
        :label="labelOf(it)"
        :title="it.preview?.title"
        :description="it.preview?.description"
        :image="it.preview?.image"
        :site-name="it.preview?.siteName"
        :loading="loading && !it.preview"
      />
    </section>
    <footer class="ev__foot">
      <span>adicionado por {{ event.by }}</span>
      <slot name="actions" />
    </footer>
  </article>
</template>

<style scoped>
.ev {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
  min-width: 0;
}
.ev__date {
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
.ev__title {
  font-size: var(--tl-text-2xl);
  letter-spacing: var(--tl-tracking-tight);
  line-height: var(--tl-leading-snug);
}
.ev--compact .ev__title {
  font-size: var(--tl-text-xl);
}
.ev__desc {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-3);
  font-size: var(--tl-text-base);
  line-height: var(--tl-leading-normal);
  color: var(--tl-color-text-body);
}
.ev__desc,
.ev__links {
  max-width: 72ch;
}
.ev__desc p {
  white-space: pre-line;
  overflow-wrap: anywhere;
}
.ev__links {
  display: flex;
  flex-direction: column;
  gap: var(--tl-space-2);
  margin-top: var(--tl-space-1);
}
.ev__links-title {
  font-family: var(--tl-font-body);
  font-size: var(--tl-text-sm);
  font-weight: var(--tl-weight-semibold);
  color: var(--tl-color-text-muted);
  letter-spacing: 0;
}
.ev__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 16px;
  font-size: var(--tl-text-sm);
  color: var(--tl-color-text-muted);
}
</style>
