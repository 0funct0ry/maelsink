import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

interface ModalProps {
  open: boolean
  onClose: () => void
  children: ReactNode
  maxWidthClass?: string
}

// Ported from web/src/components/common/Modal.tsx (trimmed to the dialog
// variant only — compose has no mobile drawer nav in this milestone).
export default function Modal({ open, onClose, children, maxWidthClass = 'max-w-md' }: ModalProps) {
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    cardRef.current?.focus()
  }, [open])

  useEffect(() => {
    if (!open) return
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-text-primary/40"
      onClick={onClose}
    >
      <div
        ref={cardRef}
        tabIndex={-1}
        className={`w-full ${maxWidthClass} rounded-lg bg-bg p-6 shadow-lg`}
        onClick={(event) => event.stopPropagation()}
      >
        {children}
      </div>
    </div>,
    document.body,
  )
}
