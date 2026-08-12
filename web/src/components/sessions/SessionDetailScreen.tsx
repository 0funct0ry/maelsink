import { useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Mail } from 'lucide-react'
import { useSessionStore } from '../../stores/useSessionStore'
import { formatAbsoluteDate, formatExactTime, formatRelativeTime } from '../../lib/format'
import Badge from '../common/Badge'
import Button from '../common/Button'

const STATUS_VARIANT: Record<string, 'default' | 'success' | 'warning' | 'danger'> = {
  completed: 'success',
  aborted: 'warning',
  timeout: 'warning',
  rejected: 'danger',
}

export default function SessionDetailScreen() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { selected, selectedStatus, fetchSession, clearSelected } = useSessionStore()

  useEffect(() => {
    if (!id) return
    void fetchSession(id)
    return () => clearSelected()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id])

  if (!selected) {
    if (selectedStatus === 'not_found') {
      return (
        <div className="flex h-full flex-col items-center justify-center gap-4 p-12 text-center">
          <h2 className="text-lg font-semibold text-text-primary">Session not found</h2>
          <p className="text-sm text-text-secondary">
            This session may have been removed by the retention sweeper.
          </p>
          <Button variant="secondary" onClick={() => navigate('/sessions')}>
            Back to Sessions
          </Button>
        </div>
      )
    }

    if (selectedStatus === 'error') {
      return (
        <div className="flex h-full flex-col items-center justify-center gap-4 p-12 text-center">
          <h2 className="text-lg font-semibold text-text-primary">Something went wrong loading this session</h2>
          <p className="text-sm text-text-secondary">Please try again, or go back to Sessions.</p>
          <Button variant="secondary" onClick={() => navigate('/sessions')}>
            Back to Sessions
          </Button>
        </div>
      )
    }

    return (
      <div className="p-6">
        <div className="mb-4 h-6 w-48 animate-pulse rounded bg-surface-2" />
        <div className="mb-2 h-4 w-full max-w-md animate-pulse rounded bg-surface-2" />
        <div className="h-4 w-full max-w-sm animate-pulse rounded bg-surface-2" />
      </div>
    )
  }

  const session = selected

  return (
    <div className="scrollbar-thin flex h-full flex-col overflow-y-auto overflow-x-hidden">
      <div className="flex flex-none items-center gap-3.5 border-b border-border-soft px-6 py-3.5">
        <button
          type="button"
          onClick={() => navigate('/sessions')}
          className="flex flex-none items-center gap-1.5 rounded-sm px-2 py-1.5 text-[13px] font-medium text-text-secondary transition-colors hover:bg-surface hover:text-text-primary"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to sessions
        </button>
        {session.message_id && (
          <button
            type="button"
            onClick={() => navigate(`/messages/${session.message_id}`)}
            className="ml-auto flex items-center gap-1.5 rounded-sm border border-border-soft bg-bg px-[11px] py-[7px] text-[12.8px] font-medium text-text-secondary transition-colors hover:border-border hover:bg-surface hover:text-text-primary"
          >
            <Mail className="h-3.5 w-3.5" aria-hidden="true" />
            View message
          </button>
        )}
      </div>

      <div className="flex-none border-b border-border-soft px-6 py-4">
        <h1 className="mb-3 text-[19px] font-semibold tracking-tight text-text-primary">SMTP Session</h1>
        <div className="flex flex-col gap-1.5">
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-16 flex-none text-text-tertiary">Client</span>
            <span className="break-all font-mono text-[12.6px] text-text-primary">{session.client_ip}</span>
          </div>
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-16 flex-none text-text-tertiary">HELO</span>
            <span className="break-all font-mono text-[12.6px] text-text-primary">
              {session.client_helo || '(none)'}
            </span>
          </div>
          <div className="flex gap-2.5 text-[13px]">
            <span className="w-16 flex-none text-text-tertiary">Started</span>
            <span
              className="break-all font-mono text-[12.6px] text-text-primary"
              title={formatExactTime(session.started_at)}
            >
              {formatRelativeTime(session.started_at)}
            </span>
            <span className="break-all font-mono text-[12.6px] text-text-tertiary">
              ({formatAbsoluteDate(session.started_at)})
            </span>
          </div>
          {session.ended_at && (
            <div className="flex gap-2.5 text-[13px]">
              <span className="w-16 flex-none text-text-tertiary">Ended</span>
              <span
                className="break-all font-mono text-[12.6px] text-text-primary"
                title={formatExactTime(session.ended_at)}
              >
                {formatRelativeTime(session.ended_at)}
              </span>
            </div>
          )}
        </div>
        <div className="mt-2.5 flex flex-wrap items-center gap-2.5">
          {session.status ? (
            <Badge variant={STATUS_VARIANT[session.status] ?? 'default'}>{session.status}</Badge>
          ) : (
            <Badge>In progress</Badge>
          )}
        </div>
      </div>

      <div className="px-6 py-5">
        <h2 className="mb-2.5 text-[13px] font-semibold text-text-primary">Protocol transcript</h2>
        <div className="rounded-md border border-border-soft bg-surface p-3 font-mono text-[12.4px] leading-relaxed">
          {session.transcript.length === 0 ? (
            <p className="text-text-tertiary">No transcript recorded.</p>
          ) : (
            session.transcript.map((line) => (
              <div
                key={line.position}
                className={line.direction === 'C' ? 'text-text-primary' : 'text-accent'}
              >
                <span className="select-none font-semibold">{line.direction}: </span>
                <span className="whitespace-pre-wrap break-all">{line.line}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
