import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, ChevronDown, ChevronRight, Paperclip } from 'lucide-react'
import Button from '../components/Button'
import { getMessage, type MessageDetail } from '../lib/composeApi'

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function AddressRow({ label, values }: { label: string; values: string[] }) {
  if (values.length === 0) return null
  return (
    <div className="flex gap-2 text-sm">
      <span className="w-12 shrink-0 text-text-tertiary">{label}</span>
      <span className="min-w-0 truncate text-text-primary">{values.join(', ')}</span>
    </div>
  )
}

export default function MessageDetailView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [message, setMessage] = useState<MessageDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [bodyTab, setBodyTab] = useState<'text' | 'html'>('text')
  const [headersOpen, setHeadersOpen] = useState(false)

  useEffect(() => {
    setMessage(null)
    setError(null)
    if (!id) return
    getMessage(id)
      .then((m) => {
        setMessage(m)
        setBodyTab(m.text_body ? 'text' : 'html')
      })
      .catch((err) => setError(err instanceof Error ? err.message : String(err)))
  }, [id])

  return (
    <div className="scrollbar-thin flex h-full flex-col overflow-y-auto p-4">
      <div>
        <Button variant="ghost" onClick={() => navigate('/messages')}>
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to list
        </Button>
      </div>

      {error && <p className="mt-3 text-sm text-danger">{error}</p>}

      {message && (
        <div className="mx-auto mt-4 w-full max-w-3xl space-y-4">
          <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
            <h1 className="text-lg font-semibold text-text-primary">{message.subject || '(no subject)'}</h1>
            <p className="mt-1 text-xs text-text-tertiary">{new Date(message.received_at).toLocaleString()}</p>

            <div className="mt-4 space-y-1.5 border-t border-border-soft pt-4">
              <AddressRow label="From" values={[message.from]} />
              <AddressRow label="To" values={message.to} />
              <AddressRow label="Cc" values={message.cc} />
              <AddressRow label="Bcc" values={message.bcc} />
            </div>

            <div className="mt-4 flex flex-wrap items-center gap-2 border-t border-border-soft pt-3 text-xs text-text-secondary">
              <span className="rounded-full bg-surface-2 px-2 py-0.5">{formatBytes(message.size_bytes)}</span>
              {message.has_attachments && (
                <span className="flex items-center gap-1 rounded-full bg-surface-2 px-2 py-0.5">
                  <Paperclip className="h-3 w-3" aria-hidden="true" />
                  {message.attachment_count} attachment{message.attachment_count === 1 ? '' : 's'}
                </span>
              )}
              {message.parse_warning && (
                <span className="rounded-full bg-warning-soft px-2 py-0.5 text-warning">Parse warning</span>
              )}
              {message.tags.map((tag) => (
                <span key={tag} className="rounded-full bg-accent-soft px-2 py-0.5 text-accent">
                  {tag}
                </span>
              ))}
            </div>
          </div>

          {message.attachments.length > 0 && (
            <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
              <h2 className="mb-2 text-sm font-semibold text-text-primary">Attachments</h2>
              <ul className="space-y-1.5">
                {message.attachments.map((att) => (
                  <li key={att.id} className="flex items-center gap-2 text-sm text-text-secondary">
                    <Paperclip className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                    <span className="truncate text-text-primary">{att.filename}</span>
                    <span className="shrink-0 text-xs text-text-tertiary">
                      {att.content_type} · {formatBytes(att.size_bytes)}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="rounded-lg border border-border bg-surface shadow-sm">
            <div className="flex items-center gap-1 border-b border-border-soft px-4 pt-3">
              {message.text_body && (
                <button
                  type="button"
                  onClick={() => setBodyTab('text')}
                  className={`rounded-t-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    bodyTab === 'text'
                      ? 'border-b-2 border-accent text-accent'
                      : 'text-text-secondary hover:text-text-primary'
                  }`}
                >
                  Text
                </button>
              )}
              {message.html_body && (
                <button
                  type="button"
                  onClick={() => setBodyTab('html')}
                  className={`rounded-t-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    bodyTab === 'html'
                      ? 'border-b-2 border-accent text-accent'
                      : 'text-text-secondary hover:text-text-primary'
                  }`}
                >
                  HTML
                </button>
              )}
            </div>
            <div className="p-4">
              {bodyTab === 'text' && message.text_body && (
                <pre className="scrollbar-thin max-h-96 overflow-auto whitespace-pre-wrap text-sm text-text-primary">
                  {message.text_body}
                </pre>
              )}
              {bodyTab === 'html' && message.html_body && (
                <iframe
                  title="Message HTML body"
                  srcDoc={message.html_body}
                  sandbox=""
                  className="h-96 w-full rounded-md border border-border-soft bg-white"
                />
              )}
              {!message.text_body && !message.html_body && (
                <p className="text-sm text-text-tertiary">This message has no text or HTML body.</p>
              )}
            </div>
          </div>

          <div className="rounded-lg border border-border bg-surface shadow-sm">
            <button
              type="button"
              onClick={() => setHeadersOpen((v) => !v)}
              className="flex w-full items-center gap-2 px-4 py-3 text-left text-sm font-semibold text-text-primary"
            >
              {headersOpen ? (
                <ChevronDown className="h-4 w-4" aria-hidden="true" />
              ) : (
                <ChevronRight className="h-4 w-4" aria-hidden="true" />
              )}
              Raw headers ({message.headers.length})
            </button>
            {headersOpen && (
              <div className="scrollbar-thin max-h-64 space-y-1 overflow-auto border-t border-border-soft px-4 py-3 font-mono text-xs">
                {message.headers.map((h, i) => (
                  <div key={`${h.name}-${i}`} className="flex gap-2">
                    <span className="shrink-0 text-text-tertiary">{h.name}:</span>
                    <span className="min-w-0 break-all text-text-secondary">{h.value}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
