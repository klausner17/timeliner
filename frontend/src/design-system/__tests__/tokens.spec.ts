import { readdirSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { colorGroups, periodColors, radiusScale, spaceScale, trackColors, typeScale } from '../tokens'

const root = resolve(process.cwd(), 'src/design-system')
const css = readFileSync(resolve(root, 'tokens/tokens.css'), 'utf8')
const defined = new Set([...css.matchAll(/(--tl-[\w-]+)\s*:/g)].map((m) => m[1]))

describe('catálogo de tokens', () => {
  it('só lista tokens que existem em tokens.css', () => {
    const listed = [
      ...colorGroups.flatMap((g) => g.tokens.map((t) => t.name)),
      ...trackColors.flatMap((c) => [`--tl-track-${c}`, `--tl-track-${c}-soft`]),
      ...periodColors.flatMap((c) => [`--tl-period-${c}`, `--tl-period-${c}-soft`]),
      ...typeScale.map((t) => t.name),
      ...spaceScale,
      ...radiusScale,
    ]
    expect(defined.size).toBeGreaterThan(50)
    expect(listed.filter((n) => !defined.has(n))).toEqual([])
  })
})

describe('componentes', () => {
  it('não usam cores em hex fora dos tokens', () => {
    const dir = resolve(root, 'components')
    const files = readdirSync(dir).filter((f) => f.endsWith('.vue'))
    expect(files.length).toBeGreaterThan(10)
    const offenders = files.flatMap((f) => {
      const style = readFileSync(resolve(dir, f), 'utf8').split('<style')[1] ?? ''
      return [...style.matchAll(/#[0-9a-fA-F]{3,8}\b/g)].map((m) => `${f}: ${m[0]}`)
    })
    // Exceção conhecida: a linha de grade da comparação é mais clara que qualquer borda.
    expect(offenders.filter((o) => o !== 'TlCompareTracks.vue: #f0f1f4')).toEqual([])
  })
})
