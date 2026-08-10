import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Download, Eye, Mail, MailOpen, MoreVertical, Trash2 } from 'lucide-react'
import IconButton from '../common/IconButton'
import { exportMessage } from '../../lib/apiClient'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import type { MessageSummary } from '../../lib/apiTypes'

interface MessageRowActionsProps {
  message: MessageSummary
  onPreview: () => void
}

interface MenuPosition {
  top: number
  right: number
}

// Per-row action menu (SPEC.md §8.1 requires per-row delete; the mockup has
// no equivalent, so this is a deliberate addition beyond it) replacing the
// bare hover-delete icon with a dropdown covering every row-level action:
// preview, mark read/unread, export, delete. Delete still goes through a
// confirmation dialog (a per-row destructive action inside a menu is easy to
// mis-click, unlike the always-visible optimistic delete this replaces).
//
// The menu is rendered through a portal into document.body, positioned by
// the trigger button's on-screen coordinates, rather than as an
// absolutely-positioned child of the row. Every row shares the same
// `relative` stacking context at the same z-level, so a menu positioned
// *inside* one row painted behind a later sibling row in DOM order — the
// portal sidesteps that stacking bug entirely and also avoids the row's own
// `overflow`/clipping.
export default function MessageRowActions({ message, onPreview }: MessageRowActionsProps) {
  const [open, setOpen] = useState(false)
  const [position, setPosition] = useState<MenuPosition | null>(null)
  const [exporting, setExporting] = useState(false)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const markRead = useMessageStore((state) => state.markRead)
  const deleteMessageOptimistic = useMessageStore((state) => state.deleteMessageOptimistic)
  const openConfirm = useUIStore((state) => state.openConfirm)

  function handleDeleteClick() {
    openConfirm({
      title: 'Delete this message?',
      body: 'This message will be permanently deleted. This action cannot be undone.',
      confirmLabel: 'Delete',
      danger: true,
      onConfirm: () => void deleteMessageOptimistic(message.id),
    })
  }

  function handleToggle() {
    if (!open && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      setPosition({ top: rect.bottom + 4, right: window.innerWidth - rect.right })
    }
    setOpen((v) => !v)
  }

  useEffect(() => {
    if (!open) return
    function handlePointerDown(e: MouseEvent) {
      const target = e.target as Node
      if (menuRef.current?.contains(target) || triggerRef.current?.contains(target)) return
      setOpen(false)
    }
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    function handleReposition() {
      setOpen(false)
    }
    document.addEventListener('mousedown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    // Scroll position is computed once at open time; if the list (or the
    // page) scrolls or resizes while open, close rather than show a
    // stale/misaligned menu.
    window.addEventListener('scroll', handleReposition, true)
    window.addEventListener('resize', handleReposition)
    return () => {
      document.removeEventListener('mousedown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('scroll', handleReposition, true)
      window.removeEventListener('resize', handleReposition)
    }
  }, [open])

  function closeAnd(fn: () => void) {
    setOpen(false)
    fn()
  }

  async function handleExport() {
    setExporting(true)
    try {
      const blob = await exportMessage(message.id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${message.id}.eml`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      useUIStore.getState().pushToast('success', 'Export started')
    } catch {
      useUIStore.getState().pushToast('danger', 'Failed to export message')
    } finally {
      setExporting(false)
    }
  }

  return (
    <span ref={triggerRef} onClick={(e) => e.stopPropagation()} className="inline-flex">
      <IconButton icon={<MoreVertical className="h-4 w-4" />} aria-label="Message actions" onClick={handleToggle} />

      {open &&
        position &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            onClick={(e) => e.stopPropagation()}
            style={{ top: position.top, right: position.right }}
            className="fixed z-50 w-44 rounded-md border border-border bg-bg py-1 shadow-lg"
          >
            <button
              type="button"
              role="menuitem"
              onClick={() => closeAnd(onPreview)}
              className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-[13px] text-text-primary hover:bg-surface"
            >
              <Eye className="h-3.5 w-3.5" aria-hidden="true" />
              Preview
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => closeAnd(() => void markRead(message.id, !message.read))}
              className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-[13px] text-text-primary hover:bg-surface"
            >
              {message.read ? (
                <Mail className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <MailOpen className="h-3.5 w-3.5" aria-hidden="true" />
              )}
              Mark as {message.read ? 'unread' : 'read'}
            </button>
            <button
              type="button"
              role="menuitem"
              disabled={exporting}
              onClick={() => closeAnd(() => void handleExport())}
              className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-[13px] text-text-primary hover:bg-surface disabled:opacity-60"
            >
              <Download className="h-3.5 w-3.5" aria-hidden="true" />
              Export .eml
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => closeAnd(handleDeleteClick)}
              className="flex w-full items-center gap-2.5 px-3 py-2 text-left text-[13px] text-danger hover:bg-danger-soft"
            >
              <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              Delete
            </button>
          </div>,
          document.body,
        )}
    </span>
  )
}
