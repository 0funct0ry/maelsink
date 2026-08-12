import { Paperclip } from 'lucide-react'
import Badge from '../common/Badge'
import TagBadge from '../common/TagBadge'
import MessageRowActions from './MessageRowActions'
import { formatBytes, formatExactTime, formatRelativeTime, truncateList } from '../../lib/format'
import type { MessageSummary } from '../../lib/apiTypes'

interface MessageRowProps {
  message: MessageSummary
  onOpen: () => void
  onPreview: () => void
  /** Briefly true right after a realtime message.created event (M7.0), to
   * flash the row so it's obvious a new message just arrived. */
  highlighted?: boolean
  /** Position in the current page, used only to stagger the mount-in
   * animation's delay — capped so a long list doesn't produce an absurdly
   * long wait for rows near the bottom. */
  index?: number
}

// Grid columns mirror MOCKUP.html's .msg-row (unread dot / from / subject
// area / meta / time), with "to" and size folded into the subject/meta
// clusters (SPEC.md §8.1 requires showing both, but the mockup's row
// doesn't have dedicated columns for them) rather than adding extra columns
// that would widen the row beyond the mockup's density. The row actions menu
// trigger gets its own trailing grid column (previously absolutely
// positioned over the time column, which visually overlapped it) and is
// always visible (not a hover-only affordance) since a hidden trigger is
// hard to discover/keyboard-reach (mockup has no per-row actions; SPEC.md
// §8.1 requires delete). The body preview snippet was deliberately dropped
// from the row (kept only in the preview modal/detail screen) to reduce
// list density/clutter — subject + tags is enough to scan a row.
export default function MessageRow({ message, onOpen, onPreview, highlighted, index = 0 }: MessageRowProps) {
  const { shown, more } = truncateList(message.to, 1)
  // Capped so a long list's later rows don't wait an unreasonably long time
  // for their entrance animation.
  const animationDelayMs = Math.min(index, 12) * 35

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onOpen()
      }}
      style={{ animationDelay: `${animationDelayMs}ms` }}
      className={`animate-row-in group grid cursor-pointer grid-cols-[20px_1fr_32px] items-stretch gap-4 border-b border-border-soft px-[22px] py-3 text-sm transition-colors hover:bg-surface-2 md:grid-cols-[20px_190px_1fr_100px_32px] ${
        highlighted ? 'bg-accent-soft' : ''
      }`}
    >
      <span className="flex items-stretch justify-center">
        <span
          aria-label={message.read ? undefined : 'Unread'}
          className={`w-[3px] self-stretch rounded-full transition-colors ${message.read ? 'bg-transparent' : 'bg-accent'}`}
        />
      </span>

      <span className="hidden min-w-0 flex-col gap-0.5 self-center md:flex">
        <span
          className={`truncate text-[12.8px] ${
            message.read ? 'text-text-secondary' : 'font-semibold text-text-primary'
          }`}
        >
          {message.from_name || message.from}
        </span>
        <span className="truncate text-[11px] text-text-tertiary">To: {shown[0]}</span>
      </span>

      <span className="flex min-w-0 flex-col gap-0.5 self-center">
        <span className="flex min-w-0 items-baseline gap-2">
          <span
            className={`truncate text-[13.5px] ${message.read ? 'text-text-primary' : 'font-semibold text-text-primary'}`}
          >
            {message.subject}
          </span>
          {message.parse_warning && <Badge variant="warning">Parse warning</Badge>}
          {more > 0 && <Badge>+{more} more</Badge>}
        </span>
        {message.tags.length > 0 && (
          <span className="msg-tag flex flex-wrap items-center gap-1.5">
            {message.tags.map((tag) => (
              <TagBadge key={tag} tag={tag} />
            ))}
          </span>
        )}
      </span>

      {/* Time above size, both right-aligned in one column — mirrors
          proposed-mockup.html's .row-meta rather than two separate
          side-by-side columns, so the pair reads as a single unit. */}
      <span className="hidden flex-col items-end gap-0.5 self-center font-mono md:flex">
        <span className="text-[11.5px] font-medium text-accent" title={formatExactTime(message.received_at)}>
          {formatRelativeTime(message.received_at)}
        </span>
        <span className="flex items-center gap-1 text-[10.5px] text-text-tertiary">
          {formatBytes(message.size_bytes)}
          {message.has_attachments && (
            <span className="flex items-center gap-0.5">
              <Paperclip className="h-[11px] w-[11px]" aria-hidden="true" />
              {message.attachment_count}
            </span>
          )}
        </span>
      </span>

      <span className="flex items-center justify-end self-center">
        <MessageRowActions message={message} onPreview={onPreview} />
      </span>
    </div>
  )
}
