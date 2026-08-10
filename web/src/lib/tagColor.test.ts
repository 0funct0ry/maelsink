import { describe, expect, it } from 'vitest'
import { tagColor } from './tagColor'

describe('tagColor', () => {
  it('returns the same color for the same tag every call', () => {
    expect(tagColor('smoke')).toEqual(tagColor('smoke'))
  })

  it('returns a dot and text class', () => {
    const c = tagColor('release')
    expect(c.dot).toMatch(/^bg-/)
    expect(c.text).toMatch(/^text-/)
  })

  it('is deterministic across different tag names (not necessarily unique)', () => {
    const a = tagColor('alpha')
    const b = tagColor('beta')
    expect(a).toEqual(tagColor('alpha'))
    expect(b).toEqual(tagColor('beta'))
  })
})
