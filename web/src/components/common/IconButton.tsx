import type { ReactNode } from 'react'

type IconButtonVariant = 'default' | 'danger'
type IconButtonSize = 'sm' | 'md'

interface IconButtonProps {
  icon: ReactNode
  // Required (not optional) — an icon-only button with no accessible name
  // is a TS-enforced mistake, not a runtime one.
  'aria-label': string
  onClick?: () => void
  variant?: IconButtonVariant
  size?: IconButtonSize
}

const variantClasses: Record<IconButtonVariant, string> = {
  default: 'text-text-secondary hover:bg-surface',
  danger: 'text-danger hover:bg-danger-soft',
}

const sizeClasses: Record<IconButtonSize, string> = {
  sm: 'h-7 w-7',
  md: 'h-8 w-8',
}

export default function IconButton({
  icon,
  'aria-label': ariaLabel,
  onClick,
  variant = 'default',
  size = 'md',
}: IconButtonProps) {
  return (
    <span className="group relative inline-flex">
      <button
        type="button"
        aria-label={ariaLabel}
        onClick={onClick}
        className={`inline-flex items-center justify-center rounded-md transition-colors ${sizeClasses[size]} ${variantClasses[variant]}`}
      >
        {icon}
      </button>
      {/* Visible hover tooltip, not just the aria-label screen readers get
          (M8.7) — CSS-only via group-hover so no extra state/listeners. */}
      <span
        role="tooltip"
        className="pointer-events-none absolute left-1/2 top-full z-10 mt-1.5 -translate-x-1/2 whitespace-nowrap rounded-md bg-text-primary px-2 py-1 text-xs text-bg opacity-0 shadow-md transition-opacity delay-300 group-hover:opacity-100"
      >
        {ariaLabel}
      </span>
    </span>
  )
}
