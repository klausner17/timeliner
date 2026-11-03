import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { errorMessage } from '@/data/api'
import { favoriteTimeline, session } from '@/data/store'
import type { Timeline } from '@/data/types'

/**
 * Dar e tirar estrela de uma timeline. Sem login, leva para entrar e volta para a mesma
 * página. Atualiza `favorited` e `favorites` na hora e desfaz se a API recusar.
 */
export function useFavorite(onError?: (msg: string) => void) {
  const route = useRoute()
  const router = useRouter()
  const busy = ref(new Set<string>())

  async function toggle(t: Pick<Timeline, 'slug' | 'favorites' | 'favorited'>) {
    if (!session.user) {
      router.push({ name: 'login', query: { next: route.fullPath } })
      return
    }
    if (busy.value.has(t.slug)) return
    const on = !t.favorited
    t.favorited = on
    t.favorites += on ? 1 : -1
    busy.value = new Set(busy.value).add(t.slug)
    try {
      await favoriteTimeline(t.slug, on)
    } catch (e) {
      t.favorited = !on
      t.favorites += on ? -1 : 1
      onError?.(errorMessage(e))
    } finally {
      const next = new Set(busy.value)
      next.delete(t.slug)
      busy.value = next
    }
  }

  return { toggle, busy }
}
