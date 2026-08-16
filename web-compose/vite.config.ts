import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Relative asset paths ('./') matching web/'s config, though compose has no
// reverse-proxy base-path story in this milestone — kept for consistency.
export default defineConfig({
  base: './',
  plugins: [react()],
  build: {
    // internal/compose embeds this directory directly via go:embed.
    outDir: '../internal/compose/dist',
    emptyOutDir: true,
  },
})
