import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Mail, Trash2 } from 'lucide-react'
import { useSessionStore } from '../../stores/useSessionStore'
import { useUIStore } from '../../stores/useUIStore'
import { formatExactTime, formatRelativeTime } from '../../lib/format'
import Badge from '../common/Badge'
import IconButton from '../common/IconButton'

const STATUS_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger'> = {
  completed: 'success',
  aborted: 'warning',
  timeout: 'warning',
  rejected: 'danger',
}

function StatusPill({ status }: { status: string }) {
  if (!status) {
    return <Badge>In progress</Badge>
  }
  return <Badge variant={STATUS_VARIANT[status] ?? 'default'}>{status}</Badge>
}

export default function SessionsScreen() {
  const navigate = useNavigate()
  const sessions = useSessionStore((state) => state.sessions)
  const listStatus = useSessionStore((state) => state.listStatus)
  const listError = useSessionStore((state) => state.listError)
  const fetchSessions = useSessionStore((state) => state.fetchSessions)
  const offset = useSessionStore((state) => state.offset)
  const limit = useSessionStore((state) => state.limit)
  const total = useSessionStore((state) => state.total)
  const setPage = useSessionStore((state) => state.setPage)
  const deleteSessionOptimistic = useSessionStore((state) => state.deleteSessionOptimistic)
  const clearAll = useSessionStore((state) => state.clearAll)
  const openConfirm = useUIStore((state) => state.openConfirm)

  useEffect(() => {
    void fetchSessions()
  }, [fetchSessions])

  const rangeStart = total === 0 ? 0 : offset + 1
  const rangeEnd = Math.min(offset + limit, total)

  function handleDeleteClick(id: string) {
    openConfirm({
      title: 'Delete this session?',
      body: 'This session and its protocol transcript will be permanently deleted. Any message it produced is not affected. This action cannot be undone.',
      confirmLabel: 'Delete',
      danger: true,
      onConfirm: () => void deleteSessionOptimistic(id),
    })
  }

  function handleClearAllClick() {
    openConfirm({
      title: 'Clear all sessions?',
      body: 'This will permanently delete every SMTP session and its transcript. Messages already stored are not affected. This action cannot be undone.',
      confirmLabel: 'Clear all',
      danger: true,
      onConfirm: () => void clearAll(),
    })
  }

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-border-soft px-[22px] py-3">
        <h1 className="text-[17px] font-semibold tracking-[-0.01em] text-text-primary">Sessions</h1>
        <IconButton
          icon={<Trash2 className="h-[17px] w-[17px]" aria-hidden="true" />}
          aria-label="Clear all sessions"
          variant="danger"
          onClick={handleClearAllClick}
        />
      </div>

      <div className="flex-1 overflow-y-auto">
        {listStatus === 'loading' && (
          <div className="flex flex-col gap-2 p-4">
            {[0, 1, 2, 3, 4].map((i) => (
              <div key={i} className="h-10 animate-pulse rounded-md bg-surface-2" />
            ))}
          </div>
        )}

        {listStatus === 'error' && (
          <div className="p-6 text-sm text-danger">{listError?.message ?? 'Failed to load sessions.'}</div>
        )}

        {listStatus !== 'loading' && listStatus !== 'error' && sessions.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-2 p-12 text-center text-text-secondary">
            <p className="text-sm">No SMTP sessions yet.</p>
            <p className="text-xs text-text-tertiary">Sessions appear here as soon as a client connects.</p>
          </div>
        )}

        {listStatus !== 'loading' && listStatus !== 'error' && sessions.length > 0 && (
          <div>
            {sessions.map((session) => (
              <div
                key={session.id}
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/sessions/${session.id}`)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') navigate(`/sessions/${session.id}`)
                }}
                className="grid cursor-pointer grid-cols-[1fr_1fr_auto_auto_auto] items-center gap-4 border-b border-border-soft px-[22px] py-3 text-sm transition-colors hover:bg-surface"
              >
                <span
                  className="truncate font-mono text-[12.6px] text-text-primary"
                  title={formatExactTime(session.started_at)}
                >
                  {formatRelativeTime(session.started_at)}
                </span>
                <span className="truncate font-mono text-[12.6px] text-text-secondary">
                  {session.client_ip}
                  {session.client_helo ? ` (${session.client_helo})` : ''}
                </span>
                <StatusPill status={session.status} />
                {session.message_id ? (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      navigate(`/messages/${session.message_id}`)
                    }}
                    className="flex items-center gap-1.5 rounded-sm px-2 py-1 text-[12px] font-medium text-accent hover:bg-accent-soft"
                  >
                    <Mail className="h-3.5 w-3.5" aria-hidden="true" />
                    View message
                  </button>
                ) : (
                  <span />
                )}
                <span onClick={(e) => e.stopPropagation()}>
                  <IconButton
                    icon={<Trash2 className="h-4 w-4" aria-hidden="true" />}
                    aria-label={`Delete session ${session.id}`}
                    variant="danger"
                    size="sm"
                    onClick={() => handleDeleteClick(session.id)}
                  />
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="flex items-center justify-between border-t border-border-soft px-4 py-3 text-sm text-text-secondary">
        <span>
          {rangeStart}-{rangeEnd} of {total}
        </span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            aria-label="Previous page"
            disabled={offset === 0}
            onClick={() => setPage(Math.max(0, offset - limit))}
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-secondary transition-colors hover:bg-surface disabled:cursor-not-allowed disabled:opacity-40"
          >
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label="Next page"
            disabled={offset + limit >= total}
            onClick={() => setPage(offset + limit)}
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-text-secondary transition-colors hover:bg-surface disabled:cursor-not-allowed disabled:opacity-40"
          >
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </div>
    </div>
  )
}
