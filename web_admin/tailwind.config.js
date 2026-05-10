/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      fontFamily: {
        mono: ['"JetBrains Mono"', '"Fira Code"', 'monospace'],
        display: ['"Space Mono"', 'monospace'],
        sans: ['"DM Sans"', 'sans-serif'],
      },
      colors: {
        space: {
          950: '#020408',
          900: '#060d14',
          800: '#0a1628',
          700: '#0f2040',
          600: '#152b55',
        },
        cyan: {
          400: '#22d3ee',
          300: '#67e8f9',
          500: '#06b6d4',
        },
        amber: {
          400: '#fbbf24',
          300: '#fcd34d',
        },
        emerald: {
          400: '#34d399',
          500: '#10b981',
        },
        rose: {
          400: '#fb7185',
          500: '#f43f5e',
        }
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'spin-slow': 'spin 8s linear infinite',
        'fade-in': 'fadeIn 0.4s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'blink': 'blink 1.2s step-end infinite',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        blink: {
          '0%, 100%': { opacity: '1' },
          '50%': { opacity: '0' },
        },
      },
    },
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        vnu: {
          'primary': '#22d3ee',
          'primary-content': '#020408',
          'secondary': '#fbbf24',
          'secondary-content': '#020408',
          'accent': '#34d399',
          'accent-content': '#020408',
          'neutral': '#0a1628',
          'neutral-content': '#94a3b8',
          'base-100': '#060d14',
          'base-200': '#0a1628',
          'base-300': '#0f2040',
          'base-content': '#e2e8f0',
          'info': '#06b6d4',
          'success': '#10b981',
          'warning': '#fbbf24',
          'error': '#f43f5e',
        },
      },
    ],
  },
};
