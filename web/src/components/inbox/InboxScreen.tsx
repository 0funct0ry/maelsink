import { ChevronDown, Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { exportAllUrl } from '../../lib/apiClient'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import ActiveFilterChips from './ActiveFilterChips'
import FieldFilterBar from './FieldFilterBar'
import MessageList from './MessageList'
import MessagePreviewModal from './MessagePreviewModal'
import Pagination from './Pagination'

export default function InboxScreen() {
  const navigate = useNavigate()
  const [previewId, setPreviewId] = useState<string | null>(null)
  const fetchMessages = useMessageStore((state) => state.fetchMessages)
  const sort = useMessageStore((state) => state.query.sort ?? 'received_at_desc')
  const query = useMessageStore((state) => state.query)
  const setQuery = useMessageStore((state) => state.setQuery)
  const [sortOpen, setSortOpen] = useState(false)

  useEffect(() => {
    void fetchMessages()
  }, [fetchMessages])

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-border-soft px-[22px] py-3">
        <div>
          <h1 className="text-[17px] font-semibold tracking-[-0.01em] text-text-primary">Messages</h1>
        </div>
        <div className="flex items-center gap-2">
          <FieldFilterBar />
          <div className="relative">
            <button
              type="button"
              onClick={() => setSortOpen((v) => !v)}
              className="flex items-center gap-1.5 rounded-sm border border-border-soft bg-surface px-2.5 py-1.5 text-[12.5px] text-text-secondary"
            >
              <ChevronDown className="h-[13px] w-[13px]" aria-hidden="true" />
              Sort: {sort === 'received_at_asc' ? 'Oldest' : 'Newest'}
            </button>
            {sortOpen && (
              <div className="absolute right-0 top-full z-10 mt-1 w-32 rounded-md border border-border bg-bg py-1 shadow-md">
                {(
                  [
                    ['received_at_desc', 'Newest'],
                    ['received_at_asc', 'Oldest'],
                  ] as const
                ).map(([value, label]) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => {
                      setQuery({ sort: value })
                      setSortOpen(false)
                    }}
                    className="block w-full px-3 py-1.5 text-left text-[12.5px] text-text-secondary hover:bg-surface"
                  >
                    {label}
                  </button>
                ))}
              </div>
            )}
          </div>
          <a
            href={exportAllUrl(query)}
            download
            title="Export all messages as .zip"
            onClick={() => useUIStore.getState().pushToast('info', 'Export started')}
            className="flex h-8 w-8 items-center justify-center rounded-sm text-text-secondary transition-colors hover:bg-surface hover:text-text-primary"
          >
            <Download className="h-[17px] w-[17px]" aria-hidden="true" />
          </a>
        </div>
      </div>

      <ActiveFilterChips />

      <div className="scrollbar-thin flex-1 overflow-y-auto overflow-x-hidden">
        <MessageList onOpenMessage={(id) => navigate(`/messages/${id}`)} onPreviewMessage={setPreviewId} />
      </div>

      <Pagination />

      <MessagePreviewModal messageId={previewId} onClose={() => setPreviewId(null)} />
    </div>
  )
}
