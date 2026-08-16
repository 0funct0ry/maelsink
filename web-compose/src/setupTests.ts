import '@testing-library/jest-dom'

// jsdom doesn't implement matchMedia — stub it so code that reads
// prefers-color-scheme (e.g. lib/useIsDarkMode.ts) doesn't crash under test.
if (typeof window.matchMedia !== 'function') {
  window.matchMedia = (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}
