import { List, Menu, Settings, Tag, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { getInfo } from '../../lib/uiApiClient'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import SearchBar from '../inbox/SearchBar'
import IconButton from '../common/IconButton'
import ReconnectBadge from '../inbox/ReconnectBadge'
import BrandMark from '../icons/BrandMark'
import Modal from '../common/Modal'
import Sidebar from './Sidebar'

// 56px top bar per STYLE_GUIDE.md §1.4 / MOCKUP.html's .topbar: brand mark +
// wordmark, a live SMTP connection pill, a global search box, and
// settings/tags/clear-all shortcuts — all always-visible chrome, not
// per-screen content. Clear-all lives here (not just in the Sidebar) so it's
// reachable from every screen, including Message Detail where the Sidebar's
// own bottom-pinned button isn't visible below the fold on short viewports.
// The tags shortcut (M8.5) similarly guarantees a path to /tags even when
// the Sidebar has ≤5 tags and its own "More…" link isn't shown.
export default function TopBar() {
  const navigate = useNavigate()
  const location = useLocation()
  const openConfirm = useUIStore((state) => state.openConfirm)
  const wsStatus = useUIStore((state) => state.wsStatus)
  const [smtp, setSmtp] = useState<{ host: string; port: number } | null>(null)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Every sidebar nav action navigates (mailbox filter, tag, saved search,
  // "All messages"), so closing the drawer on route change covers all of
  // them without threading a close callback through Sidebar's many
  // onClick handlers.
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

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
      <div className="flex-none md:hidden">
        <IconButton
          icon={<Menu className="h-[19px] w-[19px]" aria-hidden="true" />}
          aria-label="Open navigation"
          onClick={() => setDrawerOpen(true)}
        />
      </div>
      <Modal open={drawerOpen} onClose={() => setDrawerOpen(false)} variant="drawer" maxWidthClass="max-w-[280px]">
        <Sidebar variant="drawer" />
      </Modal>

      <div className="flex h-8 flex-none items-center gap-[9px] border-r border-border-soft pr-4">
        <div className="flex h-[26px] w-[26px] flex-none items-center justify-center rounded-[7px] bg-gradient-to-br from-accent to-[#8b7fff] shadow-[0_2px_6px_rgba(99,91,255,0.35)]">
          <BrandMark className="h-[15px] w-[15px] text-white" />
        </div>
        <span className="font-mono text-[16px] font-semibold tracking-[-0.02em] text-text-primary">
          maelsink
        </span>
      </div>

      {smtp && (
        // Hidden below sm: on a narrow viewport there isn't room for the
        // brand mark, hamburger, this pill, and the search box all in one
        // 56px row without overflowing. The same info is always reachable
        // via Settings > Connection Info and the Inbox empty state.
        <div className="hidden flex-none items-center gap-[7px] rounded-full border border-border-soft bg-surface py-[5px] pl-2 pr-2.5 font-mono text-xs text-text-secondary sm:flex">
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
          icon={<List className="h-[17px] w-[17px]" aria-hidden="true" />}
          aria-label="View SMTP sessions"
          onClick={() => navigate('/sessions')}
        />
        <IconButton
          icon={<Tag className="h-[17px] w-[17px]" aria-hidden="true" />}
          aria-label="Manage tags"
          onClick={() => navigate('/tags')}
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
