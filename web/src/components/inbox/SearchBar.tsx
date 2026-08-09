import { Search, X } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useMessageStore } from '../../stores/useMessageStore'

const DEBOUNCE_MS = 300

// Lives in the TopBar (per MOCKUP.html's global topbar-search), so it's
// reachable from any screen — typing here always jumps back to the Inbox.
export default function SearchBar() {
  const q = useMessageStore((state) => state.query.q ?? '')
  const setQuery = useMessageStore((state) => state.setQuery)
  const [value, setValue] = useState(q)
  const [focused, setFocused] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    setValue(q)
  }, [q])

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      const active = document.activeElement
      const isTyping = active instanceof HTMLInputElement || active instanceof HTMLTextAreaElement
      if (e.key === '/' && !isTyping) {
        e.preventDefault()
        inputRef.current?.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  function handleChange(next: string) {
    setValue(next)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => {
      setQuery({ q: next })
      if (location.pathname !== '/') navigate('/')
    }, DEBOUNCE_MS)
  }

  function handleClear() {
    if (timerRef.current) clearTimeout(timerRef.current)
    setValue('')
    setQuery({ q: '' })
  }

  return (
    <div
      className={`flex h-[34px] w-full max-w-[460px] items-center gap-2 rounded-md border px-3 transition-colors ${
        focused ? 'border-accent ring-2 ring-accent-soft' : 'border-border-soft hover:border-border'
      } bg-surface`}
    >
      <Search className="h-[15px] w-[15px] flex-none text-text-tertiary" aria-hidden="true" />
      <input
        ref={inputRef}
        type="text"
        role="searchbox"
        aria-label="Search messages"
        placeholder="Search subject, from, or to…"
        value={value}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        onChange={(e) => handleChange(e.target.value)}
        className="w-full bg-transparent text-[13.5px] text-text-primary placeholder:text-text-tertiary focus:outline-none"
      />
      {value.length > 0 ? (
        <button
          type="button"
          aria-label="Clear search"
          onClick={handleClear}
          className="flex h-5 w-5 flex-none items-center justify-center rounded-full text-text-tertiary hover:bg-surface-2"
        >
          <X className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      ) : (
        <span className="flex-none rounded border border-border-soft bg-bg px-1 font-mono text-[11px] text-text-tertiary">
          /
        </span>
      )}
    </div>
  )
}
