import { useEffect, useState } from 'react'
import { Server } from 'lucide-react'
import { getInfo } from '../../lib/uiApiClient'
import type { UiInfo } from '../../lib/apiTypes'
import Badge from '../common/Badge'

export default function ConnectionInfoCard() {
  const [info, setInfo] = useState<UiInfo | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getInfo()
      .then((i) => {
        if (!cancelled) setInfo(i)
      })
      .catch(() => {
        if (!cancelled) setError('Failed to load connection info.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2">
        <Server className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Connection Info</h2>
      </div>

      {loading && (
        <div className="space-y-2" data-testid="connection-loading">
          <div className="h-4 w-2/3 animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-1/3 animate-pulse rounded-sm bg-surface-2" />
        </div>
      )}

      {!loading && error && <p className="text-sm text-danger">{error}</p>}

      {!loading && !error && info && (
        <div className="space-y-3 text-sm">
          <div>
            <p className="font-mono text-base text-text-primary">
              {info.smtp.host}:{info.smtp.port}
            </p>
            <p className="mt-1 text-sm text-text-tertiary">
              Point your application&apos;s SMTP client at this address.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs uppercase tracking-wide text-text-tertiary">API auth:</span>
            <Badge variant={info.auth_enabled ? 'warning' : 'default'}>
              {info.auth_enabled ? 'Enabled' : 'Disabled'}
            </Badge>
          </div>
        </div>
      )}
    </div>
  )
}
