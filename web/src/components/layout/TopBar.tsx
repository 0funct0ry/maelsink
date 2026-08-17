import { List, Menu, Settings, Tag, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import SearchBar from '../inbox/SearchBar'
import IconButton from '../common/IconButton'
import ReconnectBadge from '../inbox/ReconnectBadge'
import BrandMark from '../icons/BrandMark'
import Modal from '../common/Modal'
import Sidebar from './Sidebar'

// 56px top bar per STYLE_GUIDE.md §1.4 / MOCKUP.html's .topbar: brand mark +
// wordmark, a global search box, and settings/tags/clear-all shortcuts — all
// always-visible chrome, not per-screen content. Clear-all lives here (not
// just in the Sidebar) so it's reachable from every screen, including
// Message Detail where the Sidebar's own bottom-pinned button isn't visible
// below the fold on short viewports. The tags shortcut (M8.5) similarly
// guarantees a path to /tags even when the Sidebar has ≤5 tags and its own
// "More…" link isn't shown.
export default function TopBar() {
  const navigate = useNavigate()
  const location = useLocation()
  const openConfirm = useUIStore((state) => state.openConfirm)
  const wsStatus = useUIStore((state) => state.wsStatus)
  const [drawerOpen, setDrawerOpen] = useState(false)

  // Every sidebar nav action navigates (mailbox filter, tag, saved search,
  // "All messages"), so closing the drawer on route change covers all of
  // them without threading a close callback through Sidebar's many
  // onClick handlers.
  useEffect(() => {
    setDrawerOpen(false)
  }, [location.pathname])

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

      {/* w-[196px]: header's px-5 (20px) left padding + this width lands the
          border-r exactly on the sidebar's right edge (w-[216px] in
          Sidebar.tsx), instead of sizing to the brand mark + wordmark's
          content width. */}
      <div className="flex h-8 w-[196px] flex-none items-center gap-[9px] border-r border-border-soft">
        <div className="flex h-[26px] w-[26px] flex-none items-center justify-center rounded-[7px] bg-gradient-to-br from-accent to-[#8b7fff] shadow-[0_2px_6px_rgba(99,91,255,0.35)]">
          <BrandMark className="h-[15px] w-[15px] text-white" />
        </div>
        <span className="font-mono text-[16px] font-semibold tracking-[-0.02em] text-text-primary">
          maelsink
        </span>
      </div>

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
