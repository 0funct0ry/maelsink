import { Check } from 'lucide-react'
import { PALETTE_TOKENS, paletteByToken } from '../../lib/tagColor'

interface ColorSwatchPickerProps {
  value: string
  onChange: (color: string) => void
}

// 8-swatch picker over the fixed persisted-color enum (M8.5) — no free-form
// hex entry, per SPEC.md §5.2's tag color contract.
export default function ColorSwatchPicker({ value, onChange }: ColorSwatchPickerProps) {
  return (
    <div className="flex flex-wrap gap-1.5" role="group" aria-label="Tag color">
      {PALETTE_TOKENS.map((token) => {
        const color = paletteByToken(token)
        const selected = token === value
        return (
          <button
            key={token}
            type="button"
            aria-label={token}
            aria-pressed={selected}
            onClick={() => onChange(token)}
            className={`flex h-6 w-6 items-center justify-center rounded-full ${color.dot} ${
              selected ? 'ring-2 ring-accent ring-offset-1' : ''
            }`}
          >
            {selected && <Check className="h-3.5 w-3.5 text-white" aria-hidden="true" />}
          </button>
        )
      })}
    </div>
  )
}
