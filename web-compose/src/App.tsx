import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import HeaderStrip from './components/HeaderStrip'
import Sidebar from './components/Sidebar'
import { useConnectionStore } from './stores/useConnectionStore'
import MessageListScreen from './screens/MessageListScreen'
import MessageDetailView from './screens/MessageDetailView'
import VarsScreen from './screens/VarsScreen'
import ComposerPlaceholder from './screens/ComposerPlaceholder'
import ApiExplorerPlaceholder from './screens/ApiExplorerPlaceholder'
import JobsPanelPlaceholder from './screens/JobsPanelPlaceholder'

export default function App() {
  const startPolling = useConnectionStore((s) => s.startPolling)

  useEffect(() => {
    const stop = startPolling()
    return stop
  }, [startPolling])

  return (
    <div className="flex h-full">
      <Sidebar />
      <div className="flex h-full flex-1 flex-col overflow-hidden">
        <HeaderStrip />
        <main className="flex-1 overflow-hidden">
          <Routes>
            <Route path="/" element={<Navigate to="/messages" replace />} />
            <Route path="/messages" element={<MessageListScreen />} />
            <Route path="/messages/:id" element={<MessageDetailView />} />
            <Route path="/vars" element={<VarsScreen />} />
            <Route path="/composer" element={<ComposerPlaceholder />} />
            <Route path="/api-explorer" element={<ApiExplorerPlaceholder />} />
            <Route path="/jobs" element={<JobsPanelPlaceholder />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
