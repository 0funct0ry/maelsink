// 56px top bar per STYLE_GUIDE.md §1.4. Wordmark is always set in mono
// (STYLE_GUIDE.md §1.2).
export default function TopBar() {
  return (
    <header className="flex h-14 shrink-0 items-center border-b border-border bg-bg px-5">
      <span className="font-mono text-[15px] font-semibold tracking-[-0.02em] text-text-primary">
        maelsink
      </span>
    </header>
  )
}
