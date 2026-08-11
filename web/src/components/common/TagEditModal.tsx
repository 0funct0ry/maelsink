import { useState, type KeyboardEvent } from 'react'
import Modal from './Modal'
import Button from './Button'
import TagBadge from './TagBadge'
import { useMessageStore } from '../../stores/useMessageStore'
import type { MessageDetail, MessageSummary } from '../../lib/apiTypes'

interface TagEditModalProps {
  open: boolean
  onClose: () => void
  message: MessageSummary | MessageDetail
}

// Add/remove tags on a message (M8.2). Each chip removal and each add
// applies immediately via updateTagsOptimistic — there's no separate Save
// step, matching the "add/remove via the Web UI" framing of the milestone
// and avoiding batching state the spec doesn't call for.
export default function TagEditModal({ open, onClose, message }: TagEditModalProps) {
  const updateTagsOptimistic = useMessageStore((state) => state.updateTagsOptimistic)
  const sidebarTags = useMessageStore((state) => state.sidebarTags)
  const [value, setValue] = useState('')

  const suggestions = sidebarTags.map((tc) => tc.name).filter((tag) => !message.tags.includes(tag))

  function handleRemove(tag: string) {
    void updateTagsOptimistic(message.id, [], [tag])
  }

  function handleAdd() {
    const trimmed = value.trim()
    if (!trimmed) return
    void updateTagsOptimistic(message.id, [trimmed], [])
    setValue('')
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter') {
      event.preventDefault()
      handleAdd()
    }
  }

  return (
    <Modal open={open} onClose={onClose}>
      <h2 className="text-lg font-semibold text-text-primary">Edit tags</h2>

      <div className="mt-4 flex flex-wrap gap-1.5">
        {message.tags.length === 0 && <p className="text-sm text-text-secondary">No tags yet.</p>}
        {message.tags.map((tag) => (
          <TagBadge key={tag} tag={tag} onRemove={() => handleRemove(tag)} />
        ))}
      </div>

      <div className="mt-4">
        <label htmlFor="tag-edit-input" className="block text-sm font-medium text-text-primary">
          Add a tag
        </label>
        <div className="mt-1 flex gap-2">
          <input
            id="tag-edit-input"
            type="text"
            value={value}
            list="tag-edit-suggestions"
            onChange={(event) => setValue(event.target.value)}
            onKeyDown={handleKeyDown}
            className="flex-1 rounded-md border border-border bg-surface px-3 py-2 text-sm text-text-primary focus:outline-none focus:ring-2 focus:ring-accent"
            placeholder="tag name"
          />
          <datalist id="tag-edit-suggestions">
            {suggestions.map((tag) => (
              <option key={tag} value={tag} />
            ))}
          </datalist>
          <Button variant="secondary" onClick={handleAdd}>
            Add
          </Button>
        </div>
      </div>

      <div className="mt-6 flex justify-end">
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      </div>
    </Modal>
  )
}
