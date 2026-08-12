/** @type {import('tailwindcss').Config} */

// Every color below resolves through a CSS custom property (defined in
// src/index.css's :root / [data-theme="dark"] / prefers-color-scheme
// blocks) rather than a fixed hex, so the whole token layer responds to the
// dark-mode toggle uniformly (M8.7) — components keep using the same
// `bg-surface`/`text-text-primary`-style classes either way.
function themeColor(varName) {
  return `rgb(var(${varName}) / <alpha-value>)`
}

export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: themeColor('--color-bg'),
        surface: themeColor('--color-surface'),
        'surface-2': themeColor('--color-surface-2'),
        border: {
          DEFAULT: themeColor('--color-border'),
          soft: themeColor('--color-border-soft'),
        },
        'text-primary': themeColor('--color-text-primary'),
        'text-secondary': themeColor('--color-text-secondary'),
        'text-tertiary': themeColor('--color-text-tertiary'),
        accent: {
          DEFAULT: themeColor('--color-accent'),
          hover: themeColor('--color-accent-hover'),
          soft: themeColor('--color-accent-soft'),
          'soft-border': themeColor('--color-accent-soft-border'),
        },
        success: {
          DEFAULT: themeColor('--color-success'),
          soft: themeColor('--color-success-soft'),
        },
        danger: {
          DEFAULT: themeColor('--color-danger'),
          soft: themeColor('--color-danger-soft'),
        },
        warning: {
          DEFAULT: themeColor('--color-warning'),
          soft: themeColor('--color-warning-soft'),
        },
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      fontSize: {
        base: '14px',
      },
      borderRadius: {
        sm: '6px',
        md: '10px',
        lg: '16px',
      },
      boxShadow: {
        sm: '0 1px 2px rgba(26,31,54,0.04)',
        md: '0 4px 12px rgba(26,31,54,0.08)',
        lg: '0 12px 32px rgba(26,31,54,0.12)',
      },
    },
  },
  plugins: [],
}
