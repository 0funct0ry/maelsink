import { useEffect } from 'react'
import { X } from 'lucide-react'

export type ToastVariant = 'info' | 'success' | 'danger'

interface ToastProps {
  variant: ToastVariant
  message: string
  onDismiss: () => void
}

const AUTO_DISMISS_MS = 4000

// Uses the same soft-background/solid-text/solid-border variant colors as
// Badge (bg-*-soft + text-* + border-*), so a toast's color always matches
// what the same variant looks like everywhere else in the app, instead of a
// one-off left-border-only treatment.
const variantClasses: Record<ToastVariant, string> = {
  info: 'border-accent bg-accent-soft text-accent',
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
      className={`flex w-80 items-center gap-3 border-l-4 px-4 py-3 text-sm shadow-lg ${variantClasses[variant]}`}
    >
      <span className="flex-1">{message}</span>
      <button
        type="button"
        aria-label="Dismiss"
        onClick={onDismiss}
        className="p-1 text-current opacity-70 hover:opacity-100"
      >
        <X className="h-4 w-4" aria-hidden="true" />
      </button>
    </div>
  )
}
