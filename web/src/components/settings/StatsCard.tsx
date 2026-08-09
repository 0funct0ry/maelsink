import { useEffect, useState } from 'react'
import { Database } from 'lucide-react'
import { getStats } from '../../lib/apiClient'
import { formatBytes, formatExactTime } from '../../lib/format'
import type { Stats } from '../../lib/apiTypes'

export default function StatsCard() {
  const [stats, setStats] = useState<Stats | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getStats()
      .then((s) => {
        if (!cancelled) setStats(s)
      })
      .catch(() => {
        if (!cancelled) setError('Failed to load message stats.')
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
        <Database className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Message Stats</h2>
      </div>

      {loading && (
        <div className="space-y-2" data-testid="stats-loading">
          <div className="h-4 w-3/4 animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-1/2 animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-2/3 animate-pulse rounded-sm bg-surface-2" />
        </div>
      )}

      {!loading && error && <p className="text-sm text-danger">{error}</p>}

      {!loading && !error && stats && (
        <dl className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <dt className="text-xs uppercase tracking-wide text-text-tertiary">
              Total Messages
            </dt>
            <dd className="font-mono text-text-primary">{stats.total_messages}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wide text-text-tertiary">Total Size</dt>
            <dd className="font-mono text-text-primary">{formatBytes(stats.total_size_bytes)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wide text-text-tertiary">
              Oldest Received
            </dt>
            <dd className="font-mono text-text-primary">
              {stats.oldest_received_at ? formatExactTime(stats.oldest_received_at) : '—'}
            </dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wide text-text-tertiary">
              Newest Received
            </dt>
            <dd className="font-mono text-text-primary">
              {stats.newest_received_at ? formatExactTime(stats.newest_received_at) : '—'}
            </dd>
          </div>
        </dl>
      )}
    </div>
  )
}
