import { describe, expect, it } from 'vitest'
import { nextPath } from '../nextPath'

describe('nextPath', () => {
  it('só volta para caminhos do próprio app', () => {
    expect(nextPath('/t/brasil')).toBe('/t/brasil')
    expect(nextPath('//evil.com')).toBe('/perfil')
    expect(nextPath('/\\evil.com')).toBe('/perfil')
    expect(nextPath('https://evil.com')).toBe('/perfil')
    expect(nextPath(undefined, '/nova')).toBe('/nova')
  })
})
