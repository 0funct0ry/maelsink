import AttachmentCard from './AttachmentCard'
import type { AttachmentInfo } from '../../lib/apiTypes'

interface AttachmentGridProps {
  messageId: string
  attachments: AttachmentInfo[]
}

export default function AttachmentGrid({ messageId, attachments }: AttachmentGridProps) {
  if (attachments.length === 0) return null

  return (
    <div>
      <h2 className="mb-2.5 text-xs font-semibold uppercase tracking-wide text-text-tertiary">
        Attachments ({attachments.length})
      </h2>
      <div className="flex flex-wrap gap-2.5">
        {attachments.map((attachment) => (
          <AttachmentCard key={attachment.id} messageId={messageId} attachment={attachment} />
        ))}
      </div>
    </div>
  )
}
