import { Paperclip, Trash2 } from 'lucide-react'
import Badge from '../common/Badge'
import IconButton from '../common/IconButton'
import { formatBytes, formatExactTime, formatRelativeTime, truncateList } from '../../lib/format'
import { useMessageStore } from '../../stores/useMessageStore'
import type { MessageSummary } from '../../lib/apiTypes'

interface MessageRowProps {
  message: MessageSummary
  onOpen: () => void
}

// Grid columns mirror MOCKUP.html's .msg-row (unread dot / from / subject
// area / meta / time), with "to" and size folded into the subject/meta
// clusters (SPEC.md §8.1 requires showing both, but the mockup's row
// doesn't have dedicated columns for them) rather than adding extra columns
// that would widen the row beyond the mockup's density. The delete icon is
// an absolutely-positioned hover affordance (mockup has no per-row delete;
// SPEC.md §8.1 does) so it doesn't consume grid space when not hovered.
export default function MessageRow({ message, onOpen }: MessageRowProps) {
  const deleteMessageOptimistic = useMessageStore((state) => state.deleteMessageOptimistic)
  const { shown, more } = truncateList(message.to, 1)

  function handleDelete() {
    void deleteMessageOptimistic(message.id)
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') onOpen()
      }}
      className="group relative grid cursor-pointer grid-cols-[20px_190px_1fr_90px_70px] items-center gap-4 border-b border-border-soft px-[22px] py-3 text-sm transition-colors hover:bg-surface"
    >
      <span className="flex h-[7px] w-[7px] items-center justify-center">
        {!message.read && <span className="h-[7px] w-[7px] rounded-full bg-accent" aria-label="Unread" />}
      </span>

      <span
        className={`truncate font-mono text-[12.8px] ${
          message.read ? 'text-text-secondary' : 'font-medium text-text-primary'
        }`}
      >
        {message.from}
      </span>

      <span className="flex min-w-0 items-baseline gap-2">
        <span className={`truncate text-[13.5px] ${message.read ? 'text-text-primary' : 'font-medium text-text-primary'}`}>
          {message.subject}
        </span>
        {message.parse_warning && <Badge variant="warning">Parse warning</Badge>}
        <span className="flex min-w-0 flex-1 items-center gap-1 truncate text-[13px] text-text-tertiary">
          <span>— to</span>
          <span className="truncate">{shown[0]}</span>
          {more > 0 && <Badge>+{more} more</Badge>}
        </span>
      </span>

      <span className="flex items-center justify-end gap-1.5 text-[11.5px] text-text-tertiary">
        <span className="font-mono">{formatBytes(message.size_bytes)}</span>
        {message.has_attachments && (
          <span className="flex items-center gap-1">
            <Paperclip className="h-[13px] w-[13px]" aria-hidden="true" />
            {message.attachment_count}
          </span>
        )}
      </span>

      <span
        className="text-right font-mono text-[11.5px] text-text-tertiary"
        title={formatExactTime(message.received_at)}
      >
        {formatRelativeTime(message.received_at)}
      </span>

      <span
        onClick={(e) => e.stopPropagation()}
        className="absolute right-3 top-1/2 -translate-y-1/2 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100"
      >
        <IconButton icon={<Trash2 className="h-4 w-4" />} aria-label="Delete message" onClick={handleDelete} />
      </span>
    </div>
  )
}
