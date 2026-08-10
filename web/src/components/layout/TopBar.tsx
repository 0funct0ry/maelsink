import { Mail, Settings, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getInfo } from '../../lib/uiApiClient'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import SearchBar from '../inbox/SearchBar'
import IconButton from '../common/IconButton'
import ReconnectBadge from '../inbox/ReconnectBadge'

// 56px top bar per STYLE_GUIDE.md §1.4 / MOCKUP.html's .topbar: brand mark +
// wordmark, a live SMTP connection pill, a global search box, and
// settings/clear-all shortcuts — all always-visible chrome, not per-screen
// content. Clear-all lives here (not just in the Sidebar) so it's reachable
// from every screen, including Message Detail where the Sidebar's own
// bottom-pinned button isn't visible below the fold on short viewports.
export default function TopBar() {
  const navigate = useNavigate()
  const openConfirm = useUIStore((state) => state.openConfirm)
  const wsStatus = useUIStore((state) => state.wsStatus)
  const [smtp, setSmtp] = useState<{ host: string; port: number } | null>(null)

  useEffect(() => {
    let cancelled = false
    getInfo()
      .then((info) => {
        if (!cancelled) setSmtp(info.smtp)
      })
      .catch(() => {
        // Non-fatal: the pill just doesn't render until this succeeds.
      })
    return () => {
      cancelled = true
    }
  }, [])

  function handleClearAll() {
    openConfirm({
      title: 'Clear all messages?',
      body: 'This will permanently delete every message in the inbox. This action cannot be undone.',
      confirmLabel: 'Clear all',
      danger: true,
      onConfirm: () => {
        void useMessageStore.getState().clearAll()
      },
    })
  }

  return (
    <header className="flex h-14 shrink-0 items-center gap-5 border-b border-border bg-bg px-5">
      <div className="flex h-8 flex-none items-center gap-[9px] border-r border-border-soft pr-4">
        <div className="flex h-[26px] w-[26px] flex-none items-center justify-center rounded-[7px] bg-gradient-to-br from-accent to-[#8b7fff] shadow-[0_2px_6px_rgba(99,91,255,0.35)]">
          <Mail className="h-[15px] w-[15px] text-white" strokeWidth={1.8} aria-hidden="true" />
        </div>
        <span className="font-mono text-[14.5px] font-semibold tracking-[-0.02em] text-text-primary">
          maelsink
        </span>
      </div>

      {smtp && (
        <div className="flex flex-none items-center gap-[7px] rounded-full border border-border-soft bg-surface py-[5px] pl-2 pr-2.5 font-mono text-xs text-text-secondary">
          <span className="relative flex h-[7px] w-[7px] flex-none">
            <span className="absolute inset-[-4px] animate-ping rounded-full border border-success opacity-60" />
            <span className="h-[7px] w-[7px] rounded-full bg-success" />
          </span>
          smtp://{smtp.host}:{smtp.port}
        </div>
      )}

      <div className="flex flex-1 justify-start">
        <SearchBar />
      </div>

      <div className="ml-auto flex flex-none items-center gap-2">
        <ReconnectBadge status={wsStatus} />
        <IconButton
          icon={<Trash2 className="h-[17px] w-[17px]" aria-hidden="true" />}
          aria-label="Clear all messages"
          variant="danger"
          onClick={handleClearAll}
        />
        <IconButton
          icon={<Settings className="h-[17px] w-[17px]" aria-hidden="true" />}
          aria-label="Settings"
          onClick={() => navigate('/settings')}
        />
      </div>
    </header>
  )
}
