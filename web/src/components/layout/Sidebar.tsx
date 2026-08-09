import { Inbox, Settings } from 'lucide-react'
import { NavLink } from 'react-router-dom'

const NAV_ITEMS = [
  { to: '/', label: 'Inbox', icon: Inbox, end: true },
  { to: '/settings', label: 'Settings', icon: Settings, end: false },
]

export default function Sidebar() {
  return (
    <nav className="flex w-52 shrink-0 flex-col gap-1 border-r border-border bg-surface p-3">
      {NAV_ITEMS.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to}
          to={to}
          end={end}
          className={({ isActive }) =>
            `flex items-center gap-2 rounded-sm px-3 py-2 text-sm font-medium ${
              isActive
                ? 'bg-accent-soft text-accent'
                : 'text-text-secondary hover:bg-surface-2'
            }`
          }
        >
          <Icon size={16} />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
