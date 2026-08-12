import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Modal from '../common/Modal'
import Button from '../common/Button'
import MessageTabs from '../message/MessageTabs'
import { getMessage } from '../../lib/apiClient'
import type { MessageDetail } from '../../lib/apiTypes'

interface MessagePreviewModalProps {
  messageId: string | null
  onClose: () => void
}

// Lets a message be inspected from the Inbox row menu without leaving the
// list (unlike opening the row, which navigates to /messages/:id). Reuses
// MessageTabs so the preview gets the same sandboxed-HTML/plain-text/raw/
// headers behavior as the full Message Detail screen, just fetched fresh
// each time it opens rather than sharing useMessageStore's `selected` state.
export default function MessagePreviewModal({ messageId, onClose }: MessagePreviewModalProps) {
  const navigate = useNavigate()
  const [message, setMessage] = useState<MessageDetail | null>(null)
  const [error, setError] = useState(false)

  useEffect(() => {
    if (!messageId) {
      setMessage(null)
      setError(false)
      return
    }
    let cancelled = false
    getMessage(messageId)
      .then((detail) => {
        if (!cancelled) setMessage(detail)
      })
      .catch(() => {
        if (!cancelled) setError(true)
      })
    return () => {
      cancelled = true
    }
  }, [messageId])

  return (
    <Modal open={messageId !== null} onClose={onClose} maxWidthClass="max-w-2xl">
      {error && (
        <div className="py-6 text-center text-sm text-text-secondary">Failed to load this message.</div>
      )}
      {!error && !message && (
        <div className="py-6">
          <div className="mb-3 h-5 w-2/3 animate-pulse rounded bg-surface-2" />
          <div className="h-4 w-full animate-pulse rounded bg-surface-2" />
        </div>
      )}
      {message && (
        <div className="scrollbar-thin max-h-[80vh] overflow-y-auto overflow-x-hidden">
          <h2 className="mb-1 text-[17px] font-semibold text-text-primary">{message.subject || '(no subject)'}</h2>
          <p className="mb-4 truncate font-mono text-[12px] text-text-tertiary">
            {message.from} &rarr; {message.to.join(', ')}
          </p>
          <MessageTabs message={message} />
          <div className="mt-5 flex justify-end gap-2.5">
            <Button variant="secondary" onClick={onClose}>
              Close
            </Button>
            <Button variant="primary" onClick={() => navigate(`/messages/${message.id}`)}>
              Open full message
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}
