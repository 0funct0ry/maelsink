import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

interface ModalProps {
  open: boolean
  onClose: () => void
  children: ReactNode
  maxWidthClass?: string
  /** When false, Escape and backdrop clicks do not close the modal — used
   * for mandatory gates like ApiKeyModal. Defaults to true. */
  dismissable?: boolean
  /** 'dialog' (default): centered card. 'drawer': full-height panel sliding
   * in from the left, for the mobile navigation drawer (M8.7) — reuses the
   * same backdrop/Escape/focus-trap wiring below rather than a separate
   * component. */
  variant?: 'dialog' | 'drawer'
}

export default function Modal({
  open,
  onClose,
  children,
  maxWidthClass = 'max-w-md',
  dismissable = true,
  variant = 'dialog',
}: ModalProps) {
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    cardRef.current?.focus()
  }, [open])

  useEffect(() => {
    if (!open || !dismissable) return
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, dismissable, onClose])

  if (!open) return null

  const isDrawer = variant === 'drawer'

  // Portaled to document.body rather than rendered inline wherever the
  // caller happens to sit in the tree: a `fixed` element is only fixed to
  // the viewport if every ancestor lacks a transform/filter/etc that would
  // make it a containing block instead — e.g. MessageRow's `animate-row-in`
  // (a transform-based animation) does exactly that, which shrank this
  // backdrop down to that row's box instead of the full screen. Portaling
  // sidesteps the whole class of bug regardless of what future ancestor
  // might introduce a containing block.
  return createPortal(
    <div
      className={`fixed inset-0 z-50 flex bg-text-primary/40 ${isDrawer ? 'items-stretch justify-start' : 'items-center justify-center'}`}
      onClick={() => {
        if (dismissable) onClose()
      }}
    >
      <div
        ref={cardRef}
        tabIndex={-1}
        className={
          isDrawer
            ? `scrollbar-thin h-full w-full ${maxWidthClass} overflow-y-auto overflow-x-hidden bg-bg shadow-lg`
            : `w-full ${maxWidthClass} rounded-lg bg-bg p-6 shadow-lg`
        }
        onClick={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  )
}
