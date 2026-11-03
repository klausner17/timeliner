import { describe, expect, it } from 'vitest'
import { nextTick, ref } from 'vue'
import { usePaged, type Page } from '../usePaged'

const flush = () => new Promise((r) => setTimeout(r, 0))

function fakeApi(all: string[]) {
  const calls: { q: string; page: number }[] = []
  const load = async (q: string, page: number): Promise<Page<string>> => {
    calls.push({ q, page })
    const hits = all.filter((s) => s.includes(q))
    return { items: hits.slice((page - 1) * 2, page * 2), page, pageSize: 2, total: hits.length }
  }
  return { calls, load }
}

describe('usePaged', () => {
  it('carrega a primeira página, junta as seguintes e para no total', async () => {
    const api = fakeApi(['a1', 'a2', 'a3', 'b1', 'a4'])
    const q = ref('a')
    const p = usePaged(() => q.value, api.load, (s) => s)
    await flush()
    expect(p.items.value).toEqual(['a1', 'a2'])
    expect(p.total.value).toBe(4)
    expect(p.hasMore.value).toBe(true)
    await p.loadMore()
    await p.loadMore()
    expect(p.items.value).toEqual(['a1', 'a2', 'a3', 'a4'])
    expect(p.hasMore.value).toBe(false)
    await p.loadMore()
    expect(api.calls.length).toBe(2) // depois do total, carregar mais nem chama a API
  })

  it('recomeça da página 1 quando a busca muda e descarta resposta velha', async () => {
    const api = fakeApi(['a1', 'a2', 'a3', 'b1'])
    const q = ref<string | undefined>('a')
    const p = usePaged(() => q.value, api.load, (s) => s)
    await flush()
    await p.loadMore()
    q.value = 'b'
    await nextTick()
    await flush()
    expect(p.items.value).toEqual(['b1'])
    expect(p.hasMore.value).toBe(false)
    expect(api.calls.at(-1)).toEqual({ q: 'b', page: 1 })
    q.value = undefined
    await nextTick()
    expect(api.calls.length).toBe(3)
  })
})
