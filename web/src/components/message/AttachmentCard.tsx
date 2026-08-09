import { useEffect, useState } from 'react'
import { Download, FileText, Image as ImageIcon } from 'lucide-react'
import { getAttachmentBlob, getAttachmentDownloadUrl } from '../../lib/apiClient'
import { formatBytes } from '../../lib/format'
import type { AttachmentInfo } from '../../lib/apiTypes'

interface AttachmentCardProps {
  messageId: string
  attachment: AttachmentInfo
}

function isImage(contentType: string): boolean {
  return contentType.startsWith('image/')
}

function isPdf(contentType: string): boolean {
  return contentType === 'application/pdf'
}

export default function AttachmentCard({ messageId, attachment }: AttachmentCardProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null)
  const previewable = isImage(attachment.content_type) || isPdf(attachment.content_type)

  useEffect(() => {
    if (!previewable) return
    let objectUrl: string | null = null
    let cancelled = false

    getAttachmentBlob(messageId, attachment.id)
      .then((blob) => {
        if (cancelled) return
        objectUrl = URL.createObjectURL(blob)
        setPreviewUrl(objectUrl)
      })
      .catch(() => {
        // No preview available — non-fatal, just skip rendering it.
      })

    return () => {
      cancelled = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [messageId, attachment.id, previewable])

  const pdf = isPdf(attachment.content_type)
  const image = isImage(attachment.content_type)
  const Icon = pdf ? FileText : image ? ImageIcon : FileText
  const tileClasses = pdf
    ? 'bg-danger-soft text-danger'
    : image
      ? 'bg-accent-soft text-accent'
      : 'bg-surface-2 text-text-tertiary'

  return (
    <div className="flex w-full items-center gap-2.5 rounded-md border border-border-soft p-2.5 transition-colors hover:border-accent-soft-border hover:shadow-sm sm:w-[220px]">
      <div className={`flex h-[34px] w-[34px] flex-none items-center justify-center overflow-hidden rounded-md ${tileClasses}`}>
        {image && previewUrl ? (
          <img src={previewUrl} alt={attachment.filename} className="h-full w-full object-cover" />
        ) : (
          <Icon className="h-[17px] w-[17px]" aria-hidden="true" />
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[12.5px] font-medium text-text-primary" title={attachment.filename}>
          {attachment.filename}
        </p>
        <p className="mt-0.5 font-mono text-[11px] text-text-tertiary">{formatBytes(attachment.size_bytes)}</p>
      </div>
      <a
        href={getAttachmentDownloadUrl(messageId, attachment.id)}
        download={attachment.filename}
        aria-label={`Download ${attachment.filename}`}
        className="flex h-[26px] w-[26px] flex-none items-center justify-center rounded-md text-text-tertiary transition-colors hover:bg-surface hover:text-accent"
      >
        <Download className="h-3.5 w-3.5" aria-hidden="true" />
      </a>
    </div>
  )
}
