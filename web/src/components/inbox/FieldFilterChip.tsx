import { X } from 'lucide-react'

interface FieldFilterChipProps {
  label: string
  value: string
  onRemove: () => void
}

// Removable pill for one active from/to/cc/bcc/subject filter (M8.1), styled
// after MOCKUP.html's .field-filter-chip / .chip-remove.
export default function FieldFilterChip({ label, value, onRemove }: FieldFilterChipProps) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-accent bg-accent-soft py-1 pl-2.5 pr-1 text-[11.5px] font-medium text-accent">
      <span>
        {label}: {value}
      </span>
      <button
        type="button"
        aria-label={`Remove ${label} filter`}
        onClick={onRemove}
        className="flex h-4 w-4 items-center justify-center rounded-full opacity-70 hover:opacity-100"
      >
        <X className="h-3 w-3" aria-hidden="true" />
      </button>
    </span>
  )
}
