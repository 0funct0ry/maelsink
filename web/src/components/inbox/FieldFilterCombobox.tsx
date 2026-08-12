import { ChevronDown } from 'lucide-react'
import { useState } from 'react'

interface FieldFilterComboboxProps {
  label: string
  value: string
  placeholder: string
  /** Distinct values for this field across the loaded messages. */
  options: string[]
  /** Free-text edit; the parent debounces this. */
  onChange: (next: string) => void
  /** Picked from the dropdown; the parent applies it immediately. */
  onSelect: (next: string) => void
}

const MAX_OPTIONS = 8

// One labelled field of the inbox filter panel: a free-text substring input
// that also offers the distinct values present in the loaded messages, so
// common addresses/subjects can be picked instead of typed.
export default function FieldFilterCombobox({
  label,
  value,
  placeholder,
  options,
  onChange,
  onSelect,
}: FieldFilterComboboxProps) {
  const [open, setOpen] = useState(false)
  const id = `field-filter-${label.toLowerCase()}`

  const needle = value.trim().toLowerCase()
  const matches = options.filter((o) => o.toLowerCase().includes(needle)).slice(0, MAX_OPTIONS)

  return (
    <div
      className="relative flex flex-col gap-1"
      onBlur={(e) => {
        if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setOpen(false)
      }}
    >
      <label htmlFor={id} className="text-[11px] font-semibold uppercase tracking-wide text-text-tertiary">
        {label}
      </label>
      <div className="flex items-center rounded border border-border-soft bg-bg pr-1 focus-within:border-accent">
        <input
          id={id}
          type="text"
          role="combobox"
          aria-expanded={open}
          aria-controls={`${id}-listbox`}
          aria-label={`Filter by ${label.toLowerCase()}`}
          autoComplete="off"
          value={value}
          placeholder={placeholder}
          onFocus={() => setOpen(true)}
          onKeyDown={(e) => {
            if (e.key === 'Escape') setOpen(false)
          }}
          onChange={(e) => {
            setOpen(true)
            onChange(e.target.value)
          }}
          className="w-[150px] bg-transparent px-2 py-1 text-[12.5px] text-text-primary placeholder:text-text-tertiary focus:outline-none"
        />
        {options.length > 0 && (
          <button
            type="button"
            tabIndex={-1}
            aria-label={`Show ${label.toLowerCase()} values`}
            onClick={() => setOpen((o) => !o)}
            className="flex h-5 w-5 flex-none items-center justify-center rounded text-text-tertiary hover:text-text-secondary"
          >
            <ChevronDown className="h-3.5 w-3.5" aria-hidden="true" />
          </button>
        )}
      </div>

      {open && matches.length > 0 && (
        <ul
          id={`${id}-listbox`}
          role="listbox"
          className="scrollbar-thin absolute left-0 top-full z-30 mt-1 max-h-52 w-full min-w-[150px] overflow-y-auto rounded-md border border-border bg-bg py-1 shadow-md"
        >
          {matches.map((option) => (
            <li key={option} role="option" aria-selected={option === value}>
              <button
                type="button"
                onClick={() => {
                  onSelect(option)
                  setOpen(false)
                }}
                className="block w-full truncate px-2.5 py-1.5 text-left text-[12.5px] text-text-secondary hover:bg-surface hover:text-text-primary"
              >
                {option}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}
