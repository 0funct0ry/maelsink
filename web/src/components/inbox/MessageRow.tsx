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
export default function MessageRow({ message, onOpen, onPreview, highlighted }: MessageRowProps) {
  const { shown, more } = truncateList(message.to, 1)

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onOpen()
      }}
      className={`group grid cursor-pointer grid-cols-[20px_1fr_32px] items-start gap-4 border-b border-border-soft px-[22px] py-3 text-sm transition-colors hover:bg-surface md:grid-cols-[20px_190px_1fr_90px_70px_32px] ${
        highlighted ? 'bg-accent-soft' : ''
      }`}
    >
      <span className="flex h-[7px] w-[7px] items-center justify-center pt-1.5">
        {!message.read && <span className="h-[7px] w-[7px] rounded-full bg-accent" aria-label="Unread" />}
      </span>

      <span className="hidden min-w-0 flex-col gap-0.5 md:flex">
        <span
          className={`truncate text-[12.8px] ${
            message.read ? 'text-text-secondary' : 'font-semibold text-text-primary'
          }`}
        >
          {message.from_name || message.from}
        </span>
        <span className="truncate text-[11px] text-text-tertiary">To: {shown[0]}</span>
      </span>

      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="flex min-w-0 items-baseline gap-2">
          <span
            className={`truncate text-[13.5px] ${message.read ? 'text-text-primary' : 'font-medium text-text-primary'}`}
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

      <span className="hidden items-center justify-end gap-1.5 pt-0.5 text-[11.5px] text-text-tertiary md:flex">
        <span className="font-mono">{formatBytes(message.size_bytes)}</span>
        {message.has_attachments && (
          <span className="flex items-center gap-1">
            <Paperclip className="h-[13px] w-[13px]" aria-hidden="true" />
            {message.attachment_count}
          </span>
        )}
      </span>

      <span
        className="hidden pt-0.5 text-right font-mono text-[11.5px] text-text-tertiary md:block"
        title={formatExactTime(message.received_at)}
      >
        {formatRelativeTime(message.received_at)}
      </span>

      <span className="flex items-center justify-end pt-0.5">
        <MessageRowActions message={message} onPreview={onPreview} />
      </span>
    </div>
  )
}
