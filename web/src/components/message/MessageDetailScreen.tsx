import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Download, ListTree, Tags, Trash2 } from 'lucide-react'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import { exportMessage } from '../../lib/apiClient'
import { formatAbsoluteDate, formatAddress, formatAddressList, formatExactTime, formatRelativeTime } from '../../lib/format'
import Button from '../common/Button'
import ConfirmDialog from '../common/ConfirmDialog'
import TagBadge from '../common/TagBadge'
import TagEditModal from '../common/TagEditModal'
import StatusBadges from './StatusBadges'
import MessageTabs from './MessageTabs'
import AttachmentGrid from './AttachmentGrid'
import DeletedMessageState from './DeletedMessageState'

export default function MessageDetailScreen() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { selected, selectedStatus, fetchMessage, markRead, deleteMessageOptimistic, clearSelected } =
    useMessageStore()
  const markedReadFor = useRef<string | null>(null)
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [tagEditOpen, setTagEditOpen] = useState(false)

  useEffect(() => {
    if (!id) return
    void fetchMessage(id)
    return () => clearSelected()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  useEffect(() => {
    if (!selected || selected.id !== id) return
    if (selected.read) return
    if (markedReadFor.current === selected.id) return
    markedReadFor.current = selected.id
    void markRead(selected.id)
  }, [selected, id, markRead])

  const handleDelete = async () => {
    if (!id) return
    setDeleting(true)
    await deleteMessageOptimistic(id)
    setDeleting(false)
    navigate('/')
  }

  const handleExport = async () => {
    if (!id) return
    setExporting(true)
    try {
      const blob = await exportMessage(id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${id}.eml`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      useUIStore.getState().pushToast('success', 'Message exported')
    } catch {
      useUIStore.getState().pushToast('danger', 'Failed to export message')
    } finally {
      setExporting(false)
    }
  }

  if (!selected) {
    if (selectedStatus === 'not_found') {
      return <DeletedMessageState messageId={id} />
    }

    if (selectedStatus === 'error') {
      return (
        <div className="flex h-full flex-col items-center justify-center gap-4 p-12 text-center">
          <h2 className="text-lg font-semibold text-text-primary">Something went wrong loading this message</h2>
          <p className="text-sm text-text-secondary">Please try again, or go back to the inbox.</p>
          <Button variant="secondary" onClick={() => navigate('/')}>
            Back to Inbox
          </Button>
        </div>
      )
    }

    return (
      <div className="p-6">
        <div className="mb-4 h-6 w-48 animate-pulse rounded bg-surface-2" />
        <div className="mb-2 h-4 w-full max-w-md animate-pulse rounded bg-surface-2" />
        <div className="h-4 w-full max-w-sm animate-pulse rounded bg-surface-2" />
      </div>
    )
  }

  const message = selected

  return (
    <div className="scrollbar-thin flex h-full flex-col overflow-y-auto overflow-x-hidden">
      <div className="flex flex-none items-center gap-3.5 border-b border-border-soft px-6 py-3.5">
        <button
          type="button"
          onClick={() => navigate('/')}
          className="flex flex-none items-center gap-1.5 rounded-sm px-2 py-1.5 text-[13px] font-medium text-text-secondary transition-colors hover:bg-surface hover:text-text-primary"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to inbox
        </button>
        <div className="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            disabled={exporting}
            onClick={() => void handleExport()}
            className="flex items-center gap-1.5 rounded-sm border border-border-soft bg-bg px-[11px] py-[7px] text-[12.8px] font-medium text-text-secondary transition-colors hover:border-border hover:bg-surface hover:text-text-primary disabled:opacity-60"
          >
            <Download className="h-3.5 w-3.5" aria-hidden="true" />
            Export .eml
          </button>
          <button
            type="button"
            disabled={deleting}
            onClick={() => setConfirmOpen(true)}
            className="flex items-center gap-1.5 rounded-sm border border-border-soft bg-bg px-[11px] py-[7px] text-[12.8px] font-medium text-danger transition-colors hover:border-danger hover:bg-danger-soft disabled:opacity-60"
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
            Delete
          </button>
        </div>
      </div>

      <div className="flex-none border-b border-border-soft px-6 py-4">
        <h1 className="mb-3 text-[19px] font-semibold tracking-tight text-text-primary">
          {message.subject || '(no subject)'}
        </h1>
        <div className="flex flex-col gap-1.5">
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-11 flex-none text-text-tertiary">From</span>
            <span className="break-all font-mono text-[12.6px] text-text-primary">
              {formatAddress(message.from, message.from_name)}
            </span>
          </div>
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-11 flex-none text-text-tertiary">To</span>
            <span className="break-all font-mono text-[12.6px] text-text-primary">
              {formatAddressList(message.to, message.to_names)}
            </span>
          </div>
          {message.cc.length > 0 && (
            <div className="flex gap-2.5 text-[13px]">
              <span className="w-11 flex-none text-text-tertiary">Cc</span>
              <span className="break-all font-mono text-[12.6px] text-text-primary">
                {formatAddressList(message.cc, message.cc_names)}
              </span>
            </div>
          )}
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-11 flex-none text-text-tertiary">Date</span>
            <span
              className="break-all font-mono text-[12.6px] text-text-primary"
              title={formatExactTime(message.received_at)}
            >
              {formatRelativeTime(message.received_at)}
            </span>
            <span className="break-all font-mono text-[12.6px] text-text-tertiary">
              ({formatAbsoluteDate(message.received_at)})
            </span>
          </div>
        </div>
        <div className="mt-2.5 flex flex-wrap items-center gap-2.5">
          <StatusBadges message={message} />
          {message.tags.length > 0 && (
            <span className="flex flex-wrap items-center gap-1.5">
              {message.tags.map((tag) => (
                <TagBadge key={tag} tag={tag} />
              ))}
            </span>
          )}
          <button
            type="button"
            onClick={() => setTagEditOpen(true)}
            className="flex items-center gap-1.5 rounded-sm border border-border-soft bg-bg px-[9px] py-[5px] text-[12px] font-medium text-text-secondary transition-colors hover:border-border hover:bg-surface hover:text-text-primary"
          >
            <Tags className="h-3.5 w-3.5" aria-hidden="true" />
            Edit tags
          </button>
          {message.session_id && (
            <button
              type="button"
              onClick={() => navigate(`/sessions/${message.session_id}`)}
              className="flex items-center gap-1.5 rounded-sm border border-border-soft bg-bg px-[9px] py-[5px] text-[12px] font-medium text-text-secondary transition-colors hover:border-border hover:bg-surface hover:text-text-primary"
            >
              <ListTree className="h-3.5 w-3.5" aria-hidden="true" />
              View session
            </button>
          )}
        </div>
      </div>

      <div className="px-6 py-5">
        <MessageTabs message={message} />
      </div>

      <div className="px-6 pb-6">
        <AttachmentGrid messageId={message.id} attachments={message.attachments} />
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => void handleDelete()}
        title="Delete this message?"
        body="This message will be permanently deleted. This action cannot be undone."
        confirmLabel="Delete"
        danger
      />

      <TagEditModal open={tagEditOpen} onClose={() => setTagEditOpen(false)} message={message} />
    </div>
  )
}
