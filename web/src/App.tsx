import { Route, Routes } from 'react-router-dom'
import AppShell from './components/layout/AppShell'
import InboxScreen from './components/inbox/InboxScreen'
import MessageDetailScreen from './components/message/MessageDetailScreen'
import SettingsScreen from './components/settings/SettingsScreen'

// Routing/providers only — no feature markup here (STYLE_GUIDE.md §2.3).
// Real screens land in M6.0; these are route placeholders proving the SPA
// shell, router, and embedding pipeline work end to end.
export default function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<InboxScreen />} />
        <Route path="/messages/:id" element={<MessageDetailScreen />} />
        <Route path="/settings" element={<SettingsScreen />} />
      </Routes>
    </AppShell>
  )
}
