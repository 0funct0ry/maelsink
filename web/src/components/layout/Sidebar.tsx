import { Inbox, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import { getStats } from '../../lib/apiClient'
import { formatBytes } from '../../lib/format'
import type { Stats } from '../../lib/apiTypes'

// Per MOCKUP.html's sidebar: a "Mailbox" nav group, a storage-usage card,
// and a destructive "Clear all messages" action pinned to the bottom.
// Settings moved to the TopBar's gear icon (mockup has no Settings item
// here). Sidebar filters beyond "All messages" (Unread/Attachments/
// Warnings), saved searches, and tags are M6.1 — they need backend support
// (live counts, tag storage) that doesn't exist yet.
export default function Sidebar() {
  const total = useMessageStore((state) => state.total)
  const openConfirm = useUIStore((state) => state.openConfirm)
  const [stats, setStats] = useState<Stats | null>(null)

  useEffect(() => {
    let cancelled = false
    getStats()
      .then((s) => {
        if (!cancelled) setStats(s)
      })
      .catch(() => {
        // Non-fatal: the storage card just doesn't render.
      })
    return () => {
      cancelled = true
    }
  }, [])

  function handleClearAll() {
    openConfirm({
      title: 'Clear all messages?',
      body: 'This will permanently delete every message in the inbox. This action cannot be undone.',
      confirmLabel: 'Clear all',
      danger: true,
      onConfirm: () => {
        void useMessageStore.getState().clearAll()
      },
    })
  }

  return (
    <aside className="flex w-[216px] flex-none flex-col gap-[22px] overflow-y-auto border-r border-border bg-bg p-3">
      <div>
        <div className="mb-1.5 px-2 text-[11px] font-semibold uppercase tracking-[0.04em] text-text-tertiary">
          Mailbox
        </div>
        <nav className="flex flex-col gap-px">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              `flex items-center justify-between gap-2.5 rounded-sm px-2 py-[7px] text-[13.5px] font-medium transition-colors ${
                isActive ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface hover:text-text-primary'
              }`
            }
          >
            {({ isActive }) => (
              <>
                <span className="flex items-center gap-2.5">
                  <Inbox className={`h-4 w-4 ${isActive ? 'opacity-100' : 'opacity-85'}`} aria-hidden="true" />
                  All messages
                </span>
                <span
                  className={`rounded-[5px] px-1.5 font-mono text-[11.5px] ${
                    isActive ? 'bg-white text-accent' : 'bg-surface text-text-tertiary'
                  }`}
                >
                  {total}
                </span>
              </>
            )}
          </NavLink>
        </nav>
      </div>

      {stats && (
        <div className="flex flex-col gap-2 rounded-md border border-border-soft bg-surface p-3">
          <div className="flex items-baseline justify-between">
            <span className="text-[11.5px] text-text-tertiary">Storage used</span>
            <span className="font-mono text-xs font-medium text-text-primary">
              {formatBytes(stats.total_size_bytes)}
            </span>
          </div>
          {/* Decorative only — no configured storage limit is exposed via
              any endpoint today, so this can't reflect a real percentage. */}
          <div className="h-[5px] overflow-hidden rounded-[3px] bg-border-soft">
            <div className="h-full w-[12%] rounded-[3px] bg-accent" />
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-[11.5px] text-text-tertiary">maelsink.db</span>
            <span className="font-mono text-xs font-medium text-text-primary">{stats.total_messages} msgs</span>
          </div>
        </div>
      )}

      <div className="mt-auto">
        <button
          type="button"
          onClick={handleClearAll}
          className="flex w-full items-center gap-2 rounded-sm border border-transparent px-2.5 py-2 text-[13px] font-medium text-danger transition-colors hover:border-[#f6c9d1] hover:bg-danger-soft"
        >
          <Trash2 className="h-[15px] w-[15px]" aria-hidden="true" />
          Clear all messages
        </button>
      </div>
    </aside>
  )
}
