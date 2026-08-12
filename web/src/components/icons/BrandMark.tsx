// maelsink's brand mark: an envelope draining into a funnel/sink, per
// internal-docs/MOCKUP.html's .brand-mark svg. Kept here rather than
// inlined in TopBar so the favicon/manifest generation (M8.7) and any
// future reuse share one source of truth for the glyph. This is the one
// bespoke icon in the app per STYLE_GUIDE.md §2.1 — every other icon stays
// Lucide.
export default function BrandMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
      aria-hidden="true"
    >
      <path
        d="M3 6.5L12 13L21 6.5"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path
        d="M3 6.5C3 5.4 3.9 4.5 5 4.5H19C20.1 4.5 21 5.4 21 6.5V15C21 17.5 18.5 20 15 20H9C5.5 20 3 17.5 3 15V6.5Z"
        stroke="currentColor"
        strokeWidth="1.8"
        strokeLinejoin="round"
      />
    </svg>
  )
}
