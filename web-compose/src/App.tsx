import { useEffect } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import HeaderStrip from './components/HeaderStrip'
import Sidebar from './components/Sidebar'
import { useConnectionStore } from './stores/useConnectionStore'
import MessageListScreen from './screens/MessageListScreen'
import MessageDetailView from './screens/MessageDetailView'
import VarsScreen from './screens/VarsScreen'
import ComposerScreen from './screens/ComposerScreen'
import ApiExplorerScreen from './screens/ApiExplorerScreen'
import JobsPanelScreen from './screens/JobsPanelScreen'

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
            <Route path="/composer" element={<ComposerScreen />} />
            <Route path="/api-explorer" element={<ApiExplorerScreen />} />
            <Route path="/jobs" element={<JobsPanelScreen />} />
          </Routes>
        </main>
      </div>
    </div>
  )
}
