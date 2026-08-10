// The five per-field filters of the inbox filter panel (M8.1), shared by
// FieldFilterBar and ActiveFilterChips so both render the same set/labels.
export type FieldFilterKey = 'from' | 'to' | 'cc' | 'bcc' | 'subject'

export const FIELD_FILTER_FIELDS: { key: FieldFilterKey; label: string; placeholder: string }[] = [
  { key: 'from', label: 'From', placeholder: 'sender@…' },
  { key: 'to', label: 'To', placeholder: 'recipient@…' },
  { key: 'cc', label: 'Cc', placeholder: 'cc@…' },
  { key: 'bcc', label: 'Bcc', placeholder: 'bcc@…' },
  { key: 'subject', label: 'Subject', placeholder: 'contains…' },
]
