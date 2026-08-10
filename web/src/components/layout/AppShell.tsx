import { useEffect, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import TopBar from './TopBar'
import Sidebar from './Sidebar'
import ToastContainer from '../common/ToastContainer'
import ConfirmDialog from '../common/ConfirmDialog'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'

interface AppShellProps {
  children: ReactNode
}

// Fixed-height, non-scrolling app shell (STYLE_GUIDE.md §1.4): top bar +
// sidebar + an independently-scrolling content region. ConfirmDialog is
// mounted once here (not per-screen) since more than one screen/the sidebar
// can trigger a confirmation via useUIStore.openConfirm.
export default function AppShell({ children }: AppShellProps) {
  const modal = useUIStore((state) => state.modal)
  const closeModal = useUIStore((state) => state.closeModal)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    void useMessageStore.getState().fetchMessages()
  }, [])

  // Global Escape-to-back: from a message detail route, Escape returns to
  // the Inbox — mirrors MOCKUP.html's keyboard affordances alongside the
  // existing "/" focus-search shortcut (SearchBar.tsx). Skipped when a modal
  // is open (Escape there should close the modal, not also navigate) and
  // when focus is in a text input/textarea, so it never steals Escape from
  // an in-progress edit.
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (modal) return
      if (!location.pathname.startsWith('/messages/')) return
      const target = e.target as HTMLElement | null
      const tag = target?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return
      navigate('/')
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [modal, location.pathname, navigate])

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden">
      <TopBar />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-y-auto bg-bg">{children}</main>
      </div>
      <ToastContainer />
      <ConfirmDialog
        open={modal?.kind === 'confirm'}
        onClose={closeModal}
        onConfirm={modal?.onConfirm ?? (() => {})}
        title={modal?.title ?? ''}
        body={modal?.body ?? ''}
        confirmLabel={modal?.confirmLabel}
        danger={modal?.danger}
      />
    </div>
  )
}
