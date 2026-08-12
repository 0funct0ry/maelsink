import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useMessageStore } from '../../stores/useMessageStore'

export default function Pagination() {
  const offset = useMessageStore((state) => state.offset)
  const limit = useMessageStore((state) => state.limit)
  const total = useMessageStore((state) => state.total)
  const setPage = useMessageStore((state) => state.setPage)

  const prevDisabled = offset === 0
  const nextDisabled = offset + limit >= total

  const rangeStart = total === 0 ? 0 : offset + 1
  const rangeEnd = Math.min(offset + limit, total)

  function handlePrev() {
    setPage(Math.max(0, offset - limit))
  }

  function handleNext() {
    setPage(offset + limit)
  }

  return (
    <div className="flex items-center justify-between border-t border-border-soft px-4 py-3 text-sm text-text-secondary">
      <span>
        {rangeStart}-{rangeEnd} of {total}
      </span>
      <div className="flex items-center gap-2">
        <button
          type="button"
          aria-label="Previous page"
          disabled={prevDisabled}
          onClick={handlePrev}
          className="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-secondary transition-colors hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          type="button"
          aria-label="Next page"
          disabled={nextDisabled}
          onClick={handleNext}
          className="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-secondary transition-colors hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </div>
  )
}
