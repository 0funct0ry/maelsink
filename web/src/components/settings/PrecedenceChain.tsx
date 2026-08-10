const STAGES = [
  { label: 'Default', swatch: 'bg-[#9aa1b1]' },
  { label: 'Config file', swatch: 'bg-[#1a63d6]' },
  { label: 'Environment variable', swatch: 'bg-[#0f8a63]' },
  { label: 'CLI flag', swatch: 'bg-accent' },
]

/**
 * Static legend of maelsink's config precedence order. The chain itself
 * stays a fixed visual (it's always default -> file -> env -> flag), but
 * since M6.1 the values in ConfigTable below it are backed by real
 * per-key provenance (internal/config.ResolveProvenance via
 * GET /ui-api/v1/config), so the caption no longer needs to disclaim this
 * as illustrative-only.
 */
export default function PrecedenceChain() {
  return (
    <div className="rounded-md border border-border-soft bg-surface p-3.5">
      <p className="mb-1 text-[11.5px] font-semibold uppercase tracking-wide text-text-tertiary">
        Override precedence (lowest → highest)
      </p>

      <div className="flex flex-wrap items-center gap-1.5">
        {STAGES.map((stage, idx) => (
          <div key={stage.label} className="flex items-center gap-1.5">
            <span className="flex items-center gap-1.5 rounded-full border border-border-soft bg-bg px-2.5 py-1.5 text-[12.5px] font-medium text-text-primary">
              <span className={`h-2 w-2 rounded-full ${stage.swatch}`} aria-hidden="true" />
              {stage.label}
            </span>
            {idx < STAGES.length - 1 && (
              <span className="text-xs text-text-tertiary" aria-hidden="true">
                →
              </span>
            )}
          </div>
        ))}
      </div>

      <p className="mt-2.5 text-xs leading-relaxed text-text-tertiary">
        A value set by a CLI flag always wins, even if it&apos;s also set in{' '}
        <span className="font-mono">maelsink.yaml</span> or as a{' '}
        <span className="font-mono">MAELSINK_*</span> env var. The table below shows exactly
        which layer resolved each key, live, for this running server.
      </p>
    </div>
  )
}
