import { describe, expect, it } from 'vitest'
import { PALETTE_TOKENS, paletteByToken, tagColor } from './tagColor'

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

describe('paletteByToken', () => {
  it('returns a distinct color for each of the 8 persisted palette tokens', () => {
    const colors = PALETTE_TOKENS.map((t) => paletteByToken(t))
    const dots = new Set(colors.map((c) => c.dot))
    expect(dots.size).toBe(PALETTE_TOKENS.length)
  })

  it('falls back to the first palette entry for an unrecognized token', () => {
    expect(paletteByToken('not-a-real-token')).toEqual(paletteByToken(PALETTE_TOKENS[0]))
  })
})
