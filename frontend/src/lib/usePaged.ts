import { computed, shallowRef, watch, type ComputedRef, type ShallowRef } from 'vue'

export type Page<T> = { items: T[]; page: number; pageSize: number; total: number }

export type Paged<T> = {
  /** Itens de todas as páginas já carregadas; undefined até a primeira chegar. */
  items: ShallowRef<T[] | undefined>
  total: ShallowRef<number>
  loading: ShallowRef<boolean>
  error: ShallowRef<string>
  hasMore: ComputedRef<boolean>
  /** Busca a página seguinte e junta às que já estão na tela. */
  loadMore: () => Promise<void>
}

/**
 * Lista paginada no servidor com "carregar mais". Recomeça da página 1 sempre que o que
 * key lê mudar (busca, filtros); undefined deixa a lista parada. Uma resposta que chega
 * depois de a busca mudar é descartada.
 */
export function usePaged<T, K>(key: () => K | undefined, load: (key: K, page: number) => Promise<Page<T>>, idOf: (item: T) => string): Paged<T> {
  const items = shallowRef<T[] | undefined>()
  const total = shallowRef(0)
  const loading = shallowRef(false)
  const error = shallowRef('')
  let page = 0
  let seq = 0
  let current: K | undefined

  async function fetchPage(k: K, n: number, id: number) {
    loading.value = true
    try {
      const res = await load(k, n)
      if (id !== seq) return
      // Sem duplicar o que já está na tela se a ordem mudou entre uma página e outra.
      const seen = new Set((n > 1 ? (items.value ?? []) : []).map(idOf))
      items.value = n > 1 ? [...(items.value ?? []), ...res.items.filter((x) => !seen.has(idOf(x)))] : res.items
      total.value = res.total
      page = n
      error.value = ''
    } catch (e) {
      if (id === seq) error.value = e instanceof Error ? e.message : 'Não deu para carregar.'
    } finally {
      if (id === seq) loading.value = false
    }
  }

  watch(
    key,
    (k) => {
      const id = ++seq
      current = k
      page = 0
      if (k === undefined) {
        loading.value = false
        return
      }
      void fetchPage(k, 1, id)
    },
    { immediate: true, deep: true },
  )

  const hasMore = computed(() => !!items.value && items.value.length < total.value)

  async function loadMore() {
    if (current === undefined || loading.value || !hasMore.value) return
    await fetchPage(current, page + 1, seq)
  }

  return { items, total, loading, error, hasMore, loadMore }
}
