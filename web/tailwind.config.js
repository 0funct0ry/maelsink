/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: '#ffffff',
        surface: '#f7f7fa',
        'surface-2': '#f0f0f5',
        border: {
          DEFAULT: '#e3e5ea',
          soft: '#edeef2',
        },
        'text-primary': '#1a1f36',
        'text-secondary': '#6b7280',
        'text-tertiary': '#9aa1b1',
        accent: {
          DEFAULT: '#635bff',
          hover: '#514adb',
          soft: '#eeecff',
          'soft-border': '#d9d5ff',
        },
        success: {
          DEFAULT: '#1f9254',
          soft: '#e3f8ec',
        },
        danger: {
          DEFAULT: '#df1b41',
          soft: '#feeaec',
        },
        warning: {
          DEFAULT: '#b25e09',
          soft: '#fdf0dd',
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
