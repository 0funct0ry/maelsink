import { AlertTriangle, Check, Paperclip } from 'lucide-react'
import { formatBytes } from '../../lib/format'

interface StatusBadgesProps {
  message: {
    parse_warning: boolean
    has_attachments: boolean
    attachment_count: number
    size_bytes: number
  }
}

const badgeBase = 'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold'

export default function StatusBadges({ message }: StatusBadgesProps) {
  const { parse_warning, has_attachments, attachment_count, size_bytes } = message

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className={`${badgeBase} bg-success-soft text-success`}>
        <Check className="h-[11px] w-[11px]" aria-hidden="true" />
        Captured
      </span>
      <span className={`${badgeBase} border border-border-soft bg-surface text-text-secondary`}>
        {formatBytes(size_bytes)}
      </span>
      {has_attachments && (
        <span className={`${badgeBase} border border-border-soft bg-surface text-text-secondary`}>
          <Paperclip className="h-[11px] w-[11px]" aria-hidden="true" />
          {attachment_count} attachment{attachment_count === 1 ? '' : 's'}
        </span>
      )}
      {parse_warning && (
        <span className={`${badgeBase} bg-warning-soft text-warning`}>
          <AlertTriangle className="h-[11px] w-[11px]" aria-hidden="true" />
          Parse warning
        </span>
      )}
    </div>
  )
}
