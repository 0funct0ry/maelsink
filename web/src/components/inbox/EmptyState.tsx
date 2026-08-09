import { Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import { getInfo } from '../../lib/uiApiClient'

export default function EmptyState() {
  const [smtp, setSmtp] = useState<{ host: string; port: number } | null>(null)

  useEffect(() => {
    let cancelled = false
    getInfo()
      .then((info) => {
        if (!cancelled) setSmtp(info.smtp)
      })
      .catch(() => {
        // Non-fatal: the empty state just falls back to generic copy.
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
      <Inbox className="h-10 w-10 text-text-tertiary" aria-hidden="true" />
      <h2 className="text-base font-semibold text-text-primary">No messages yet</h2>
      {smtp ? (
        <p className="text-sm text-text-secondary">
          Point your app&apos;s SMTP at{' '}
          <span className="font-mono text-text-primary">
            {smtp.host}:{smtp.port}
          </span>{' '}
          and sent mail will show up here.
        </p>
      ) : (
        <p className="text-sm text-text-secondary">Send mail through the SMTP server and it will show up here.</p>
      )}
    </div>
  )
}
