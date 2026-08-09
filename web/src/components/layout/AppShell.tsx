import type { ReactNode } from 'react'
import TopBar from './TopBar'
import Sidebar from './Sidebar'

interface AppShellProps {
  children: ReactNode
}

// Fixed-height, non-scrolling app shell (STYLE_GUIDE.md §1.4): top bar +
// sidebar + an independently-scrolling content region.
export default function AppShell({ children }: AppShellProps) {
  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden">
      <TopBar />
      <div className="flex flex-1 overflow-hidden">
        <Sidebar />
        <main className="flex-1 overflow-y-auto bg-bg">{children}</main>
      </div>
    </div>
  )
}
