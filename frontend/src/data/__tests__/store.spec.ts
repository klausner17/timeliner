import { beforeEach, describe, expect, it } from 'vitest'
import {
  createTimeline,
  loadComparison,
  loadEvents,
  loadSummary,
  decideSuggestion,
  findTimeline,
  rangeLabel,
  resetStore,
  saveComparison,
  state,
  suggest,
  trackPoints,
  undoSuggestion,
} from '../store'

beforeEach(() => resetStore())

describe('sugestões', () => {
  it('aprovar um evento novo coloca na timeline e desfazer tira', () => {
    const t = findTimeline('historia-do-brasil')!
    const before = t.events.length
    decideSuggestion('s3', 'aprovada')
    expect(t.events).toHaveLength(before + 1)
    expect(t.events.find((e) => e.title === 'Inauguração de Brasília')?.by).toBe('@ana.m')
    undoSuggestion('s3')
    expect(t.events).toHaveLength(before)
    expect(state.suggestions.find((s) => s.id === 's3')?.status).toBe('pendente')
  })

  it('aprovar uma correção troca o evento e desfazer volta o texto antigo', () => {
    const t = findTimeline('historia-do-brasil')!
    const lei = () => t.events.find((e) => e.title === 'Lei Áurea')!
    const original = lei().description
    decideSuggestion('s2', 'aprovada')
    expect(lei().description).toContain('último país das Américas')
    undoSuggestion('s2')
    expect(lei().description).toBe(original)
  })

  it('recusar não mexe nos eventos', () => {
    const t = findTimeline('historia-do-brasil')!
    const before = t.events.length
    decideSuggestion('s1', 'recusada')
    expect(t.events).toHaveLength(before)
  })

  it('uma sugestão nova entra como pendente do usuário atual', () => {
    const s = suggest({ timeline: 'copas-do-mundo', kind: 'Novo evento', event: { year: 1982, title: 'Tragédia do Sarriá', description: 'Itália 3 x 2 Brasil.' } })
    expect(s.status).toBe('pendente')
    expect(s.by).toBe('@klausner')
  })
})

describe('timelines', () => {
  it('criar reaproveita um rascunho vazio', () => {
    const a = createTimeline()
    const b = createTimeline()
    expect(a.slug).toBe(b.slug)
    expect(a.draft).toBe(true)
  })

  it('calcula o período e os pontos do cartão', () => {
    const t = findTimeline('escandalos-politicos')!
    expect(rangeLabel(t)).toBe('1992–2017')
    const pts = trackPoints(t)
    expect(pts.length).toBeLessThanOrEqual(5)
    expect(Math.min(...pts)).toBeGreaterThanOrEqual(0)
    expect(Math.max(...pts)).toBeLessThanOrEqual(100)
  })
})

describe('comparações', () => {
  it('salvar a mesma comparação duas vezes não duplica', () => {
    const n = state.savedComparisons.length
    saveComparison(['copas-do-mundo', 'presidentes-da-republica'], 1930, 2020)
    saveComparison(['copas-do-mundo', 'presidentes-da-republica'], 1930, 2020)
    expect(state.savedComparisons).toHaveLength(n + 1)
  })
})

describe('timelines grandes', () => {
  it('a demonstração tem 20 mil eventos e não vai para o localStorage', async () => {
    const t = findTimeline('demonstracao-20-mil-eventos')!
    expect(t.events).toHaveLength(20000)
    const first = await loadEvents(t.slug, { limit: 100 })
    expect(first.items).toHaveLength(100)
    const second = await loadEvents(t.slug, { limit: 100, cursor: first.nextCursor })
    expect(second.items[0].year).toBeGreaterThanOrEqual(first.items[99].year)
    const summary = await loadSummary(t.slug, { buckets: 40 })
    expect(summary.mode).toBe('faixas')
    expect(summary.total).toBe(20000)
    state.bio = 'mexe no estado para salvar'
    await Promise.resolve()
    expect(localStorage.getItem('timeliner:v1') ?? '').not.toContain('demonstracao-20-mil-eventos')
  })

  it('a comparação traz cada trilha no nível de detalhe certo', async () => {
    const c = await loadComparison(['historia-do-brasil', 'demonstracao-20-mil-eventos'], 1500, 2025)
    expect(c.lanes[0].mode).toBe('eventos')
    expect(c.lanes[1].mode).toBe('faixas')
    expect(c.overlapsTotal).toBeGreaterThan(0)
  })
})
