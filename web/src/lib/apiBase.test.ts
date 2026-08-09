import { afterEach, describe, expect, it } from 'vitest'
import { apiUrl, basePath, wsUrl } from './apiBase'

afterEach(() => {
  delete window.__MAELSINK_BASE__
})

describe('basePath', () => {
  it('returns "" when unset', () => {
    expect(basePath()).toBe('')
  })

  it('returns "" for the unresolved placeholder', () => {
    window.__MAELSINK_BASE__ = '__MAELSINK_BASE_PATH__'
    expect(basePath()).toBe('')
  })

  it('strips a trailing slash from a configured base path', () => {
    window.__MAELSINK_BASE__ = '/maelsink/'
    expect(basePath()).toBe('/maelsink')
  })
})

describe('apiUrl', () => {
  it('joins the base path onto an absolute path', () => {
    window.__MAELSINK_BASE__ = '/maelsink'
    expect(apiUrl('/api/v1/messages')).toBe('/maelsink/api/v1/messages')
  })

  it('works at the root when no base path is set', () => {
    expect(apiUrl('/api/v1/messages')).toBe('/api/v1/messages')
  })
})

describe('wsUrl', () => {
  it('builds a ws:// URL under the resolved base path', () => {
    window.__MAELSINK_BASE__ = '/maelsink'
    expect(wsUrl()).toBe(`ws://${window.location.host}/maelsink/ws`)
  })
})
