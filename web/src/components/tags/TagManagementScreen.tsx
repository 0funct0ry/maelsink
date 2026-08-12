import { useEffect, useState } from 'react'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useMessageStore } from '../../stores/useMessageStore'
import { useUIStore } from '../../stores/useUIStore'
import * as apiClient from '../../lib/apiClient'
import ColorSwatchPicker from './ColorSwatchPicker'
import TagRow from './TagRow'
import Button from '../common/Button'
import type { TagStats } from '../../lib/apiTypes'

type SortKey = 'count' | 'last_used'

// Full tag management screen (M8.5), reachable from the Sidebar's "More…"
// link once more than 5 tags exist. Reads/writes the same sidebarTags list
// the Sidebar's top-5 view derives from (useMessageStore), so realtime
// tag.* WS events (handled in AppShell) keep this screen in sync with no
// polling of its own.
export default function TagManagementScreen() {
  const tags = useMessageStore((state) => state.sidebarTags)
  const fetchSidebarData = useMessageStore((state) => state.fetchSidebarData)
  const pushToast = useUIStore((state) => state.pushToast)

  const [sortKey, setSortKey] = useState<SortKey>('count')
  const [sortDesc, setSortDesc] = useState(true)
  const [newName, setNewName] = useState('')
  const [newColor, setNewColor] = useState('indigo')
  const [createError, setCreateError] = useState<string | null>(null)

  useEffect(() => {
    void fetchSidebarData()
  }, [fetchSidebarData])

  const sorted = [...tags].sort((a, b) => {
    const av = sortKey === 'count' ? a.count : a.last_used ? new Date(a.last_used).getTime() : 0
    const bv = sortKey === 'count' ? b.count : b.last_used ? new Date(b.last_used).getTime() : 0
    return sortDesc ? bv - av : av - bv
  })

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDesc((d) => !d)
    } else {
      setSortKey(key)
      setSortDesc(true)
    }
  }

  async function handleCreate() {
    const trimmed = newName.trim()
    if (!trimmed) return
    setCreateError(null)
    try {
      await apiClient.createTag(trimmed, newColor)
      setNewName('')
      void fetchSidebarData()
    } catch {
      setCreateError(`A tag named "${trimmed}" already exists, or the name is invalid.`)
    }
  }

  async function handleRename(oldName: string, newNameValue: string) {
    try {
      await apiClient.renameTag(oldName, newNameValue)
      void fetchSidebarData()
    } catch {
      pushToast('danger', `Failed to rename tag "${oldName}"`)
    }
  }

  async function handleRecolor(name: string, color: string) {
    try {
      await apiClient.recolorTag(name, color)
      void fetchSidebarData()
    } catch {
      pushToast('danger', `Failed to recolor tag "${name}"`)
    }
  }

  async function handleDelete(name: string) {
    try {
      await apiClient.deleteTag(name)
      void fetchSidebarData()
    } catch {
      pushToast('danger', `Failed to remove tag "${name}"`)
    }
  }

  async function handleDeleteWithMessages(name: string) {
    try {
      await apiClient.deleteTagWithMessages(name)
      void fetchSidebarData()
      pushToast('success', `Deleted tag "${name}" and its messages`)
    } catch {
      pushToast('danger', `Failed to delete tag "${name}" and its messages`)
    }
  }

  const sortIcon = (key: SortKey) => {
    if (sortKey !== key) return null
    return sortDesc ? <ChevronDown className="inline h-3.5 w-3.5" aria-hidden="true" /> : <ChevronUp className="inline h-3.5 w-3.5" aria-hidden="true" />
  }

  const existingNames: string[] = tags.map((t: TagStats) => t.name)

  return (
    <div>
      <div className="border-b border-border-soft px-6 py-4">
        <h1 className="mb-1 text-[19px] font-semibold tracking-tight text-text-primary">Tags</h1>
        <p className="max-w-xl text-sm leading-relaxed text-text-secondary">
          Rename, recolor, merge, or delete tags across every message.
        </p>
      </div>

      <div className="p-6">
        <div className="mb-4 flex flex-wrap items-end gap-3 rounded-md border border-border-soft bg-surface p-3">
          <div>
            <label htmlFor="new-tag-name" className="block text-xs font-medium text-text-tertiary">
              New tag
            </label>
            <input
              id="new-tag-name"
              type="text"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') void handleCreate()
              }}
              placeholder="tag name"
              className="mt-1 rounded-sm border border-border-soft bg-bg px-2 py-1.5 text-sm text-text-primary focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>
          <div>
            <span className="block text-xs font-medium text-text-tertiary">Color</span>
            <div className="mt-1">
              <ColorSwatchPicker value={newColor} onChange={setNewColor} />
            </div>
          </div>
          <Button variant="secondary" onClick={handleCreate}>
            Add tag
          </Button>
          {createError && <p className="text-sm text-danger">{createError}</p>}
        </div>

        {tags.length === 0 ? (
          <p className="text-sm text-text-secondary">No tags yet.</p>
        ) : (
          <div className="scrollbar-thin overflow-x-auto">
            <table className="w-full min-w-[520px] border-collapse rounded-md border border-border-soft text-left">
              <thead className="bg-surface">
                <tr>
                  <th className="w-6 px-3 py-2" />
                  <th className="px-3 py-2 text-xs font-semibold uppercase tracking-wide text-text-tertiary">Name</th>
                  <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-text-tertiary">
                    <button type="button" onClick={() => toggleSort('count')} className="hover:text-text-primary">
                      Count {sortIcon('count')}
                    </button>
                  </th>
                  <th className="px-3 py-2 text-xs font-semibold uppercase tracking-wide text-text-tertiary">
                    <button type="button" onClick={() => toggleSort('last_used')} className="hover:text-text-primary">
                      Last used {sortIcon('last_used')}
                    </button>
                  </th>
                  <th className="px-3 py-2" />
                </tr>
              </thead>
              <tbody>
                {sorted.map((tag) => (
                  <TagRow
                    key={tag.name}
                    tag={tag}
                    existingNames={existingNames}
                    onRename={handleRename}
                    onRecolor={handleRecolor}
                    onDelete={handleDelete}
                    onDeleteWithMessages={handleDeleteWithMessages}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
