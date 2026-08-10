import { useEffect, useState } from 'react'
import { Info } from 'lucide-react'
import { getVersion } from '../../lib/apiClient'
import type { Version } from '../../lib/apiTypes'
import StatsCard from './StatsCard'
import ConnectionInfoCard from './ConnectionInfoCard'
import ConfigTable from './ConfigTable'
import PrecedenceChain from './PrecedenceChain'

function VersionCard() {
  const [version, setVersion] = useState<Version | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getVersion()
      .then((v) => {
        if (!cancelled) setVersion(v)
      })
      .catch(() => {
        if (!cancelled) setError('Failed to load version info.')
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
        <Info className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Version</h2>
      </div>

      {loading && (
        <div className="space-y-2" data-testid="version-loading">
          <div className="h-4 w-1/2 animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-2/3 animate-pulse rounded-sm bg-surface-2" />
          <div className="h-4 w-1/3 animate-pulse rounded-sm bg-surface-2" />
        </div>
      )}

      {!loading && error && <p className="text-sm text-danger">{error}</p>}

      {!loading && !error && version && (
        <dl className="space-y-1 text-sm">
          <div className="flex gap-2">
            <dt className="text-text-tertiary">Version</dt>
            <dd className="font-mono text-text-primary">{version.version}</dd>
          </div>
          <div className="flex gap-2">
            <dt className="text-text-tertiary">Commit</dt>
            <dd className="font-mono text-text-primary">{version.commit}</dd>
          </div>
        </dl>
      )}
    </div>
  )
}

export default function SettingsScreen() {
  return (
    <div>
      <div className="border-b border-border-soft px-6 py-4">
        <h1 className="mb-1 text-[19px] font-semibold tracking-tight text-text-primary">
          Settings
        </h1>
        <p className="max-w-xl text-sm leading-relaxed text-text-secondary">
          The server configuration and runtime info maelsink is currently running with.
        </p>
      </div>

      <div className="p-6">
        <PrecedenceChain />

        <div className="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
          <StatsCard />
          <ConnectionInfoCard />
          <VersionCard />
        </div>

        <div className="mt-4">
          <ConfigTable />
        </div>
      </div>
    </div>
  )
}
