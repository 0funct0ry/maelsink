import { useState } from 'react'
import Button from '../components/Button'
import { useVarsStore } from '../stores/useVarsStore'

// Vars screen (SPEC.md §7.7.1): a key/value table backed entirely by
// browser localStorage via useVarsStore — no backend calls happen here at
// all.
export default function VarsScreen() {
  const vars = useVarsStore((s) => s.vars)
  const setVar = useVarsStore((s) => s.setVar)
  const deleteVar = useVarsStore((s) => s.deleteVar)

  const [newKey, setNewKey] = useState('')
  const [newValue, setNewValue] = useState('')

  function handleAdd() {
    const key = newKey.trim()
    if (!key) return
    setVar(key, newValue)
    setNewKey('')
    setNewValue('')
  }

  const entries = Object.entries(vars)

  return (
    <div className="flex h-full flex-col p-4">
      <h1 className="mb-4 text-lg font-semibold text-text-primary">Vars</h1>
      <p className="mb-4 text-sm text-text-secondary">
        Session variables live only in this browser's storage — they are never sent to the compose
        server except as part of a render/send request (arriving in M13.1).
      </p>

      {entries.length === 0 ? (
        <p className="text-sm text-text-secondary">No variables yet.</p>
      ) : (
        <table className="mb-4 w-full text-sm">
          <thead>
            <tr className="border-b border-border-soft text-left text-text-secondary">
              <th className="py-2">Key</th>
              <th className="py-2">Value</th>
              <th className="py-2" />
            </tr>
          </thead>
          <tbody>
            {entries.map(([key, value]) => (
              <tr key={key} className="border-b border-border-soft">
                <td className="py-2 font-mono text-text-primary">{key}</td>
                <td className="py-2">
                  <input
                    className="w-full rounded-md border border-border bg-bg px-2 py-1 text-text-primary"
                    value={value}
                    onChange={(e) => setVar(key, e.target.value)}
                  />
                </td>
                <td className="py-2 text-right">
                  <Button variant="ghost" onClick={() => deleteVar(key)}>
                    Delete
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <div className="flex items-end gap-2">
        <div className="flex-1">
          <label htmlFor="vars-new-key" className="mb-1 block text-xs text-text-secondary">
            Key
          </label>
          <input
            id="vars-new-key"
            className="w-full rounded-md border border-border bg-bg px-2 py-1 text-text-primary"
            value={newKey}
            onChange={(e) => setNewKey(e.target.value)}
          />
        </div>
        <div className="flex-1">
          <label htmlFor="vars-new-value" className="mb-1 block text-xs text-text-secondary">
            Value
          </label>
          <input
            id="vars-new-value"
            className="w-full rounded-md border border-border bg-bg px-2 py-1 text-text-primary"
            value={newValue}
            onChange={(e) => setNewValue(e.target.value)}
          />
        </div>
        <Button onClick={handleAdd}>Add</Button>
      </div>
    </div>
  )
}
