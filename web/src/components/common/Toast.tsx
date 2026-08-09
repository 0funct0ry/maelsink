import { useEffect } from 'react'
import { X } from 'lucide-react'

export type ToastVariant = 'info' | 'success' | 'danger'

interface ToastProps {
  variant: ToastVariant
  message: string
  onDismiss: () => void
}

const AUTO_DISMISS_MS = 4000

const variantClasses: Record<ToastVariant, string> = {
  info: 'border-border text-text-secondary',
  success: 'border-success bg-success-soft text-success',
  danger: 'border-danger bg-danger-soft text-danger',
}

export default function Toast({ variant, message, onDismiss }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onDismiss()
    }, AUTO_DISMISS_MS)
    return () => clearTimeout(timer)
  }, [onDismiss])

  return (
    <div
      role="status"
      className={`flex items-center gap-3 rounded-md border-l-4 bg-surface px-4 py-3 text-sm shadow-md ${variantClasses[variant]}`}
    >
      <span className="flex-1">{message}</span>
      <button
        type="button"
        aria-label="Dismiss"
        onClick={onDismiss}
        className="rounded-sm p-1 text-text-tertiary hover:bg-surface-2"
      >
        <X className="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  )
}
