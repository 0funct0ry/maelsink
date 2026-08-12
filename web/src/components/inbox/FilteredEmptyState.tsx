import { SearchX } from 'lucide-react'

// Shown when the current filters/search match zero messages, distinct from
// EmptyState's first-run "no messages in the whole mailbox yet" copy
// (M8.7) — this case doesn't need the SMTP connection string since the
// mailbox already has mail, just none matching the active query.
export default function FilteredEmptyState() {
  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      <SearchX className="h-10 w-10 text-text-tertiary" aria-hidden="true" />
      <h2 className="text-base font-semibold text-text-primary">No matching messages</h2>
      <p className="text-sm text-text-secondary">
        No messages match the current filters or search. Try broadening your search or clearing a filter.
      </p>
    </div>
  )
}
