import type { ReactNode } from 'react'

type BadgeVariant =
  | 'default'
  | 'success'
  | 'warning'
  | 'danger'
  | 'accent'
  | 'source-default'
  | 'source-file'
  | 'source-env'
  | 'source-flag'

interface BadgeProps {
  variant?: BadgeVariant
  children: ReactNode
}

const variantClasses: Record<BadgeVariant, string> = {
  default: 'bg-surface-2 text-text-secondary',
  success: 'bg-success-soft text-success',
  warning: 'bg-warning-soft text-warning',
  danger: 'bg-danger-soft text-danger',
  accent: 'bg-accent-soft text-accent',
  // Source badges mirror PrecedenceChain's default/file/env/flag swatch
  // colors (M6.1's Settings screen provenance table), so a value's badge
  // color always matches the same layer's color in the precedence legend.
  'source-default': 'bg-surface-2 text-text-secondary',
  'source-file': 'bg-[#e8f0fd] text-[#1a63d6]',
  'source-env': 'bg-[#e4f7ef] text-[#0f8a63]',
  'source-flag': 'bg-accent-soft text-accent',
}

export default function Badge({ variant = 'default', children }: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${variantClasses[variant]}`}
    >
      {children}
    </span>
  )
}
