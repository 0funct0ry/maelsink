import { ShieldCheck } from 'lucide-react'
import Button from '../common/Button'
import { useUIStore } from '../../stores/useUIStore'

// Surfaces the one security-relevant action a developer can take on the
// client side: purging the API key this browser has cached in localStorage
// (M8.7 abstraction-leakage hardening). There's no server-side session to
// revoke instead — see the comment above readStoredToken in useUIStore.
export default function SecurityCard() {
  const authToken = useUIStore((state) => state.authToken)
  const clearAuthToken = useUIStore((state) => state.clearAuthToken)
  const openConfirm = useUIStore((state) => state.openConfirm)

  return (
    <div className="rounded-lg border border-border bg-surface p-4 shadow-sm">
      <div className="mb-3 flex items-center gap-2">
        <ShieldCheck className="h-4 w-4 text-text-tertiary" aria-hidden="true" />
        <h2 className="text-sm font-semibold text-text-primary">Security</h2>
      </div>

      <p className="text-sm text-text-secondary">
        The API key you enter to authenticate is stored in this browser&apos;s
        localStorage. Clear it if you&apos;re done using a shared or public machine.
      </p>

      <div className="mt-3">
        <Button
          variant="secondary"
          disabled={!authToken}
          onClick={() =>
            openConfirm({
              title: 'Clear stored API key?',
              body: "This removes the API key saved in this browser. You'll need to re-enter it next time authentication is required.",
              confirmLabel: 'Clear stored key',
              danger: true,
              onConfirm: clearAuthToken,
            })
          }
        >
          Clear stored key
        </Button>
      </div>
    </div>
  )
}
