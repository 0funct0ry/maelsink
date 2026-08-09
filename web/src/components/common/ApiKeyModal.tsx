import { useState, type FormEvent } from 'react'
import Modal from './Modal'
import Button from './Button'
import { useUIStore } from '../../stores/useUIStore'

// This modal is a mandatory auth gate: it only goes away via a successful
// submit (setAuthToken clears authRequired). We deliberately don't wire
// Escape/backdrop dismissal (Modal's dismissable=false) and don't render a
// close button — closing any other way would just re-show it on the next
// failed request anyway, so there's no point fighting the store for it.
export default function ApiKeyModal() {
  const authRequired = useUIStore((state) => state.authRequired)
  const setAuthToken = useUIStore((state) => state.setAuthToken)
  const [value, setValue] = useState('')
  const [error, setError] = useState<string | null>(null)

  function handleSubmit(event: FormEvent) {
    event.preventDefault()
    const trimmed = value.trim()
    if (!trimmed) {
      setError('API key is required.')
      return
    }
    setError(null)
    setAuthToken(trimmed)
    setValue('')
  }

  return (
    <Modal open={authRequired} onClose={() => {}} dismissable={false}>
      <h2 className="text-lg font-semibold text-text-primary">Authentication required</h2>
      <p className="mt-2 text-sm text-text-secondary">
        This maelsink instance requires an API key to access the REST API.
      </p>
      <form onSubmit={handleSubmit} className="mt-4">
        <label htmlFor="api-key-input" className="block text-sm font-medium text-text-primary">
          Enter API Key
        </label>
        <input
          id="api-key-input"
          type="password"
          value={value}
          onChange={(event) => setValue(event.target.value)}
          className="mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 font-mono text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
          autoFocus
        />
        {error && <p className="mt-1 text-sm text-danger">{error}</p>}
        <div className="mt-4 flex justify-end">
          <Button type="submit" variant="primary">
            Submit
          </Button>
        </div>
      </form>
    </Modal>
  )
}
