import { Monitor, Moon, Sun } from 'lucide-react'
import { useThemeStore, type Theme } from '../stores/useThemeStore'

const OPTIONS: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'system', label: 'System', icon: Monitor },
]

// Ported from web/src/components/settings/ThemeToggle.tsx.
export default function ThemeToggle() {
  const theme = useThemeStore((s) => s.theme)
  const setTheme = useThemeStore((s) => s.setTheme)

  return (
    <div className="inline-flex rounded-md border border-border bg-surface p-0.5" role="group" aria-label="Theme">
      {OPTIONS.map(({ value, label, icon: Icon }) => (
        <button
          key={value}
          type="button"
          aria-pressed={theme === value}
          title={label}
          onClick={() => setTheme(value)}
          className={`flex items-center gap-1.5 rounded-sm px-2 py-1.5 text-sm font-medium transition-colors ${
            theme === value
              ? 'bg-accent text-white'
              : 'text-text-secondary hover:bg-surface-2 hover:text-text-primary'
          }`}
        >
          <Icon className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      ))}
    </div>
  )
}
