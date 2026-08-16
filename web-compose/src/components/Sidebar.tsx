import { useState } from 'react'
import { NavLink } from 'react-router-dom'
import { ChevronLeft, ChevronRight, Inbox, ListChecks, MailPlus, SlidersHorizontal, Variable } from 'lucide-react'

const NAV_ITEMS = [
  { to: '/messages', label: 'Message List', icon: Inbox },
  { to: '/vars', label: 'Vars', icon: Variable },
  { to: '/composer', label: 'Composer', icon: MailPlus },
  { to: '/api-explorer', label: 'API Explorer', icon: SlidersHorizontal },
  { to: '/jobs', label: 'Jobs', icon: ListChecks },
]

const SIDEBAR_COLLAPSED_KEY = 'maelsink-compose-sidebar-collapsed'

function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === '1'
  } catch {
    return false
  }
}

function writeCollapsed(collapsed: boolean): void {
  try {
    window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, collapsed ? '1' : '0')
  } catch {
    // localStorage unavailable — the choice just won't survive a reload.
  }
}

// Collapsible left sidebar nav, replacing the old top-bar nav links so the
// header strip can stay a slim, always-visible status bar.
export default function Sidebar() {
  const [collapsed, setCollapsed] = useState(readCollapsed)

  function toggle() {
    setCollapsed((prev) => {
      const next = !prev
      writeCollapsed(next)
      return next
    })
  }

  return (
    <aside
      className={`flex h-full flex-col border-r border-border bg-surface transition-[width] duration-150 ${
        collapsed ? 'w-14' : 'w-52'
      }`}
    >
      <div className={`flex items-center px-3 py-3 ${collapsed ? 'justify-center' : 'justify-between'}`}>
        {!collapsed && <span className="truncate text-sm font-semibold text-text-primary">maelsink compose</span>}
        <button
          type="button"
          onClick={toggle}
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          className="rounded-md p-1 text-text-secondary hover:bg-surface-2 hover:text-text-primary"
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" aria-hidden="true" />
          ) : (
            <ChevronLeft className="h-4 w-4" aria-hidden="true" />
          )}
        </button>
      </div>

      <nav className="flex flex-1 flex-col gap-1 px-2">
        {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
          <NavLink
            key={to}
            to={to}
            title={collapsed ? label : undefined}
            className={({ isActive }) =>
              `flex items-center gap-2.5 rounded-md px-2.5 py-2 text-sm font-medium transition-colors ${
                collapsed ? 'justify-center' : ''
              } ${isActive ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface-2'}`
            }
          >
            <Icon className="h-4 w-4 shrink-0" aria-hidden="true" />
            {!collapsed && <span className="truncate">{label}</span>}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
