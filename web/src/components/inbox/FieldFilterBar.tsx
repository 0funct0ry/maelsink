import { ChevronDown, SlidersHorizontal } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useMessageStore } from '../../stores/useMessageStore'
import type { ListMessagesParams, MessageSummary } from '../../lib/apiTypes'
import FieldFilterCombobox from './FieldFilterCombobox'
import { FIELD_FILTER_FIELDS, type FieldFilterKey } from './fieldFilters'

const DEBOUNCE_MS = 300

const EMPTY_VALUES: Record<FieldFilterKey, string> = { from: '', to: '', cc: '', bcc: '', subject: '' }

// Distinct, sorted values for one field across the loaded messages — the
// dropdown suggestions. Scoped to the current page of results (no dedicated
// endpoint exists for distinct values), so it's a shortcut for what's on
// screen, not an exhaustive index; free text still filters server-side.
function distinctValues(messages: MessageSummary[], key: FieldFilterKey): string[] {
  const seen = new Set<string>()
  for (const m of messages) {
    const field = m[key]
    const values = Array.isArray(field) ? field : [field]
    for (const v of values) {
      const trimmed = v?.trim()
      if (trimmed) seen.add(trimmed)
    }
  }
  return [...seen].sort((a, b) => a.localeCompare(b))
}

// Per-field filter panel for the Inbox list header (M8.1): from/to/cc/bcc/
// subject, coexisting with the topbar's combined free-text `q` box. Each
// field debounces independently and writes into useMessageStore's query,
// matching SearchBar's pattern.
export default function FieldFilterBar() {
  const query = useMessageStore((state) => state.query)
  const setQuery = useMessageStore((state) => state.setQuery)
  const messages = useMessageStore((state) => state.messages)
  const [open, setOpen] = useState(false)
  const [values, setValues] = useState<Record<FieldFilterKey, string>>(EMPTY_VALUES)
  const timers = useRef<Partial<Record<FieldFilterKey, ReturnType<typeof setTimeout>>>>({})

  const options = useMemo(
    () =>
      Object.fromEntries(FIELD_FILTER_FIELDS.map((f) => [f.key, distinctValues(messages, f.key)])) as Record<
        FieldFilterKey,
        string[]
      >,
    [messages],
  )

  useEffect(() => {
    setValues({
      from: query.from ?? '',
      to: query.to ?? '',
      cc: query.cc ?? '',
      bcc: query.bcc ?? '',
      subject: query.subject ?? '',
    })
  }, [query.from, query.to, query.cc, query.bcc, query.subject])

  useEffect(() => {
    return () => {
      Object.values(timers.current).forEach((t) => t && clearTimeout(t))
    }
  }, [])

  function apply(key: FieldFilterKey, next: string) {
    const existing = timers.current[key]
    if (existing) clearTimeout(existing)
    setValues((v) => ({ ...v, [key]: next }))
    setQuery({ [key]: next } as Partial<ListMessagesParams>)
  }

  function handleChange(key: FieldFilterKey, next: string) {
    setValues((v) => ({ ...v, [key]: next }))
    const existing = timers.current[key]
    if (existing) clearTimeout(existing)
    timers.current[key] = setTimeout(() => {
      setQuery({ [key]: next } as Partial<ListMessagesParams>)
    }, DEBOUNCE_MS)
  }

  function handleClearAll() {
    Object.values(timers.current).forEach((t) => t && clearTimeout(t))
    setValues(EMPTY_VALUES)
    setQuery({ from: '', to: '', cc: '', bcc: '', subject: '' })
  }

  const activeCount = FIELD_FILTER_FIELDS.filter((f) => (query[f.key] ?? '') !== '').length

  return (
    <div className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-label="Filter by from, to, cc, bcc, subject"
        onClick={() => setOpen((o) => !o)}
        className={`flex items-center gap-1.5 rounded-sm border px-2.5 py-1.5 text-[12.5px] transition-colors ${
          open || activeCount > 0
            ? 'border-accent bg-accent-soft text-accent'
            : 'border-border-soft bg-surface text-text-secondary hover:border-border'
        }`}
      >
        <SlidersHorizontal className="h-[13px] w-[13px]" aria-hidden="true" />
        Filters
        {activeCount > 0 && (
          <span className="min-w-[16px] rounded-full bg-accent px-1.5 text-center text-[10.5px] font-semibold leading-4 text-white">
            {activeCount}
          </span>
        )}
        <ChevronDown className="h-[13px] w-[13px]" aria-hidden="true" />
      </button>

      {open && (
        <div className="absolute right-0 top-full z-20 mt-1 flex w-max max-w-[min(90vw,720px)] flex-wrap items-start gap-3 rounded-md border border-border bg-bg p-3 shadow-md">
          {FIELD_FILTER_FIELDS.map((f) => (
            <FieldFilterCombobox
              key={f.key}
              label={f.label}
              value={values[f.key]}
              placeholder={f.placeholder}
              options={options[f.key]}
              onChange={(next) => handleChange(f.key, next)}
              onSelect={(next) => apply(f.key, next)}
            />
          ))}
          <button
            type="button"
            onClick={handleClearAll}
            disabled={activeCount === 0}
            className="mt-[22px] h-[28px] whitespace-nowrap rounded px-2 text-xs font-medium text-text-tertiary hover:text-text-secondary disabled:cursor-default disabled:opacity-40"
          >
            Clear filters
          </button>
        </div>
      )}
    </div>
  )
}
