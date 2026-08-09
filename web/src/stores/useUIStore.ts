import { create } from 'zustand'

const STORAGE_KEY = 'maelsink_api_key'

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
  clearAuthToken: () => set({ authToken: null }),
  setAuthRequired: (required, retry) =>
    set({ authRequired: required, pendingRetry: required ? retry ?? null : null }),
}))
