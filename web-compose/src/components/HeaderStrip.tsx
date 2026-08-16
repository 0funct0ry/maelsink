import { useConnectionStore, type ConnectionStatus } from '../stores/useConnectionStore'
import ThemeToggle from './ThemeToggle'

const statusDotClasses: Record<ConnectionStatus, string> = {
  green: 'bg-success',
  yellow: 'bg-warning',
  red: 'bg-danger',
}

const statusLabel: Record<ConnectionStatus, string> = {
  green: 'Connected to target',
  yellow: 'Target reachable but reporting errors',
  red: 'Target unreachable — check --api-addr/--smtp-addr',
}

// Slim top bar: connection-status dot (driven by useConnectionStore) and
// the theme switcher. Navigation lives in Sidebar instead.
export default function HeaderStrip() {
  const status = useConnectionStore((s) => s.status)

  return (
    <header className="flex items-center justify-end gap-4 border-b border-border bg-surface px-4 py-2.5">
      <ThemeToggle />
      <div className="flex items-center gap-2" title={statusLabel[status]}>
        <span className={`h-2.5 w-2.5 rounded-full ${statusDotClasses[status]}`} aria-hidden="true" />
        <span className="text-xs text-text-secondary">{statusLabel[status]}</span>
      </div>
    </header>
  )
}
