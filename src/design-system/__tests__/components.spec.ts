import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { TlButton, TlCompareTracks, TlField, TlInput, TlSegmented, TlTabs, type CompareLane } from '..'

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
