import { useCallback, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Trash2 } from 'lucide-react'
import { useConnectionStore } from '../stores/useConnectionStore'
import Button from '../components/Button'
import ConfirmModal from '../components/ConfirmModal'
import { clearMessages, deleteMessage, listMessages, type MessageSummary } from '../lib/composeApi'

const PAGE_SIZE = 20

// Message List screen (SPEC.md §7.7.4): newest-first, manual-refresh only —
// no WebSocket/auto-poll for messages in M13.0, only the connection-status
// health poll runs automatically. Paginated via the proxy's limit/offset
// query params.
export default function MessageListScreen() {
  const navigate = useNavigate()
  const status = useConnectionStore((s) => s.status)
  const disabled = status === 'red'

  const [messages, setMessages] = useState<MessageSummary[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [confirmClearOpen, setConfirmClearOpen] = useState(false)
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null)

  const refresh = useCallback(async (atOffset: number) => {
    setLoading(true)
    setError(null)
    try {
      const resp = await listMessages({ limit: PAGE_SIZE, offset: atOffset })
      setMessages(resp.messages)
      setTotal(resp.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh(offset)
  }, [refresh, offset])

  async function handleDeleteConfirmed() {
    if (!pendingDeleteId) return
    await deleteMessage(pendingDeleteId)
    setPendingDeleteId(null)
    void refresh(offset)
  }

  async function handleClearAll() {
    await clearMessages()
    setOffset(0)
    setMessages([])
    setTotal(0)
  }

  const pageStart = total === 0 ? 0 : offset + 1
  const pageEnd = Math.min(offset + PAGE_SIZE, total)
  const hasPrev = offset > 0
  const hasNext = offset + PAGE_SIZE < total

  return (
    <div className="flex h-full flex-col p-4">
      <div className="mb-4 flex items-center justify-between">
        <h1 className="text-lg font-semibold text-text-primary">Messages</h1>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            onClick={() => void refresh(offset)}
            loading={loading}
            disabled={disabled}
            title={disabled ? 'Target unreachable' : undefined}
          >
            Refresh
          </Button>
          <Button
            variant="danger"
            onClick={() => setConfirmClearOpen(true)}
            disabled={disabled || total === 0}
            title={disabled ? 'Target unreachable' : undefined}
          >
            Clear all
          </Button>
        </div>
      </div>

      {error && <p className="mb-3 text-sm text-danger">{error}</p>}

      {messages.length === 0 && !loading ? (
        <p className="text-sm text-text-secondary">No messages yet.</p>
      ) : (
        <ul className="scrollbar-thin flex-1 divide-y divide-border-soft overflow-y-auto">
          {messages.map((m) => (
            <li
              key={m.id}
              onClick={() => navigate(`/messages/${m.id}`)}
              className="flex cursor-pointer items-center justify-between gap-4 px-2 py-3 hover:bg-surface-2"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-text-primary">{m.subject || '(no subject)'}</p>
                <p className="truncate text-xs text-text-secondary">
                  {m.from} → {m.to.join(', ')}
                </p>
              </div>
              <span className="shrink-0 text-xs text-text-tertiary">
                {new Date(m.received_at).toLocaleString()}
              </span>
              <span onClick={(e) => e.stopPropagation()}>
                <button
                  type="button"
                  title="Delete"
                  aria-label="Delete"
                  onClick={() => setPendingDeleteId(m.id)}
                  className="rounded-md p-2 text-text-secondary transition-colors hover:bg-danger-soft hover:text-danger"
                >
                  <Trash2 className="h-4 w-4" aria-hidden="true" />
                </button>
              </span>
            </li>
          ))}
        </ul>
      )}

      {total > 0 && (
        <div className="mt-3 flex items-center justify-between border-t border-border-soft pt-3 text-sm text-text-secondary">
          <span>
            {pageStart}–{pageEnd} of {total}
          </span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
              disabled={!hasPrev}
              aria-label="Previous page"
              className="rounded-md p-1.5 text-text-secondary transition-colors hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronLeft className="h-4 w-4" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={() => setOffset((o) => o + PAGE_SIZE)}
              disabled={!hasNext}
              aria-label="Next page"
              className="rounded-md p-1.5 text-text-secondary transition-colors hover:bg-surface-2 disabled:cursor-not-allowed disabled:opacity-40"
            >
              <ChevronRight className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        </div>
      )}

      <ConfirmModal
        open={confirmClearOpen}
        onClose={() => setConfirmClearOpen(false)}
        onConfirm={() => void handleClearAll()}
        title="Clear all messages?"
        body="This permanently deletes every message on the target. This cannot be undone."
        confirmLabel="Clear all"
        danger
      />

      <ConfirmModal
        open={pendingDeleteId !== null}
        onClose={() => setPendingDeleteId(null)}
        onConfirm={() => void handleDeleteConfirmed()}
        title="Delete this message?"
        body="This permanently deletes the message from the target. This cannot be undone."
        confirmLabel="Delete"
        danger
      />
    </div>
  )
}
