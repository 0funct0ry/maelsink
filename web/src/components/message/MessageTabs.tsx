import { useEffect, useState } from 'react'
import { ShieldAlert } from 'lucide-react'
import { getRaw } from '../../lib/apiClient'
import HeaderTable from './HeaderTable'
import type { MessageDetail } from '../../lib/apiTypes'

const TABS = ['Rendered HTML', 'Plain Text', 'Raw Source', 'Headers'] as const
type Tab = (typeof TABS)[number]

interface MessageTabsProps {
  message: MessageDetail
}

export default function MessageTabs({ message }: MessageTabsProps) {
  const [activeTab, setActiveTab] = useState<Tab>('Rendered HTML')
  const [rawSource, setRawSource] = useState<string | null>(null)
  const [rawLoading, setRawLoading] = useState(false)
  const [rawError, setRawError] = useState(false)

  useEffect(() => {
    if (activeTab !== 'Raw Source' || rawSource !== null || rawLoading) return
    setRawLoading(true)
    setRawError(false)
    getRaw(message.id)
      .then((text) => setRawSource(text))
      .catch(() => setRawError(true))
      .finally(() => setRawLoading(false))
  }, [activeTab, rawSource, rawLoading, message.id])

  return (
    <div>
      <div className="flex border-b border-border-soft" role="tablist">
        {TABS.map((tab) => (
          <button
            key={tab}
            type="button"
            role="tab"
            aria-selected={activeTab === tab}
            onClick={() => setActiveTab(tab)}
            className={`mr-5 border-b-2 px-1 py-[11px] text-[13px] font-medium transition-colors ${
              activeTab === tab
                ? 'border-accent text-accent'
                : 'border-transparent text-text-tertiary hover:text-text-primary'
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      <div className="pt-5">
        {/* The preview pane below is intentionally a fixed white canvas,
            not a theme token: it renders the email's own HTML, which
            (like most email clients) assumes a white page background
            regardless of app theme. */}
        {activeTab === 'Rendered HTML' && (
          <div className="overflow-hidden rounded-md border border-border-soft bg-white">
            <div className="flex items-center gap-1.5 border-b border-border-soft bg-surface px-3 py-[7px] font-mono text-[11px] text-text-tertiary">
              <ShieldAlert className="h-3 w-3" aria-hidden="true" />
              Sandboxed preview — scripts disabled
            </div>
            <iframe
              sandbox="allow-same-origin"
              srcDoc={message.html_body}
              className="h-96 w-full"
              title="Rendered message preview"
            />
          </div>
        )}

        {activeTab === 'Plain Text' && (
          <pre className="whitespace-pre-wrap rounded-md border border-border-soft bg-surface p-5 font-mono text-[12.8px] leading-[1.7] text-text-primary">
            {message.text_body}
          </pre>
        )}

        {activeTab === 'Raw Source' && (
          <div>
            {rawLoading && <p className="text-sm text-text-secondary">Loading raw source…</p>}
            {rawError && <p className="text-sm text-danger">Failed to load raw source.</p>}
            {!rawLoading && !rawError && rawSource !== null && (
              <pre className="whitespace-pre-wrap rounded-md bg-[#1a1f36] p-5 font-mono text-[12.2px] leading-[1.65] text-[#c6cbe0]">
                {rawSource}
              </pre>
            )}
          </div>
        )}

        {activeTab === 'Headers' && <HeaderTable headers={message.headers} />}
      </div>
    </div>
  )
}
