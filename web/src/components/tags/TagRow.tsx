import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { createPortal } from 'react-dom'
import { Check, MailX, Palette, Pencil, Trash2, X } from 'lucide-react'
import { paletteByToken } from '../../lib/tagColor'
import { useUIStore } from '../../stores/useUIStore'
import ColorSwatchPicker from './ColorSwatchPicker'
import type { TagStats } from '../../lib/apiTypes'

// Anchored from the top (opens downward) or bottom (opens upward, for rows
// near the bottom of the viewport) — never both.
type PickerPosition = { right: number } & ({ top: number; bottom?: undefined } | { bottom: number; top?: undefined })

// Rough height of the swatch grid (2 rows of 24px swatches + padding) —
// good enough for an open/upward-vs-downward decision.
const ESTIMATED_PICKER_HEIGHT = 90

interface TagRowProps {
  tag: TagStats
  existingNames: string[]
  onRename: (oldName: string, newName: string) => void
  onRecolor: (name: string, color: string) => void
  onDelete: (name: string) => void
  onDeleteWithMessages: (name: string) => void
}

function formatLastUsed(lastUsed: string | null): string {
  if (!lastUsed) return '—'
  return new Date(lastUsed).toLocaleString()
}

// One row of TagManagementScreen's table: inline rename, a recolor swatch
// popover, and the two delete variants (untag-only vs. delete-with-messages)
// — all destructive/merge-risking actions route through the app's
// ConfirmDialog (useUIStore.openConfirm), never window.confirm.
export default function TagRow({ tag, existingNames, onRename, onRecolor, onDelete, onDeleteWithMessages }: TagRowProps) {
  const openConfirm = useUIStore((state) => state.openConfirm)
  const [editing, setEditing] = useState(false)
  const [nameValue, setNameValue] = useState(tag.name)
  const [colorPickerOpen, setColorPickerOpen] = useState(false)
  const [pickerPosition, setPickerPosition] = useState<PickerPosition | null>(null)
  const colorPickerRef = useRef<HTMLDivElement>(null)
  const colorTriggerRef = useRef<HTMLButtonElement>(null)

  const color = paletteByToken(tag.color)

  useEffect(() => {
    if (!colorPickerOpen) return
    function handlePointerDown(event: PointerEvent) {
      const target = event.target as Node
      if (colorPickerRef.current?.contains(target) || colorTriggerRef.current?.contains(target)) return
      setColorPickerOpen(false)
    }
    function handleReposition() {
      setColorPickerOpen(false)
    }
    document.addEventListener('pointerdown', handlePointerDown)
    // Position is computed once at open time (below); if the table (or the
    // page) scrolls or resizes while open, close rather than show a
    // stale/misaligned picker.
    window.addEventListener('scroll', handleReposition, true)
    window.addEventListener('resize', handleReposition)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      window.removeEventListener('scroll', handleReposition, true)
      window.removeEventListener('resize', handleReposition)
    }
  }, [colorPickerOpen])

  function toggleColorPicker() {
    if (!colorPickerOpen && colorTriggerRef.current) {
      const rect = colorTriggerRef.current.getBoundingClientRect()
      const right = window.innerWidth - rect.right
      const fitsBelow = rect.bottom + 4 + ESTIMATED_PICKER_HEIGHT <= window.innerHeight
      setPickerPosition(
        fitsBelow ? { top: rect.bottom + 4, right } : { bottom: window.innerHeight - rect.top + 4, right },
      )
    }
    setColorPickerOpen((v) => !v)
  }

  function startEdit() {
    setNameValue(tag.name)
    setEditing(true)
  }

  function cancelEdit() {
    setEditing(false)
    setNameValue(tag.name)
  }

  function commitRename() {
    const trimmed = nameValue.trim()
    setEditing(false)
    if (!trimmed || trimmed === tag.name) {
      setNameValue(tag.name)
      return
    }
    const collides = existingNames.some((n) => n !== tag.name && n === trimmed)
    if (collides) {
      openConfirm({
        title: 'Merge tags?',
        body: `A tag named "${trimmed}" already exists. Renaming "${tag.name}" to "${trimmed}" will merge them: every message tagged with either will end up tagged "${trimmed}", and "${tag.name}" will no longer exist.`,
        confirmLabel: 'Merge',
        onConfirm: () => onRename(tag.name, trimmed),
      })
      return
    }
    onRename(tag.name, trimmed)
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter') {
      event.preventDefault()
      commitRename()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      cancelEdit()
    }
  }

  function handleDelete() {
    openConfirm({
      title: 'Remove tag from all messages?',
      body: `"${tag.name}" will be removed from ${tag.count} message${tag.count === 1 ? '' : 's'}. The messages themselves are not deleted.`,
      confirmLabel: 'Remove tag',
      onConfirm: () => onDelete(tag.name),
    })
  }

  function handleDeleteWithMessages() {
    openConfirm({
      title: 'Delete tag and its messages?',
      body: `This permanently deletes ${tag.count} message${tag.count === 1 ? '' : 's'} tagged "${tag.name}", along with the tag itself. This cannot be undone.`,
      confirmLabel: 'Delete messages',
      danger: true,
      onConfirm: () => onDeleteWithMessages(tag.name),
    })
  }

  return (
    <tr className="border-b border-border-soft last:border-b-0">
      <td className="w-6 px-3 py-2">
        <span className={`inline-block h-2.5 w-2.5 rounded-full ${color.dot}`} aria-hidden="true" />
      </td>
      <td className="px-3 py-2">
        {editing ? (
          <div className="flex items-center gap-1">
            <input
              autoFocus
              value={nameValue}
              onChange={(e) => setNameValue(e.target.value)}
              onKeyDown={handleKeyDown}
              onBlur={commitRename}
              aria-label={`Rename tag ${tag.name}`}
              className="rounded-sm border border-border-soft bg-surface px-2 py-1 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <button type="button" aria-label="Confirm rename" onClick={commitRename} className="text-text-tertiary hover:text-text-primary">
              <Check className="h-4 w-4" aria-hidden="true" />
            </button>
            <button type="button" aria-label="Cancel rename" onClick={cancelEdit} className="text-text-tertiary hover:text-text-primary">
              <X className="h-4 w-4" aria-hidden="true" />
            </button>
          </div>
        ) : (
          <button type="button" onClick={startEdit} className="text-sm text-text-primary hover:underline">
            {tag.name}
          </button>
        )}
      </td>
      <td className="px-3 py-2 text-right font-mono text-sm text-text-secondary">{tag.count}</td>
      <td className="px-3 py-2 text-sm text-text-secondary">{formatLastUsed(tag.last_used)}</td>
      <td className="px-3 py-2">
        <div className="flex items-center justify-end gap-2">
          <div className="relative">
            <button
              ref={colorTriggerRef}
              type="button"
              aria-label={`Recolor tag ${tag.name}`}
              title="Recolor tag"
              onClick={toggleColorPicker}
              className="text-text-tertiary hover:text-text-primary"
            >
              <Palette className="h-4 w-4" aria-hidden="true" />
            </button>
            {colorPickerOpen &&
              pickerPosition &&
              createPortal(
                <div
                  ref={colorPickerRef}
                  style={
                    pickerPosition.top !== undefined
                      ? { top: pickerPosition.top, right: pickerPosition.right }
                      : { bottom: pickerPosition.bottom, right: pickerPosition.right }
                  }
                  className="fixed z-50 rounded-md border border-border bg-surface p-2 shadow-md"
                >
                  <ColorSwatchPicker
                    value={tag.color}
                    onChange={(color) => {
                      onRecolor(tag.name, color)
                      setColorPickerOpen(false)
                    }}
                  />
                </div>,
                document.body,
              )}
          </div>
          <button
            type="button"
            aria-label={`Edit tag ${tag.name}`}
            title="Edit tag"
            onClick={startEdit}
            className="text-text-tertiary hover:text-text-primary"
          >
            <Pencil className="h-4 w-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label={`Remove tag ${tag.name}`}
            title="Remove tag"
            onClick={handleDelete}
            className="text-text-tertiary hover:text-text-primary"
          >
            <Trash2 className="h-4 w-4" aria-hidden="true" />
          </button>
          <button
            type="button"
            aria-label={`Delete tag ${tag.name} and its messages`}
            title="Delete tag and messages"
            onClick={handleDeleteWithMessages}
            className="text-danger hover:text-danger/80"
          >
            <MailX className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </td>
    </tr>
  )
}
