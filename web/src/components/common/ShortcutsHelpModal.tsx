import { useEffect, useState } from 'react'
import Modal from './Modal'

const SHORTCUTS: { keys: string; description: string }[] = [
  { keys: '/', description: 'Focus the search box' },
  { keys: 'Esc', description: 'Back to inbox from a message, or close a dialog' },
  { keys: '?', description: 'Show this list of shortcuts' },
]

// Mounted once in AppShell, alongside ConfirmDialog/ApiKeyModal — the "?"
// keyboard shortcuts overlay (M8.7) makes the existing "/" and "Escape"
// shortcuts (M6.1) discoverable without reading the source. Follows the
// same additive-listener, guard-on-input-focus pattern as those two rather
// than centralizing into one global handler.
export default function ShortcutsHelpModal() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== '?') return
      const active = document.activeElement
      const isTyping =
        active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement || active?.hasAttribute('contenteditable')
      if (isTyping) return
      setOpen(true)
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  return (
    <Modal open={open} onClose={() => setOpen(false)}>
      <h2 className="text-lg font-semibold text-text-primary">Keyboard shortcuts</h2>
      <dl className="mt-4 space-y-2">
        {SHORTCUTS.map((s) => (
          <div key={s.keys} className="flex items-center justify-between gap-4 text-sm">
            <dt className="text-text-secondary">{s.description}</dt>
            <dd>
              <kbd className="rounded border border-border-soft bg-surface px-1.5 py-0.5 font-mono text-xs text-text-primary">
                {s.keys}
              </kbd>
            </dd>
          </div>
        ))}
      </dl>
    </Modal>
  )
}
