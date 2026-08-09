import { describe, expect, it } from 'vitest'
import { formatBytes, formatExactTime, formatRelativeTime, truncateList } from './format'

describe('formatBytes', () => {
  it('renders zero and negative as 0 B', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(-5)).toBe('0 B')
  })

  it('renders whole bytes with no decimal', () => {
    expect(formatBytes(512)).toBe('512 B')
  })

  it('renders KB with one decimal under 10', () => {
    expect(formatBytes(1536)).toBe('1.5 KB')
  })

  it('renders larger KB values with no decimal', () => {
    expect(formatBytes(102400)).toBe('100 KB')
  })

  it('renders MB', () => {
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
  })
})

describe('formatRelativeTime', () => {
  const now = new Date('2026-01-01T12:00:00Z')

  it('renders "just now" for sub-5-second deltas', () => {
    expect(formatRelativeTime('2026-01-01T11:59:58Z', now)).toBe('just now')
  })

  it('renders minutes ago', () => {
    expect(formatRelativeTime('2026-01-01T11:58:00Z', now)).toBe('2m ago')
  })

  it('renders hours ago', () => {
    expect(formatRelativeTime('2026-01-01T09:00:00Z', now)).toBe('3h ago')
  })

  it('renders days ago', () => {
    expect(formatRelativeTime('2025-12-30T12:00:00Z', now)).toBe('2d ago')
  })

  it('clamps future timestamps to non-negative', () => {
    expect(formatRelativeTime('2026-01-01T12:05:00Z', now)).toBe('just now')
  })
})

describe('formatExactTime', () => {
  it('returns a non-empty locale string', () => {
    expect(formatExactTime('2026-01-01T12:00:00Z').length).toBeGreaterThan(0)
  })
})

describe('truncateList', () => {
  it('returns the full list when under the max', () => {
    expect(truncateList(['a'], 1)).toEqual({ shown: ['a'], more: 0 })
  })

  it('truncates and counts the remainder', () => {
    expect(truncateList(['a', 'b', 'c'], 1)).toEqual({ shown: ['a'], more: 2 })
  })

  it('handles an empty list', () => {
    expect(truncateList([], 1)).toEqual({ shown: [], more: 0 })
  })
})
