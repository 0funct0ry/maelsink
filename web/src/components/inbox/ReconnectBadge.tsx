import type { WsStatus } from '../../lib/wsClient'

interface ReconnectBadgeProps {
  status: WsStatus
}

/** Subtle "reconnecting..." pill, shown only while backing off a dropped
 * /ws connection (M7.0). Hidden for the normal 'open' state and for the
 * initial 'connecting'/'closed' states so it never flashes on page load. */
export default function ReconnectBadge({ status }: ReconnectBadgeProps) {
  if (status !== 'reconnecting') return null

  return (
    <span className="flex items-center gap-1.5 rounded-full border border-border-soft bg-surface px-2.5 py-1 text-[11.5px] text-text-tertiary">
      <span className="h-[6px] w-[6px] animate-pulse rounded-full bg-warning" aria-hidden="true" />
      Reconnecting…
    </span>
  )
}
