import { describe, expect, it } from 'vitest'
import { LABEL_W, MAX_ROWS, clampWindow, mergeByColumn, niceStep, pack, posLabel, tickLabel, windowTicks, zoomWindow, type TrackItem } from '../components/trackLayout'

const item = (px: number, id = `e${px}`): TrackItem => ({ kind: 'event', key: id, event: { id, year: px, title: id }, year: px, px })

describe('pack e mergeByColumn', () => {
  it('encaixa rótulos próximos em linhas diferentes', () => {
    const placed = pack([item(0), item(50), item(LABEL_W)], 900)!
    expect(placed.map((p) => p.row)).toEqual([0, 1, 0])
  })

  it('desiste quando precisaria de mais de MAX_ROWS linhas, e os grupos por coluna cabem', () => {
    const crowded = Array.from({ length: 40 }, (_, i) => item(i * 5))
    expect(pack(crowded, 900)).toBeUndefined()
    const merged = mergeByColumn(crowded, (y) => y)
    expect(merged).toHaveLength(2)
    expect(merged[0]).toMatchObject({ kind: 'cluster', from: 0, to: 175, count: 36 })
    expect(pack(merged, 900)!.every((p) => p.row < MAX_ROWS)).toBe(true)
  })
})

describe('zoom', () => {
  it('aproxima em volta do ponto e respeita os limites', () => {
    expect(zoomWindow(1900, 2000, 0.5, 1950)).toEqual([1925, 1975])
    expect(zoomWindow(1900, 2000, 0.5, 1900)).toEqual([1900, 1950])
    expect(zoomWindow(1900, 2000, 4, 1950, { min: 1800, max: 2020 })).toEqual([1800, 2020])
    // O zoom chega a três meses, em meses inteiros.
    const [a, b] = zoomWindow(1900, 1902, 0.1, 1901)
    expect(Math.round((b - a) * 12)).toBe(3)
    expect(Number.isInteger(Math.round(a * 12 * 1e6) / 1e6)).toBe(true)
  })

  it('arraste não sai dos limites nem muda o tamanho', () => {
    expect(clampWindow(1790.4, 1890.4, { min: 1800, max: 2020 })).toEqual([1800, 1900])
    expect(clampWindow(1950, 2050, { min: 1800, max: 2020 })).toEqual([1920, 2020])
  })

  it('escolhe passos legíveis para as marcações', () => {
    expect(niceStep(40)).toBe(5)
    expect(niceStep(4000)).toBe(500)
    expect(niceStep(3)).toBe(1)
    expect(niceStep(1)).toBe(2 / 12)
    expect(niceStep(0.5)).toBe(1 / 12)
  })

  it('marca e escreve os meses no eixo', () => {
    expect(windowTicks(1930.5, 1931.25, 3 / 12).map(tickLabel)).toEqual(['jul.', 'out.', '1931', 'abr.'])
    expect(posLabel(1930 + 9 / 12)).toBe('out. 1930')
    expect(posLabel(1930)).toBe('1930')
  })
})
