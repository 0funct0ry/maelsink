import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Relative asset paths ('./') so the same build works whether served from
// '/' or a reverse-proxy subpath — the Go server injects the actual runtime
// base via a templated <base href> in index.html (SPEC.md §3.4), not Vite.
export default defineConfig({
  base: './',
  plugins: [react()],
  build: {
    // internal/webui embeds this directory directly via go:embed.
    outDir: '../internal/webui/dist',
    emptyOutDir: true,
  },
})
