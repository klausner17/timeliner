import { shallowRef, watchEffect, type ShallowRef } from 'vue'

/**
 * Roda load sempre que o que ele lê mudar e guarda só a resposta mais recente: se o
 * zoom muda rápido, uma resposta antiga que chega depois é descartada.
 */
export function useLatest<T>(load: () => Promise<T> | undefined, initial: T): { data: ShallowRef<T>; error: ShallowRef<string> } {
  const data = shallowRef(initial)
  const error = shallowRef('')
  let seq = 0
  watchEffect(() => {
    const id = ++seq
    const p = load()
    if (!p) return
    p.then(
      (v) => {
        if (id !== seq) return
        data.value = v
        error.value = ''
      },
      (e: unknown) => {
        if (id === seq) error.value = e instanceof Error ? e.message : 'Não deu para carregar os eventos.'
      },
    )
  })
  return { data, error }
}
