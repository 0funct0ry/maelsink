import { useEffect, useState, type ReactNode } from 'react'
import { Trash2 } from 'lucide-react'
import { useConnectionStore } from '../stores/useConnectionStore'
import { useJobsStore } from '../stores/useJobsStore'
import Button from '../components/Button'
import ConfirmModal from '../components/ConfirmModal'
import Modal from '../components/Modal'
import {
  ComposeApiError,
  cancelJob,
  startJob,
  type BlastParams,
  type DelugeParams,
  type IntMsgParams,
  type JobKind,
  type JobSnapshot,
  type RandMsgParams,
  type WeirdMsgParams,
} from '../lib/composeJobsApi'

// Jobs Panel (SPEC.md §7.7.4.3, M13.3): one button per command, each
// opening a modal with that command's params (§7.6.2-§7.6.4), a Start
// action that opens the job's WS progress stream, and a Cancel action for
// in-flight jobs. Completed/cancelled jobs stay listed for the life of the
// compose process — held in useJobsStore (in-memory only, no
// localStorage, per SPEC.md §7.7.7).

interface KindMeta {
  kind: JobKind
  title: string
  description: string
}

const KINDS: KindMeta[] = [
  { kind: 'randmsg', title: 'randmsg', description: 'Send a randomly-generated message' },
  { kind: 'intmsg', title: 'intmsg', description: 'Send random messages at randomized intervals' },
  { kind: 'weirdmsg', title: 'weirdmsg', description: 'Send one message of an awkward, edge-case shape' },
  { kind: 'blast', title: 'blast', description: 'Send one message to many generated recipients' },
  { kind: 'deluge', title: 'deluge', description: 'Fire N random messages at maximum throughput' },
]

const inputClass =
  'rounded-md border border-border-soft bg-bg px-2 py-1.5 text-sm text-text-primary focus:border-accent focus:outline-none'

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-text-secondary">
      {label}
      {children}
    </label>
  )
}

function errMessage(err: unknown): string {
  if (err instanceof ComposeApiError) return `${err.code}: ${err.message}`
  return err instanceof Error ? err.message : String(err)
}

function statusColor(status: JobSnapshot['status']): string {
  switch (status) {
    case 'running':
      return 'text-accent'
    case 'completed':
      return 'text-success'
    case 'cancelled':
      return 'text-text-tertiary'
    case 'failed':
      return 'text-danger'
  }
}

// JobsTable renders the recent-jobs list tabularly — kind/status/sent/
// failed/elapsed columns plus a Cancel action for still-running jobs.
function JobsTable({ jobs, onCancel }: { jobs: JobSnapshot[]; onCancel: (jobId: string) => void }) {
  return (
    <div className="scrollbar-thin overflow-x-auto rounded-md border border-border-soft">
      <table className="w-full text-left text-xs">
        <thead>
          <tr className="border-b border-border-soft bg-surface-2 text-text-tertiary">
            <th className="px-3 py-2 font-medium">Kind</th>
            <th className="px-3 py-2 font-medium">Status</th>
            <th className="px-3 py-2 font-medium">Sent</th>
            <th className="px-3 py-2 font-medium">Failed</th>
            <th className="px-3 py-2 font-medium">Elapsed</th>
            <th className="px-3 py-2 font-medium">Started</th>
            <th className="px-3 py-2 font-medium" />
          </tr>
        </thead>
        <tbody>
          {jobs.map((j) => (
            <tr key={j.jobId} className="border-b border-border-soft last:border-b-0 hover:bg-surface-2">
              <td className="px-3 py-2 font-mono text-text-primary">{j.kind}</td>
              <td className={`px-3 py-2 font-semibold ${statusColor(j.status)}`}>
                {j.status}
                {j.error && <span className="ml-2 font-normal text-danger">{j.error}</span>}
              </td>
              <td className="px-3 py-2 text-text-secondary">{j.sent}</td>
              <td className="px-3 py-2 text-text-secondary">{j.failed}</td>
              <td className="px-3 py-2 text-text-secondary">{j.elapsedSeconds.toFixed(1)}s</td>
              <td className="px-3 py-2 text-text-tertiary">{new Date(j.startedAt).toLocaleTimeString()}</td>
              <td className="px-3 py-2 text-right">
                {j.status === 'running' && (
                  <Button variant="danger" onClick={() => onCancel(j.jobId)}>
                    Cancel
                  </Button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// --- per-kind param forms --------------------------------------------------

function ContentFields({
  params,
  set,
}: {
  params: RandMsgParams | IntMsgParams | BlastParams | DelugeParams
  set: (patch: Partial<RandMsgParams>) => void
}) {
  return (
    <div className="grid grid-cols-2 gap-2">
      <Field label="To">
        <input className={inputClass} value={params.to ?? ''} onChange={(e) => set({ to: e.target.value })} />
      </Field>
      <Field label="From">
        <input className={inputClass} value={params.from ?? ''} onChange={(e) => set({ from: e.target.value })} />
      </Field>
      <Field label="Subject">
        <input className={inputClass} value={params.subject ?? ''} onChange={(e) => set({ subject: e.target.value })} />
      </Field>
      <Field label="Body">
        <select
          className={inputClass}
          value={params.body ?? 'random'}
          onChange={(e) => set({ body: e.target.value as RandMsgParams['body'] })}
        >
          <option value="random">random</option>
          <option value="text">text</option>
          <option value="html">html</option>
          <option value="both">both</option>
        </select>
      </Field>
      <Field label="Scenario">
        <input
          className={inputClass}
          placeholder="(random)"
          value={params.scenario ?? ''}
          onChange={(e) => set({ scenario: e.target.value })}
        />
      </Field>
      <Field label="Attachments">
        <input
          type="number"
          className={inputClass}
          value={params.attachments ?? 0}
          onChange={(e) => set({ attachments: Number(e.target.value) })}
        />
      </Field>
    </div>
  )
}

function RandMsgForm({ params, set }: { params: RandMsgParams; set: (p: RandMsgParams) => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ContentFields params={params} set={(patch) => set({ ...params, ...patch })} />
      <div className="grid grid-cols-2 gap-2">
        <Field label="Count">
          <input
            type="number"
            className={inputClass}
            value={params.count ?? 1}
            onChange={(e) => set({ ...params, count: Number(e.target.value) })}
          />
        </Field>
        <Field label="Concurrency">
          <input
            type="number"
            className={inputClass}
            value={params.concurrency ?? 1}
            onChange={(e) => set({ ...params, concurrency: Number(e.target.value) })}
          />
        </Field>
      </div>
    </div>
  )
}

function IntMsgForm({ params, set }: { params: IntMsgParams; set: (p: IntMsgParams) => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ContentFields params={params} set={(patch) => set({ ...params, ...patch })} />
      <div className="grid grid-cols-2 gap-2">
        <Field label="Interval (ms)">
          <input
            type="number"
            className={inputClass}
            value={params.intervalMs ?? 1000}
            onChange={(e) => set({ ...params, intervalMs: Number(e.target.value) })}
          />
        </Field>
        <Field label="Jitter">
          <input
            className={inputClass}
            placeholder="0, 200ms, or 20%"
            value={params.jitter ?? ''}
            onChange={(e) => set({ ...params, jitter: e.target.value })}
          />
        </Field>
        <Field label="Profile">
          <select
            className={inputClass}
            value={params.profile ?? 'steady'}
            onChange={(e) => set({ ...params, profile: e.target.value as IntMsgParams['profile'] })}
          >
            <option value="steady">steady</option>
            <option value="poisson">poisson</option>
            <option value="bursty">bursty</option>
          </select>
        </Field>
        <Field label="Burst size">
          <input
            type="number"
            className={inputClass}
            value={params.burstSize ?? 5}
            onChange={(e) => set({ ...params, burstSize: Number(e.target.value) })}
          />
        </Field>
        <Field label="Count (0 = unbounded)">
          <input
            type="number"
            className={inputClass}
            value={params.count ?? 0}
            onChange={(e) => set({ ...params, count: Number(e.target.value) })}
          />
        </Field>
        <Field label="Duration (ms, 0 = unbounded)">
          <input
            type="number"
            className={inputClass}
            value={params.durationMs ?? 0}
            onChange={(e) => set({ ...params, durationMs: Number(e.target.value) })}
          />
        </Field>
      </div>
    </div>
  )
}

function WeirdMsgForm({ params, set }: { params: WeirdMsgParams; set: (p: WeirdMsgParams) => void }) {
  return (
    <div className="grid grid-cols-2 gap-2">
      <Field label="Kind">
        <select
          className={inputClass}
          value={params.kind ?? 'random'}
          onChange={(e) => set({ ...params, kind: e.target.value as WeirdMsgParams['kind'] })}
        >
          {['random', 'bounce', 'malformed', 'huge', 'unicode', 'spoof', 'thread', 'invite'].map((k) => (
            <option key={k} value={k}>
              {k}
            </option>
          ))}
        </select>
      </Field>
      <Field label="To">
        <input className={inputClass} value={params.to ?? ''} onChange={(e) => set({ ...params, to: e.target.value })} />
      </Field>
      <Field label="From">
        <input
          className={inputClass}
          value={params.from ?? ''}
          onChange={(e) => set({ ...params, from: e.target.value })}
        />
      </Field>
      <Field label="Size (--kind huge)">
        <input
          className={inputClass}
          placeholder="10MB"
          value={params.size ?? ''}
          onChange={(e) => set({ ...params, size: e.target.value })}
        />
      </Field>
      <Field label="Depth (--kind thread)">
        <input
          type="number"
          className={inputClass}
          value={params.depth ?? 5}
          onChange={(e) => set({ ...params, depth: Number(e.target.value) })}
        />
      </Field>
    </div>
  )
}

function BlastForm({ params, set }: { params: BlastParams; set: (p: BlastParams) => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ContentFields params={params} set={(patch) => set({ ...params, ...patch })} />
      <div className="grid grid-cols-2 gap-2">
        <Field label="Recipients">
          <input
            type="number"
            className={inputClass}
            value={params.recipients ?? 10}
            onChange={(e) => set({ ...params, recipients: Number(e.target.value) })}
          />
        </Field>
        <Field label="Split">
          <select
            className={inputClass}
            value={params.split ?? 'to'}
            onChange={(e) => set({ ...params, split: e.target.value as BlastParams['split'] })}
          >
            <option value="to">to</option>
            <option value="cc">cc</option>
            <option value="bcc">bcc</option>
            <option value="mixed">mixed</option>
          </select>
        </Field>
      </div>
    </div>
  )
}

function DelugeForm({ params, set }: { params: DelugeParams; set: (p: DelugeParams) => void }) {
  return (
    <div className="flex flex-col gap-3">
      <ContentFields params={params} set={(patch) => set({ ...params, ...patch })} />
      <div className="grid grid-cols-2 gap-2">
        <Field label="Count">
          <input
            type="number"
            className={inputClass}
            value={params.count ?? 100}
            onChange={(e) => set({ ...params, count: Number(e.target.value) })}
          />
        </Field>
        <Field label="Concurrency">
          <input
            type="number"
            className={inputClass}
            value={params.concurrency ?? 10}
            onChange={(e) => set({ ...params, concurrency: Number(e.target.value) })}
          />
        </Field>
      </div>
    </div>
  )
}

// --- job modal --------------------------------------------------------------

function JobModal({ meta, onClose }: { meta: KindMeta; onClose: () => void }) {
  const upsert = useJobsStore((s) => s.upsert)
  const trackJob = useJobsStore((s) => s.trackJob)
  const [randParams, setRandParams] = useState<RandMsgParams>({})
  const [intParams, setIntParams] = useState<IntMsgParams>({})
  const [weirdParams, setWeirdParams] = useState<WeirdMsgParams>({})
  const [blastParams, setBlastParams] = useState<BlastParams>({})
  const [delugeParams, setDelugeParams] = useState<DelugeParams>({})
  const [starting, setStarting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleStart() {
    setStarting(true)
    setError(null)
    try {
      let result: { jobId: string }
      switch (meta.kind) {
        case 'randmsg':
          result = await startJob('randmsg', randParams)
          break
        case 'intmsg':
          result = await startJob('intmsg', intParams)
          break
        case 'weirdmsg':
          result = await startJob('weirdmsg', weirdParams)
          break
        case 'blast':
          result = await startJob('blast', blastParams)
          break
        case 'deluge':
          result = await startJob('deluge', delugeParams)
          break
      }
      const placeholder: JobSnapshot = {
        jobId: result.jobId,
        kind: meta.kind,
        status: 'running',
        sent: 0,
        failed: 0,
        startedAt: new Date().toISOString(),
        elapsedSeconds: 0,
      }
      // Add it to the Recent Jobs list and start tracking its progress via
      // the store (survives this modal closing) before closing the modal —
      // the job's live status from here on is only shown in that list.
      upsert(placeholder)
      trackJob(result.jobId)
      onClose()
    } catch (err) {
      setError(errMessage(err))
    } finally {
      setStarting(false)
    }
  }

  return (
    <Modal open onClose={onClose} maxWidthClass="max-w-lg">
      <h2 className="text-lg font-semibold text-text-primary">{meta.title}</h2>
      <p className="mt-1 text-xs text-text-tertiary">{meta.description}</p>

      <div className="mt-4">
        {meta.kind === 'randmsg' && <RandMsgForm params={randParams} set={setRandParams} />}
        {meta.kind === 'intmsg' && <IntMsgForm params={intParams} set={setIntParams} />}
        {meta.kind === 'weirdmsg' && <WeirdMsgForm params={weirdParams} set={setWeirdParams} />}
        {meta.kind === 'blast' && <BlastForm params={blastParams} set={setBlastParams} />}
        {meta.kind === 'deluge' && <DelugeForm params={delugeParams} set={setDelugeParams} />}
      </div>

      {error && <p className="mt-3 text-xs text-danger">{error}</p>}

      <div className="mt-4 flex justify-end gap-3">
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button onClick={() => void handleStart()} loading={starting}>
          Start
        </Button>
      </div>
    </Modal>
  )
}

// --- screen --------------------------------------------------------------

export default function JobsPanelScreen() {
  const status = useConnectionStore((s) => s.status)
  const disabled = status === 'red'
  const jobs = useJobsStore((s) => s.jobs)
  const refresh = useJobsStore((s) => s.refresh)
  const upsert = useJobsStore((s) => s.upsert)
  const clearAll = useJobsStore((s) => s.clearAll)
  const [openKind, setOpenKind] = useState<JobKind | null>(null)
  const [confirmClearOpen, setConfirmClearOpen] = useState(false)

  useEffect(() => {
    void refresh()
  }, [refresh])

  async function handleCancelFromList(jobId: string) {
    try {
      const snap = await cancelJob(jobId)
      upsert(snap)
    } catch {
      // surfaced via the modal for jobs started this session; recent-jobs
      // list cancel failures are non-fatal (job likely already finished).
    }
  }

  const openMeta = KINDS.find((k) => k.kind === openKind) ?? null

  return (
    <div className="scrollbar-thin h-full overflow-y-auto p-4">
      <h1 className="mb-3 text-sm font-semibold text-text-primary">Jobs</h1>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        {KINDS.map((meta) => (
          <button
            key={meta.kind}
            type="button"
            disabled={disabled}
            onClick={() => setOpenKind(meta.kind)}
            className="flex flex-col gap-1 rounded-md border border-border-soft bg-surface-2 p-3 text-left transition-colors hover:bg-surface disabled:cursor-not-allowed disabled:opacity-60"
          >
            <span className="font-mono text-sm font-semibold text-text-primary">{meta.title}</span>
            <span className="text-xs text-text-tertiary">{meta.description}</span>
          </button>
        ))}
      </div>

      <div className="mb-2 mt-6 flex items-center justify-between">
        <h2 className="text-xs font-semibold uppercase tracking-wide text-text-tertiary">Recent jobs</h2>
        {jobs.length > 0 && (
          <button
            type="button"
            aria-label="Clear all recent jobs"
            title="Clear all recent jobs"
            onClick={() => setConfirmClearOpen(true)}
            className="rounded-md p-1.5 text-text-tertiary transition-colors hover:bg-surface-2 hover:text-danger"
          >
            <Trash2 className="h-4 w-4" aria-hidden="true" />
          </button>
        )}
      </div>
      {jobs.length === 0 ? (
        <p className="text-sm text-text-tertiary">No jobs started yet this session.</p>
      ) : (
        <JobsTable jobs={jobs} onCancel={handleCancelFromList} />
      )}

      {openMeta && <JobModal meta={openMeta} onClose={() => setOpenKind(null)} />}

      <ConfirmModal
        open={confirmClearOpen}
        onClose={() => setConfirmClearOpen(false)}
        onConfirm={clearAll}
        title="Clear all recent jobs?"
        body="This clears the Jobs Panel's history for this compose session. It does not stop any still-running job, or delete any messages already sent."
        confirmLabel="Clear all"
        danger
      />
    </div>
  )
}
