import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { TlButton, TlCompareTracks, TlField, TlInput, TlOverviewBar, TlSegmented, TlTabs, type CompareLane } from '..'

const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/', component: { template: '<div />' } }] })

describe('TlButton', () => {
  it('é um <button type="button"> por padrão', () => {
    const w = mount(TlButton, { slots: { default: 'Salvar' } })
    expect(w.element.tagName).toBe('BUTTON')
    expect(w.attributes('type')).toBe('button')
    expect(w.text()).toBe('Salvar')
  })

  it('vira link com href e com to', async () => {
    expect(mount(TlButton, { props: { href: '/x' } }).element.tagName).toBe('A')
    const w = mount(TlButton, { props: { to: '/' }, global: { plugins: [router] } })
    await router.isReady()
    expect(w.element.tagName).toBe('A')
    expect(w.attributes('href')).toBe('/')
  })

  it('só ícone usa o label como aria-label', () => {
    const w = mount(TlButton, { props: { icon: 'share', iconOnly: true, label: 'Compartilhar' } })
    expect(w.attributes('aria-label')).toBe('Compartilhar')
    expect(w.text()).toBe('')
  })
})

describe('TlField', () => {
  it('liga o rótulo ao campo e a dica ao aria-describedby', () => {
    const w = mount({
      components: { TlField, TlInput },
      template: `<TlField label="Título" hint="Curto" v-slot="{ id, describedby }"><TlInput :id="id" :aria-describedby="describedby" /></TlField>`,
    })
    const id = w.find('label').attributes('for')
    expect(w.find('input').attributes('id')).toBe(id)
    const hintId = w.find('input').attributes('aria-describedby')
    expect(w.find(`#${CSS.escape(hintId!)}`).text()).toBe('Curto')
  })
})

describe('TlSegmented', () => {
  it('marca a opção atual e emite a nova', async () => {
    const w = mount(TlSegmented, {
      props: { label: 'Modo', modelValue: 'a', options: [{ value: 'a', label: 'A' }, { value: 'b', label: 'B' }] },
    })
    const [a, b] = w.findAll('button')
    expect(a.attributes('aria-pressed')).toBe('true')
    await b.trigger('click')
    expect(w.emitted('update:modelValue')?.[0]).toEqual(['b'])
  })
})

describe('TlTabs', () => {
  it('troca de aba com as setas', async () => {
    const tabs = [{ value: 'x', label: 'X' }, { value: 'y', label: 'Y' }]
    const w = mount(TlTabs, { props: { label: 'Abas', tabs, modelValue: 'x' }, attachTo: document.body })
    await w.find('[role=tablist]').trigger('keydown', { key: 'ArrowRight' })
    expect(w.emitted('update:modelValue')?.[0]).toEqual(['y'])
    await w.setProps({ modelValue: 'y' })
    expect(w.findAll('[role=tab]').map((t) => t.attributes('tabindex'))).toEqual(['-1', '0'])
    await w.find('[role=tablist]').trigger('keydown', { key: 'ArrowRight' })
    expect(w.emitted('update:modelValue')?.[1]).toEqual(['x'])
    w.unmount()
  })
})

describe('TlCompareTracks', () => {
  const lanes: CompareLane[] = [
    { id: 'a', name: 'A', color: 'a', events: [{ id: 'a1', year: 1990, title: 'Um' }, { id: 'a0', year: 1970, title: 'Fora' }] },
    { id: 'b', name: 'B', color: 'b', events: [{ id: 'b1', year: 2000, title: 'Dois' }] },
  ]

  it('mostra só eventos dentro do período e seleciona ao clicar', async () => {
    const w = mount(TlCompareTracks, { props: { lanes, start: 1980, end: 2020 } })
    const buttons = w.findAll('button')
    expect(buttons.map((b) => b.text())).toEqual(['1990Um', '2000Dois'])
    await buttons[1].trigger('click')
    expect(w.emitted('update:selected')?.[0]).toEqual(['b1'])
  })

  it('posiciona pelo ano no eixo', () => {
    const w = mount(TlCompareTracks, { props: { lanes, start: 1980, end: 2020 } })
    expect(w.findAll('button')[0].attributes('style')).toContain('left: 25%')
  })
})

describe('TlCompareTracks com timelines grandes', () => {
  const big: CompareLane[] = [
    {
      id: 'g',
      name: 'Grande',
      color: 'a',
      buckets: [
        { from: 1900, to: 1919, count: 40, sample: [{ id: 'x', year: 1900, title: 'X' }] },
        { from: 1960, to: 1979, count: 1, sample: [{ id: 'só', year: 1965, title: 'Sozinho' }] },
      ],
    },
  ]

  it('mostra faixas como grupos e faixa com um evento como evento', () => {
    const w = mount(TlCompareTracks, { props: { lanes: big, start: 1900, end: 2000 } })
    expect(w.findAll('.tl-compare__event').map((b) => b.text())).toEqual(['1900–191940 eventos', '1965Sozinho'])
  })

  it('agrupa eventos soltos que não cabem em três linhas', () => {
    const events = Array.from({ length: 30 }, (_, i) => ({ id: `e${i}`, year: 1900 + i, title: `E${i}` }))
    const w = mount(TlCompareTracks, { props: { lanes: [{ id: 'a', name: 'A', color: 'a', events }], start: 1900, end: 2000 } })
    const clusters = w.findAll('.tl-compare__event--cluster')
    expect(clusters.length).toBeGreaterThan(0)
    const grouped = clusters.reduce((n, c) => n + Number(c.find('.tl-compare__title').text().replace(' eventos', '')), 0)
    expect(grouped + w.findAll('.tl-compare__event:not(.tl-compare__event--cluster)').length).toBe(30)
  })

  it('clicar num grupo aproxima naquele trecho', async () => {
    const w = mount(TlCompareTracks, { props: { lanes: big, start: 1900, end: 2000, zoomable: true, min: 1800, max: 2000 } })
    await w.find('.tl-compare__event--cluster').trigger('click')
    expect(w.emitted('zoom')?.[0]).toEqual([[1900, 1919]])
    expect(w.emitted('update:start')?.[0]).toEqual([1898])
    expect(w.emitted('update:end')?.[0]).toEqual([1922])
  })

  it('botões de zoom mudam a janela', async () => {
    const w = mount(TlCompareTracks, { props: { lanes: big, start: 1900, end: 2000, zoomable: true } })
    await w.find('button[aria-label="Aproximar"]').trigger('click')
    expect(w.emitted('update:start')?.[0]).toEqual([1925])
    expect(w.emitted('update:end')?.[0]).toEqual([1975])
  })
})

describe('TlOverviewBar', () => {
  it('desenha um histograma a partir das faixas', () => {
    const w = mount(TlOverviewBar, { props: { start: 0, end: 100, buckets: [{ from: 0, to: 9, count: 100 }, { from: 50, to: 59, count: 1 }] } })
    expect(w.findAll('.tl-overview__mark')).toHaveLength(2)
    expect(w.find('[role="img"]').attributes('aria-label')).toBe('Visão geral: 101 eventos de 0 a 100')
  })

  it('a janela anda com as setas quando é interativa', async () => {
    const w = mount(TlOverviewBar, { props: { start: 0, end: 100, years: [10, 20], window: [10, 30], interactive: true } })
    await w.find('[role="slider"]').trigger('keydown', { key: 'ArrowRight' })
    expect(w.emitted('update:window')?.[0]).toEqual([[12, 32]])
  })
})
