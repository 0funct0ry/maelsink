import { hasActiveFilter } from './queryFilters'

describe('hasActiveFilter', () => {
  it('is false for an empty query', () => {
    expect(hasActiveFilter({})).toBe(false)
  })

  it('is false when only sort/pagination fields are set', () => {
    expect(hasActiveFilter({ sort: 'received_at_asc', limit: 50, offset: 0 })).toBe(false)
  })

  it('is true when a search query is set', () => {
    expect(hasActiveFilter({ q: 'invoice' })).toBe(true)
  })

  it('is true when a boolean filter is explicitly false', () => {
    expect(hasActiveFilter({ read: false })).toBe(true)
  })

  it('is true when tags are set', () => {
    expect(hasActiveFilter({ tag: ['work'] })).toBe(true)
  })

  it('is false when tag is an empty array', () => {
    expect(hasActiveFilter({ tag: [] })).toBe(false)
  })
})
