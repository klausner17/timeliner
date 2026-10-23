<script setup lang="ts">
import { useRouter } from 'vue-router'
import { createTimeline } from '@/data/store'

// "Criar timeline" já abre o editor com um rascunho novo.
const router = useRouter()
const t = createTimeline()
router.replace({ path: `/t/${t.slug}/editar`, query: { nova: '1' } })
</script>

<template>
  <main class="tl-container" aria-busy="true"></main>
</template>
