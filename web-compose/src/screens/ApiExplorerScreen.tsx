import { useState, type FormEvent, type ReactNode } from 'react'
import { useConnectionStore } from '../stores/useConnectionStore'
import Button from '../components/Button'
import ConfirmModal from '../components/ConfirmModal'
import Modal from '../components/Modal'
import {
  ComposeApiError,
  clearMessages,
  deleteMessage,
  exportMessages,
  getMessage,
  getStats,
  getVersion,
  health,
  downloadAttachment,
  listMessages,
  triggerDownload,
  type AttachmentInfo,
  type ExportParams,
  type ListMessagesParams,
  type MessageSummary,
} from '../lib/composeApi'

// API Explorer (SPEC.md §7.7.4.2): one curated card per REST endpoint, each
// showing the raw outgoing request and raw response — not a generic
// method/path/body builder (SPEC.md §7.7.7 rules that out).
//
// Laid out as a Postman-style two-pane view: a left nav listing every
// endpoint (grouped, with a method badge) and a right pane showing only the
// selected endpoint's form + raw request/response — rather than a grid of
// same-sized cards, which wastes space on endpoints with no params (get,
// stats, ...) and crowds endpoints with many filter fields (list, export).

type EndpointKey =
  | 'list'
  | 'get'
  | 'delete'
  | 'clear'
  | 'export'
  | 'attachment'
  | 'stats'
  | 'health'
  | 'version'

interface EndpointMeta {
  key: EndpointKey
  title: string
  method: string
  path: string
  group: string
}

const ENDPOINTS: EndpointMeta[] = [
  { key: 'list', title: 'list', method: 'GET', path: '/api/v1/messages', group: 'Messages' },
  { key: 'get', title: 'get', method: 'GET', path: '/api/v1/messages/:id', group: 'Messages' },
  { key: 'delete', title: 'delete', method: 'DELETE', path: '/api/v1/messages/:id', group: 'Messages' },
  { key: 'clear', title: 'clear', method: 'DELETE', path: '/api/v1/messages?confirm=true', group: 'Messages' },
  { key: 'export', title: 'export', method: 'GET', path: '/api/v1/messages/export', group: 'Messages' },
  {
    key: 'attachment',
    title: 'attachment',
    method: 'GET',
    path: '/api/v1/messages/:id/attachments/:attachmentId',
    group: 'Attachments',
  },
  { key: 'stats', title: 'stats', method: 'GET', path: '/api/v1/stats', group: 'System' },
  { key: 'health', title: 'health', method: 'GET', path: '/api/v1/health', group: 'System' },
  { key: 'version', title: 'version', method: 'GET', path: '/api/v1/version', group: 'System' },
]

const GROUPS = ['Messages', 'Attachments', 'System']

interface RequestInfo {
  method: string
  path: string
  body?: unknown
}

function MethodBadge({ method }: { method: string }) {
  const cls = method === 'DELETE' ? 'text-danger' : 'text-success'
  return <span className={`w-12 shrink-0 text-[10px] font-bold ${cls}`}>{method}</span>
}

function RawPanel({
  request,
  response,
  error,
}: {
  request: RequestInfo | null
  response: unknown
  error: string | null
}) {
  if (!request) return null
  return (
    <div className="mt-3 space-y-2 rounded-md bg-surface-2 p-3 text-xs">
      <div>
        <p className="font-semibold text-text-secondary">Request</p>
        <pre className="scrollbar-thin overflow-x-auto whitespace-pre-wrap break-all text-text-primary">
          {request.method} {request.path}
          {request.body !== undefined ? '\n' + JSON.stringify(request.body, null, 2) : ''}
        </pre>
      </div>
      <div>
        <p className="font-semibold text-text-secondary">Response</p>
        {error ? (
          <pre className="whitespace-pre-wrap text-danger">{error}</pre>
        ) : (
          <pre className="scrollbar-thin overflow-x-auto whitespace-pre-wrap break-all text-text-primary">
            {response === undefined ? '' : JSON.stringify(response, null, 2)}
          </pre>
        )}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="flex flex-col gap-1 text-xs text-text-secondary">
      {label}
      {children}
    </label>
  )
}

const inputClass =
  'rounded-md border border-border-soft bg-bg px-2 py-1.5 text-sm text-text-primary focus:border-accent focus:outline-none'

function errMessage(err: unknown): string {
  if (err instanceof ComposeApiError) return `${err.code}: ${err.message}`
  return err instanceof Error ? err.message : String(err)
}

// --- Fire-and-display panels (no params) ---------------------------------

function FireAndDisplayPanel({
  method,
  path,
  disabled,
  run,
}: {
  method: string
  path: string
  disabled: boolean
  run: () => Promise<unknown>
}) {
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleRun() {
    setLoading(true)
    setRequest({ method, path })
    setError(null)
    try {
      const result = await run()
      setResponse(result)
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Button variant="secondary" onClick={() => void handleRun()} loading={loading} disabled={disabled}>
        Run
      </Button>
      <RawPanel request={request} response={response} error={error} />
    </div>
  )
}

// --- list ------------------------------------------------------------------

function ListPanel({
  disabled,
  onResults,
}: {
  disabled: boolean
  onResults: (messages: MessageSummary[]) => void
}) {
  const [filters, setFilters] = useState<ListMessagesParams>({})
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    const params: ListMessagesParams = { ...filters, limit: 20, offset: 0 }
    setRequest({
      method: 'GET',
      path: `/api/v1/messages?${new URLSearchParams(
        Object.entries(params).filter(([, v]) => v != null && v !== '').map(([k, v]) => [k, String(v)]),
      ).toString()}`,
    })
    setError(null)
    try {
      const result = await listMessages(params)
      setResponse(result)
      onResults(result.messages)
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  function set(field: keyof ListMessagesParams) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setFilters((f) => ({ ...f, [field]: e.target.value }))
  }

  return (
    <div>
      <form onSubmit={(e) => void handleSubmit(e)} className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Field label="From">
          <input className={inputClass} value={filters.from ?? ''} onChange={set('from')} />
        </Field>
        <Field label="To">
          <input className={inputClass} value={filters.to ?? ''} onChange={set('to')} />
        </Field>
        <Field label="Subject">
          <input className={inputClass} value={filters.subject ?? ''} onChange={set('subject')} />
        </Field>
        <Field label="Search (q)">
          <input className={inputClass} value={filters.q ?? ''} onChange={set('q')} />
        </Field>
        <Field label="Since">
          <input className={inputClass} placeholder="RFC3339" value={filters.since ?? ''} onChange={set('since')} />
        </Field>
        <Field label="Until">
          <input className={inputClass} placeholder="RFC3339" value={filters.until ?? ''} onChange={set('until')} />
        </Field>
        <div className="col-span-2 sm:col-span-3">
          <Button type="submit" variant="secondary" loading={loading} disabled={disabled}>
            Run
          </Button>
        </div>
      </form>
      <RawPanel request={request} response={response} error={error} />
    </div>
  )
}

// --- get / delete (share a "pick from current list" affordance) ----------

function MessagePicker({
  messages,
  value,
  onChange,
}: {
  messages: MessageSummary[]
  value: string
  onChange: (id: string) => void
}) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row">
      <input
        className={`${inputClass} flex-1`}
        placeholder="message ID"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      {messages.length > 0 && (
        <select
          className={inputClass}
          value=""
          onChange={(e) => {
            if (e.target.value) onChange(e.target.value)
          }}
        >
          <option value="">Pick from current list…</option>
          {messages.map((m) => (
            <option key={m.id} value={m.id}>
              {m.subject || '(no subject)'} — {m.id}
            </option>
          ))}
        </select>
      )}
    </div>
  )
}

function GetPanel({ disabled, messages }: { disabled: boolean; messages: MessageSummary[] }) {
  const [id, setId] = useState('')
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleRun() {
    if (!id) return
    setLoading(true)
    setRequest({ method: 'GET', path: `/api/v1/messages/${encodeURIComponent(id)}` })
    setError(null)
    try {
      setResponse(await getMessage(id))
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <MessagePicker messages={messages} value={id} onChange={setId} />
      <div className="mt-2">
        <Button variant="secondary" onClick={() => void handleRun()} loading={loading} disabled={disabled || !id}>
          Run
        </Button>
      </div>
      <RawPanel request={request} response={response} error={error} />
    </div>
  )
}

function DeletePanel({ disabled, messages }: { disabled: boolean; messages: MessageSummary[] }) {
  const [id, setId] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleConfirmed() {
    setLoading(true)
    setRequest({ method: 'DELETE', path: `/api/v1/messages/${encodeURIComponent(id)}` })
    setError(null)
    try {
      await deleteMessage(id)
      setResponse({ status: 204, message: 'deleted' })
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <MessagePicker messages={messages} value={id} onChange={setId} />
      <div className="mt-2">
        <Button variant="danger" onClick={() => setConfirmOpen(true)} loading={loading} disabled={disabled || !id}>
          Run
        </Button>
      </div>
      <RawPanel request={request} response={response} error={error} />
      <ConfirmModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => void handleConfirmed()}
        title="Delete this message?"
        body="This permanently deletes the message from the target. This cannot be undone."
        confirmLabel="Delete"
        danger
      />
    </div>
  )
}

// --- clear -----------------------------------------------------------------

function ClearPanel({ disabled }: { disabled: boolean }) {
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleConfirmed() {
    setLoading(true)
    setRequest({ method: 'DELETE', path: '/api/v1/messages?confirm=true' })
    setError(null)
    try {
      await clearMessages()
      setResponse({ status: 204, message: 'cleared' })
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Button variant="danger" onClick={() => setConfirmOpen(true)} loading={loading} disabled={disabled}>
        Run
      </Button>
      <RawPanel request={request} response={response} error={error} />
      <ConfirmModal
        open={confirmOpen}
        onClose={() => setConfirmOpen(false)}
        onConfirm={() => void handleConfirmed()}
        title="Clear all messages?"
        body="This permanently deletes every message on the target. This cannot be undone."
        confirmLabel="Clear all"
        danger
      />
    </div>
  )
}

// --- export ------------------------------------------------------------

function ExportPanel({ disabled }: { disabled: boolean }) {
  const [modalOpen, setModalOpen] = useState(false)
  const [filters, setFilters] = useState<ExportParams>({})
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  function set(field: keyof ExportParams) {
    return (e: React.ChangeEvent<HTMLInputElement>) => setFilters((f) => ({ ...f, [field]: e.target.value }))
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setLoading(true)
    const q = new URLSearchParams(
      Object.entries(filters).filter(([, v]) => v != null && v !== '') as [string, string][],
    )
    setRequest({ method: 'GET', path: `/api/v1/messages/export?${q.toString()}` })
    setError(null)
    try {
      const result = await exportMessages(filters)
      triggerDownload(result)
      setResponse({ downloaded: result.filename, size_bytes: result.blob.size })
      setModalOpen(false)
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Button variant="secondary" onClick={() => setModalOpen(true)} disabled={disabled}>
        Export…
      </Button>
      <RawPanel request={request} response={response} error={error} />
      <Modal open={modalOpen} onClose={() => setModalOpen(false)}>
        <h2 className="text-lg font-semibold text-text-primary">Export messages</h2>
        <form onSubmit={(e) => void handleSubmit(e)} className="mt-4 grid grid-cols-2 gap-2">
          <Field label="From">
            <input className={inputClass} value={filters.from ?? ''} onChange={set('from')} />
          </Field>
          <Field label="To">
            <input className={inputClass} value={filters.to ?? ''} onChange={set('to')} />
          </Field>
          <Field label="Subject">
            <input className={inputClass} value={filters.subject ?? ''} onChange={set('subject')} />
          </Field>
          <Field label="Search (q)">
            <input className={inputClass} value={filters.q ?? ''} onChange={set('q')} />
          </Field>
          <Field label="Since">
            <input className={inputClass} placeholder="RFC3339" value={filters.since ?? ''} onChange={set('since')} />
          </Field>
          <Field label="Until">
            <input className={inputClass} placeholder="RFC3339" value={filters.until ?? ''} onChange={set('until')} />
          </Field>
          <div className="col-span-2 mt-2 flex justify-end gap-3">
            <Button type="button" variant="secondary" onClick={() => setModalOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" loading={loading}>
              Download .zip
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  )
}

// --- attachment --------------------------------------------------------

function AttachmentPanel({ disabled, messages }: { disabled: boolean; messages: MessageSummary[] }) {
  const [id, setId] = useState('')
  const [attachments, setAttachments] = useState<AttachmentInfo[]>([])
  const [loading, setLoading] = useState(false)
  const [request, setRequest] = useState<RequestInfo | null>(null)
  const [response, setResponse] = useState<unknown>(undefined)
  const [error, setError] = useState<string | null>(null)

  async function handleList() {
    if (!id) return
    setLoading(true)
    setRequest({ method: 'GET', path: `/api/v1/messages/${encodeURIComponent(id)}` })
    setError(null)
    try {
      const detail = await getMessage(id)
      setAttachments(detail.attachments)
      setResponse({ attachments: detail.attachments })
    } catch (err) {
      setAttachments([])
      setResponse(undefined)
      setError(errMessage(err))
    } finally {
      setLoading(false)
    }
  }

  async function handleDownload(att: AttachmentInfo) {
    setRequest({
      method: 'GET',
      path: `/api/v1/messages/${encodeURIComponent(id)}/attachments/${encodeURIComponent(att.id)}`,
    })
    setError(null)
    try {
      const result = await downloadAttachment(id, att.id, att.filename)
      triggerDownload(result)
      setResponse({ downloaded: result.filename, size_bytes: result.blob.size })
    } catch (err) {
      setResponse(undefined)
      setError(errMessage(err))
    }
  }

  return (
    <div>
      <MessagePicker messages={messages} value={id} onChange={setId} />
      <div className="mt-2">
        <Button variant="secondary" onClick={() => void handleList()} loading={loading} disabled={disabled || !id}>
          List attachments
        </Button>
      </div>
      {attachments.length > 0 && (
        <ul className="mt-2 space-y-1">
          {attachments.map((a) => (
            <li key={a.id}>
              <button
                type="button"
                className="text-sm text-accent hover:underline"
                onClick={() => void handleDownload(a)}
              >
                {a.filename} ({a.size_bytes} bytes)
              </button>
            </li>
          ))}
        </ul>
      )}
      <RawPanel request={request} response={response} error={error} />
    </div>
  )
}

// --- screen --------------------------------------------------------------

export default function ApiExplorerScreen() {
  const status = useConnectionStore((s) => s.status)
  const disabled = status === 'red'
  const [messages, setMessages] = useState<MessageSummary[]>([])
  const [selected, setSelected] = useState<EndpointKey>('list')

  const selectedMeta = ENDPOINTS.find((e) => e.key === selected) ?? ENDPOINTS[0]

  function renderPanel() {
    switch (selected) {
      case 'list':
        return <ListPanel disabled={disabled} onResults={setMessages} />
      case 'get':
        return <GetPanel disabled={disabled} messages={messages} />
      case 'delete':
        return <DeletePanel disabled={disabled} messages={messages} />
      case 'clear':
        return <ClearPanel disabled={disabled} />
      case 'export':
        return <ExportPanel disabled={disabled} />
      case 'attachment':
        return <AttachmentPanel disabled={disabled} messages={messages} />
      case 'stats':
        return <FireAndDisplayPanel method="GET" path="/api/v1/stats" disabled={disabled} run={getStats} />
      case 'health':
        return <FireAndDisplayPanel method="GET" path="/api/v1/health" disabled={disabled} run={health} />
      case 'version':
        return <FireAndDisplayPanel method="GET" path="/api/v1/version" disabled={disabled} run={getVersion} />
    }
  }

  return (
    <div className="flex h-full">
      <nav aria-label="API endpoints" className="scrollbar-thin w-56 shrink-0 overflow-y-auto border-r border-border-soft p-3">
        <h1 className="mb-3 px-1 text-sm font-semibold text-text-primary">API Explorer</h1>
        {GROUPS.map((group) => (
          <div key={group} className="mb-4">
            <p className="mb-1 px-1 text-[11px] font-semibold uppercase tracking-wide text-text-tertiary">
              {group}
            </p>
            <div className="flex flex-col gap-0.5">
              {ENDPOINTS.filter((e) => e.group === group).map((e) => (
                <button
                  key={e.key}
                  type="button"
                  aria-label={e.title}
                  onClick={() => setSelected(e.key)}
                  className={`flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm transition-colors ${
                    selected === e.key ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface-2'
                  }`}
                >
                  <MethodBadge method={e.method} />
                  <span className="truncate">{e.title}</span>
                </button>
              ))}
            </div>
          </div>
        ))}
      </nav>

      <main className="scrollbar-thin flex-1 overflow-y-auto p-4">
        <div className="mb-4">
          <h2 className="text-base font-semibold text-text-primary">{selectedMeta.title}</h2>
          <p className="mt-0.5 flex items-center gap-1.5 text-xs text-text-tertiary">
            <MethodBadge method={selectedMeta.method} />
            <span className="font-mono">{selectedMeta.path}</span>
          </p>
        </div>
        {renderPanel()}
      </main>
    </div>
  )
}
