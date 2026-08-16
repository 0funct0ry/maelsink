import { describe, expect, it } from 'vitest'
import { extractHtmlPreview } from './previewHtml'

describe('extractHtmlPreview', () => {
  it('extracts the body of an eml message with a text/html Content-Type', () => {
    const raw = 'From: a@b.com\r\nTo: c@d.com\r\nSubject: hi\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<h1>hi</h1>'
    expect(extractHtmlPreview('eml', raw)).toBe('<h1>hi</h1>')
  })

  it('returns null for a plain-text eml message', () => {
    const raw = 'From: a@b.com\r\nTo: c@d.com\r\nSubject: hi\r\n\r\nplain body'
    expect(extractHtmlPreview('eml', raw)).toBeNull()
  })

  it('extracts the html field from a rendered json spec', () => {
    const raw = JSON.stringify({ from: 'a@b.com', to: ['c@d.com'], subject: 'hi', html: '<p>hi</p>' })
    expect(extractHtmlPreview('json', raw)).toBe('<p>hi</p>')
  })

  it('returns null for a json spec with no html field', () => {
    const raw = JSON.stringify({ from: 'a@b.com', to: ['c@d.com'], subject: 'hi', text: 'hi' })
    expect(extractHtmlPreview('json', raw)).toBeNull()
  })

  it('returns null for invalid/partial json (mid-edit)', () => {
    expect(extractHtmlPreview('json', '{ "from": "a@b.com"')).toBeNull()
  })
})
