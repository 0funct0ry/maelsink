import { useParams } from 'react-router-dom'

// Route placeholder for M5.0 — the real detail view lands in M6.0.
export default function MessageDetailScreen() {
  const { id } = useParams<{ id: string }>()
  return (
    <div className="p-6 text-text-secondary">
      <h1 className="mb-1 text-lg font-semibold text-text-primary">
        Message <span className="font-mono text-text-secondary">{id}</span>
      </h1>
      <p className="text-sm">Message detail UI lands in M6.0.</p>
    </div>
  )
}
