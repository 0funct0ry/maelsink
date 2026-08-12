import { create } from 'zustand'
import type { WsStatus } from '../lib/wsClient'

const STORAGE_KEY = 'maelsink_api_key'
const THEME_STORAGE_KEY = 'maelsink_theme'

export type Theme = 'light' | 'dark' | 'system'

export type ToastVariant = 'info' | 'success' | 'danger'

export interface Toast {
  id: string
  variant: ToastVariant
  message: string
}

export interface ConfirmModal {
  kind: 'confirm'
  title: string
  body: string
  confirmLabel?: string
  danger?: boolean
  onConfirm: () => void
}

let toastCounter = 0
function nextToastId(): string {
  toastCounter += 1
  return `toast-${toastCounter}`
}

// The API key is kept in localStorage rather than a server-side session
// because maelsink has no backend session mechanism to put it behind — it's
// a local dev tool per SPEC.md §12, not a hosted multi-user service. The
// tradeoff: any script running on this origin (or with filesystem access to
// the browser profile) can read it, so clearAuthToken below exists to let a
// developer deliberately purge it when that risk matters to them.
function readStoredToken(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

function writeStoredToken(token: string): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, token)
  } catch {
    // localStorage unavailable (private mode, disabled storage) — the
    // session still works, the key just won't survive a reload.
  }
}

function removeStoredToken(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // localStorage unavailable — nothing to remove.
  }
}

function readStoredTheme(): Theme {
  try {
    const v = window.localStorage.getItem(THEME_STORAGE_KEY)
    if (v === 'light' || v === 'dark' || v === 'system') return v
  } catch {
    // localStorage unavailable — fall through to the default.
  }
  return 'system'
}

function writeStoredTheme(theme: Theme): void {
  try {
    window.localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // localStorage unavailable — the choice just won't survive a reload.
  }
}

// Applies theme to the document root as a data-theme attribute, which
// src/index.css's [data-theme="dark"]/[data-theme="light"] blocks key off
// of. 'system' removes the attribute entirely so the prefers-color-scheme
// media query (also in index.css) takes over.
function applyTheme(theme: Theme): void {
  const root = window.document.documentElement
  if (theme === 'system') {
    root.removeAttribute('data-theme')
  } else {
    root.setAttribute('data-theme', theme)
  }
}

interface UIState {
  modal: ConfirmModal | null
  openConfirm: (opts: Omit<ConfirmModal, 'kind'>) => void
  closeModal: () => void

  toasts: Toast[]
  pushToast: (variant: ToastVariant, message: string) => void
  dismissToast: (id: string) => void

  authToken: string | null
  authRequired: boolean
  pendingRetry: (() => void) | null
  setAuthToken: (token: string) => void
  clearAuthToken: () => void
  setAuthRequired: (required: boolean, retry?: () => void) => void

  /** Live status of the app-wide /ws connection (M7.0), owned by AppShell
   * so it survives route changes — read by TopBar to show a "reconnecting"
   * indicator regardless of which screen is currently active. */
  wsStatus: WsStatus
  setWsStatus: (status: WsStatus) => void

  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useUIStore = create<UIState>((set, get) => ({
  modal: null,
  openConfirm: (opts) => set({ modal: { kind: 'confirm', ...opts } }),
  closeModal: () => set({ modal: null }),

  toasts: [],
  pushToast: (variant, message) =>
    set((state) => ({ toasts: [...state.toasts, { id: nextToastId(), variant, message }] })),
  dismissToast: (id) => set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) })),

  authToken: readStoredToken(),
  authRequired: false,
  pendingRetry: null,
  setAuthToken: (token) => {
    writeStoredToken(token)
    const retry = get().pendingRetry
    set({ authToken: token, authRequired: false, pendingRetry: null })
    retry?.()
  },
  clearAuthToken: () => {
    removeStoredToken()
    set({ authToken: null })
  },
  setAuthRequired: (required, retry) =>
    set({ authRequired: required, pendingRetry: required ? retry ?? null : null }),

  wsStatus: 'connecting',
  setWsStatus: (status) => set({ wsStatus: status }),

  theme: readStoredTheme(),
  setTheme: (theme) => {
    writeStoredTheme(theme)
    applyTheme(theme)
    set({ theme })
  },
}))

// Apply whatever theme was already persisted as soon as the module loads,
// so the very first paint uses it instead of a light-mode flash.
applyTheme(useUIStore.getState().theme)
