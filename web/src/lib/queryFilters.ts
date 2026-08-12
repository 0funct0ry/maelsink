import type { ListMessagesParams } from './apiTypes'

// Whether query narrows the result set beyond "every message, default
// sort" — used to distinguish a first-run empty mailbox from a query that
// simply matches nothing (M8.7's two distinct empty states).
export function hasActiveFilter(query: ListMessagesParams): boolean {
  return Boolean(
    query.q ||
      query.from ||
      query.to ||
      query.cc ||
      query.bcc ||
      query.subject ||
      query.since ||
      query.until ||
      (query.tag && query.tag.length > 0) ||
      query.read !== undefined ||
      query.has_attachments !== undefined ||
      query.parse_warning !== undefined,
  )
}
