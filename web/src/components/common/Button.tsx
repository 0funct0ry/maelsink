import { Loader2 } from 'lucide-react'
import type { ReactNode } from 'react'

type ButtonVariant = 'primary' | 'secondary' | 'danger' | 'ghost'

interface ButtonProps {
  variant?: ButtonVariant
  loading?: boolean
  disabled?: boolean
  children: ReactNode
  onClick?: () => void
  type?: 'button' | 'submit'
}

const variantClasses: Record<ButtonVariant, string> = {
  primary: 'bg-accent text-white hover:bg-accent-hover',
  secondary: 'bg-surface-2 text-text-primary hover:bg-surface',
  danger: 'bg-danger text-white hover:bg-danger/90',
  ghost: 'bg-transparent text-text-secondary hover:bg-surface',
}

export default function Button({
  variant = 'primary',
  loading = false,
  disabled = false,
  children,
  onClick,
  type = 'button',
}: ButtonProps) {
  const isDisabled = disabled || loading

  return (
    <button
      type={type}
      disabled={isDisabled}
      onClick={() => {
        if (isDisabled) return
        onClick?.()
      }}
      className={`inline-flex items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${variantClasses[variant]}`}
    >
      {loading && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
      {children}
    </button>
  )
}
