import { Inbox, Mail, Paperclip, AlertTriangle, Bookmark, Tag as TagIcon, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { useMessageStore } from '../../stores/useMessageStore'
import { formatBytes } from '../../lib/format'
import { paletteByToken } from '../../lib/tagColor'
import { deleteSavedSearch, listSavedSearches, saveSearch, type SavedSearch } from '../../lib/savedSearches'
import type { ListMessagesParams } from '../../lib/apiTypes'

type MailboxFilter = 'all' | 'unread' | 'attachments' | 'warnings'

/** Which MailboxFilter (if any) the current query corresponds to, so the
 * active nav item can be highlighted even after a page reload. */
function activeMailboxFilter(query: ListMessagesParams): MailboxFilter {
  if (query.read === false) return 'unread'
  if (query.has_attachments === true) return 'attachments'
  if (query.parse_warning === true) return 'warnings'
  return 'all'
}

// Per MOCKUP.html's sidebar: a "Mailbox" filter group with live counts, a
// "Saved searches" group (client-only, localStorage — SPEC.md has no
// multi-device sync requirement for the Web UI), a "Tags" nav group, and a
// storage-usage widget. Settings and the destructive "Clear all messages"
// action both live in the TopBar (gear/trash icons) so they're reachable
// from every screen, not just the Inbox — the mockup has no Settings item
// here either.
export default function Sidebar() {
  const navigate = useNavigate()
  const total = useMessageStore((state) => state.total)
  const query = useMessageStore((state) => state.query)
  const setQuery = useMessageStore((state) => state.setQuery)
  const stats = useMessageStore((state) => state.sidebarStats)
  const tags = useMessageStore((state) => state.sidebarTags)
  const fetchSidebarData = useMessageStore((state) => state.fetchSidebarData)
  const [savedSearches, setSavedSearches] = useState<SavedSearch[]>([])
  const [newSearchName, setNewSearchName] = useState('')

  // Sidebar counts/tags live in useMessageStore (M7.0) so they stay in sync
  // with realtime message.created/deleted/messages.cleared events and with
  // this tab's own mutations (markRead, delete, clear) — the store's
  // action handlers call fetchSidebarData themselves; this effect only
  // covers the very first load.
  useEffect(() => {
    void fetchSidebarData()
  }, [fetchSidebarData])

  useEffect(() => {
    setSavedSearches(listSavedSearches())
  }, [])

  const active = activeMailboxFilter(query)

  // Top 5 by usage (M8.5) — ties broken by most recent last_used, ties in
  // that broken by name for a stable order. Full list lives at /tags.
  const topTags = [...tags]
    .sort((a, b) => {
      if (b.count !== a.count) return b.count - a.count
      const aTime = a.last_used ? new Date(a.last_used).getTime() : 0
      const bTime = b.last_used ? new Date(b.last_used).getTime() : 0
      if (bTime !== aTime) return bTime - aTime
      return a.name.localeCompare(b.name)
    })
    .slice(0, 5)

  // Every sidebar nav action (mailbox filter, tag, saved search) applies its
  // query *and* navigates to the Inbox — matching "All messages", which
  // navigates via its NavLink `to="/"`. Without this, clicking a filter
  // while on Message Detail silently updated the (unmounted) list query and
  // never actually showed the filtered list.
  function applyMailboxFilter(filter: MailboxFilter) {
    setQuery({
      read: filter === 'unread' ? false : undefined,
      has_attachments: filter === 'attachments' ? true : undefined,
      parse_warning: filter === 'warnings' ? true : undefined,
      tag: undefined,
      tag_mode: undefined,
    })
    navigate('/')
  }

  function toggleTag(tag: string) {
    const current = query.tag ?? []
    const next = current.includes(tag) ? current.filter((t) => t !== tag) : [...current, tag]
    setQuery({ tag: next.length ? next : undefined, tag_mode: next.length >= 2 ? query.tag_mode : undefined })
    navigate('/')
  }

  function setTagMode(mode: 'any' | 'all') {
    setQuery({ tag_mode: mode })
  }

  function applySavedSearch(search: SavedSearch) {
    setQuery(search.query)
    navigate('/')
  }

  function handleSaveCurrentSearch() {
    const name = newSearchName.trim()
    if (!name) return
    setSavedSearches(saveSearch(name, query))
    setNewSearchName('')
  }

  function handleDeleteSavedSearch(name: string) {
    setSavedSearches(deleteSavedSearch(name))
  }

  const navItemClass = (isActiveItem: boolean) =>
    `flex w-full items-center justify-between gap-2.5 rounded-sm px-2 py-[7px] text-left text-[13.5px] font-medium transition-colors ${
      isActiveItem ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface hover:text-text-primary'
    }`

  const countBadgeClass = (isActiveItem: boolean) =>
    `rounded-[5px] px-1.5 font-mono text-[11.5px] ${isActiveItem ? 'bg-white text-accent' : 'bg-surface text-text-tertiary'}`

  return (
    <aside className="scrollbar-thin hidden w-[216px] flex-none flex-col gap-[22px] overflow-y-auto border-r border-border bg-bg p-3 md:flex">
      <div>
        <div className="mb-1.5 px-2 text-[11px] font-semibold uppercase tracking-[0.04em] text-text-tertiary">
          Mailbox
        </div>
        <nav className="flex flex-col gap-px">
          <NavLink
            to="/"
            end
            className={({ isActive: routeActive }) => navItemClass(routeActive && active === 'all')}
            onClick={() => applyMailboxFilter('all')}
          >
            {({ isActive: routeActive }) => {
              const isActiveItem = routeActive && active === 'all'
              return (
                <>
                  <span className="flex items-center gap-2.5">
                    <Inbox className={`h-4 w-4 ${isActiveItem ? 'opacity-100' : 'opacity-85'}`} aria-hidden="true" />
                    All messages
                  </span>
                  <span className={countBadgeClass(isActiveItem)}>{stats ? stats.total_messages : total}</span>
                </>
              )
            }}
          </NavLink>

          <button type="button" className={navItemClass(active === 'unread')} onClick={() => applyMailboxFilter('unread')}>
            <span className="flex items-center gap-2.5">
              <Mail className="h-4 w-4 opacity-85" aria-hidden="true" />
              Unread
            </span>
            {stats && <span className={countBadgeClass(active === 'unread')}>{stats.unread_count}</span>}
          </button>

          <button
            type="button"
            className={navItemClass(active === 'attachments')}
            onClick={() => applyMailboxFilter('attachments')}
          >
            <span className="flex items-center gap-2.5">
              <Paperclip className="h-4 w-4 opacity-85" aria-hidden="true" />
              With attachments
            </span>
            {stats && <span className={countBadgeClass(active === 'attachments')}>{stats.attachment_count}</span>}
          </button>

          <button
            type="button"
            className={navItemClass(active === 'warnings')}
            onClick={() => applyMailboxFilter('warnings')}
          >
            <span className="flex items-center gap-2.5">
              <AlertTriangle className="h-4 w-4 opacity-85" aria-hidden="true" />
              Parse warnings
            </span>
            {stats && <span className={countBadgeClass(active === 'warnings')}>{stats.parse_warning_count}</span>}
          </button>
        </nav>
      </div>

      <div>
        <div className="mb-1.5 flex items-center justify-between px-2 text-[11px] font-semibold uppercase tracking-[0.04em] text-text-tertiary">
          Saved searches
        </div>
        <nav className="flex flex-col gap-px">
          {savedSearches.map((s) => (
            <div key={s.name} className="group flex items-center gap-1">
              <button
                type="button"
                className="flex flex-1 items-center gap-2.5 truncate rounded-sm px-2 py-[7px] text-left text-[13px] text-text-secondary transition-colors hover:bg-surface hover:text-text-primary"
                onClick={() => applySavedSearch(s)}
              >
                <Bookmark className="h-3.5 w-3.5 flex-none opacity-70" aria-hidden="true" />
                <span className="truncate">{s.name}</span>
              </button>
              <button
                type="button"
                aria-label={`Delete saved search ${s.name}`}
                className="flex-none px-1 text-text-tertiary opacity-0 transition-opacity hover:text-danger group-hover:opacity-100"
                onClick={() => handleDeleteSavedSearch(s.name)}
              >
                <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            </div>
          ))}
          <div className="flex items-center gap-1 px-2 py-1">
            <input
              type="text"
              value={newSearchName}
              onChange={(e) => setNewSearchName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSaveCurrentSearch()
              }}
              placeholder="Save current search"
              aria-label="Saved search name"
              className="w-full rounded-sm border border-border-soft bg-surface px-2 py-1 text-[12px] text-text-primary placeholder:text-text-tertiary focus:outline-none focus:ring-1 focus:ring-accent"
            />
            <button
              type="button"
              onClick={handleSaveCurrentSearch}
              disabled={!newSearchName.trim()}
              className="flex-none rounded-sm px-1.5 py-1 text-[11.5px] font-medium text-accent disabled:opacity-40"
            >
              Save
            </button>
          </div>
        </nav>
      </div>

      {tags.length > 0 && (
        <div>
          <div className="mb-1.5 px-2 text-[11px] font-semibold uppercase tracking-[0.04em] text-text-tertiary">
            Tags
          </div>
          <nav className="flex flex-col gap-px">
            {topTags.map((tc) => {
              const color = paletteByToken(tc.color)
              const isActiveItem = (query.tag ?? []).includes(tc.name)
              return (
                <button
                  key={tc.name}
                  type="button"
                  aria-pressed={isActiveItem}
                  className={navItemClass(isActiveItem)}
                  onClick={() => toggleTag(tc.name)}
                >
                  <span className="flex min-w-0 items-center gap-2.5 truncate">
                    <span className={`h-[7px] w-[7px] flex-none rounded-full ${color.dot}`} aria-hidden="true" />
                    <TagIcon className="sr-only" aria-hidden="true" />
                    <span className="truncate">{tc.name}</span>
                  </span>
                  <span className={countBadgeClass(isActiveItem)}>{tc.count}</span>
                </button>
              )
            })}
          </nav>
          {tags.length > 5 && (
            <NavLink
              to="/tags"
              className="mt-1 block px-2 py-[7px] text-left text-[13px] font-medium text-accent hover:underline"
            >
              More…
            </NavLink>
          )}
          {(query.tag?.length ?? 0) >= 2 && (
            <div className="mt-1.5 flex items-center gap-1 px-2">
              <span className="text-[11px] text-text-tertiary">Match:</span>
              <div className="flex overflow-hidden rounded-sm border border-border-soft">
                <button
                  type="button"
                  aria-pressed={(query.tag_mode ?? 'any') === 'any'}
                  onClick={() => setTagMode('any')}
                  className={`px-2 py-0.5 text-[11px] font-medium ${
                    (query.tag_mode ?? 'any') === 'any' ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface'
                  }`}
                >
                  Any
                </button>
                <button
                  type="button"
                  aria-pressed={query.tag_mode === 'all'}
                  onClick={() => setTagMode('all')}
                  className={`px-2 py-0.5 text-[11px] font-medium ${
                    query.tag_mode === 'all' ? 'bg-accent-soft text-accent' : 'text-text-secondary hover:bg-surface'
                  }`}
                >
                  All
                </button>
              </div>
            </div>
          )}
        </div>
      )}

      {stats && (
        <div className="flex flex-col gap-2 rounded-md border border-border-soft bg-surface p-3">
          <div className="flex items-baseline justify-between">
            <span className="text-[11.5px] text-text-tertiary">Storage used</span>
            <span className="font-mono text-xs font-medium text-text-primary">
              {formatBytes(stats.total_size_bytes)}
            </span>
          </div>
          {/* Decorative only — no configured storage limit is exposed via
              any endpoint today, so this can't reflect a real percentage. */}
          <div className="h-[5px] overflow-hidden rounded-[3px] bg-border-soft">
            <div className="h-full w-[12%] rounded-[3px] bg-accent" />
          </div>
          <div className="flex items-baseline justify-between">
            <span className="text-[11.5px] text-text-tertiary">maelsink.db</span>
            <span className="font-mono text-xs font-medium text-text-primary">{stats.total_messages} msgs</span>
          </div>
        </div>
      )}
    </aside>
  )
}
