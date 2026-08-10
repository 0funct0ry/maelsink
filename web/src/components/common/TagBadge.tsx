import { X } from 'lucide-react'
import { tagColor } from '../../lib/tagColor'

interface TagBadgeProps {
  tag: string
  /** When provided, renders a remove control that calls this on click (M8.2). */
  onRemove?: () => void
}

// A pill badge (not just a dot + plain text) for a message's X-Tag values,
// used in the Inbox row and Message Detail so tags read clearly as discrete
// badges rather than blending into the surrounding text (M6.1 follow-up).
// The sidebar's tag nav keeps its own dot+text treatment since it already
// sits inside a bordered nav item.
export default function TagBadge({ tag, onRemove }: TagBadgeProps) {
  const color = tagColor(tag)
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${color.bg} ${color.text}`}
    >
      <span className={`h-[6px] w-[6px] rounded-full ${color.dot}`} aria-hidden="true" />
      {tag}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={`Remove tag ${tag}`}
          className="ml-0.5 rounded-full hover:opacity-70"
        >
          <X className="h-3 w-3" aria-hidden="true" />
        </button>
      )}
    </span>
  )
}
