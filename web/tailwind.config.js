/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        mygo: {
          primary: '#1a1a2e',
          secondary: '#16213e',
          accent: '#e94560',
          light: '#f1f1f1',
          dark: '#0f0f1a',
        },
        tomori: '#7c3aed',
        anon: '#f59e0b',
        rana: '#10b981',
        soyo: '#ec4899',
        taki: '#3b82f6',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'typing': 'typing 1s steps(3) infinite',
      },
      keyframes: {
        typing: {
          '0%, 100%': { content: '"."' },
          '33%': { content: '".."' },
          '66%': { content: '"..."' },
        }
      }
    },
  },
  plugins: [],
}
