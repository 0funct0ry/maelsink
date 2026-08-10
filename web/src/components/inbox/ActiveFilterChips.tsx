import { useMessageStore } from '../../stores/useMessageStore'
import type { ListMessagesParams } from '../../lib/apiTypes'
import FieldFilterChip from './FieldFilterChip'
import { FIELD_FILTER_FIELDS } from './fieldFilters'

// Strip of removable chips for the active from/to/cc/bcc/subject filters
// (M8.1), rendered under the Inbox list header so active filters stay
// visible without opening the filter panel. Renders nothing when no field
// filter is set.
export default function ActiveFilterChips() {
  const query = useMessageStore((state) => state.query)
  const setQuery = useMessageStore((state) => state.setQuery)

  const active = FIELD_FILTER_FIELDS.filter((f) => (query[f.key] ?? '') !== '')
  if (active.length === 0) return null

  return (
    <div className="flex flex-wrap gap-1.5 border-b border-border-soft px-[22px] py-2">
      {active.map((f) => (
        <FieldFilterChip
          key={f.key}
          label={f.label.toLowerCase()}
          value={String(query[f.key])}
          onRemove={() => setQuery({ [f.key]: '' } as Partial<ListMessagesParams>)}
        />
      ))}
    </div>
  )
}
