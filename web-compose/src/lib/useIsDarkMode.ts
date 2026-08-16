import { useSyncExternalStore } from 'react'
import { useThemeStore } from '../stores/useThemeStore'

// Resolves useThemeStore's 'light'|'dark'|'system' choice down to an actual
// boolean, tracking the OS-level prefers-color-scheme media query for
// 'system' — needed by anything (like CodeMirror's theme prop) that can't
// just key off the [data-theme] CSS attribute the way Tailwind classes do.
function getSystemPrefersDark(): boolean {
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function subscribeToSystemScheme(callback: () => void): () => void {
  const mql = window.matchMedia('(prefers-color-scheme: dark)')
  mql.addEventListener('change', callback)
  return () => mql.removeEventListener('change', callback)
}

export function useIsDarkMode(): boolean {
  const theme = useThemeStore((s) => s.theme)
  const systemPrefersDark = useSyncExternalStore(subscribeToSystemScheme, getSystemPrefersDark, () => false)

  if (theme === 'dark') return true
  if (theme === 'light') return false
  return systemPrefersDark
}
