/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        bg: {
          DEFAULT: '#0A0A0C',
          card: '#111116',
          elevated: '#16161C',
          overlay: '#1C1C24',
        },
        border: {
          DEFAULT: '#242430',
          bright: '#333344',
          active: '#FF6B00',
        },
        text: {
          primary: '#E8E4DC',
          secondary: '#9A9890',
          dim: '#6A6860',
        },
        amber: {
          DEFAULT: '#FF6B00',
          bright: '#FF8C00',
          dim: 'rgba(255,107,0,0.1)',
          glow: 'rgba(255,107,0,0.35)',
        },
        cyan: {
          DEFAULT: '#00D4A8',
          bright: '#00FFCC',
          dim: 'rgba(0,212,168,0.1)',
          glow: 'rgba(0,212,168,0.3)',
        },
        // Semantic aliases (for backwards compat with any classnames)
        accent: '#FF6B00',
        success: '#00D4A8',
        warning: '#FFD600',
        error: '#FF3D5A',
        surface: {
          DEFAULT: '#0A0A0C',
          secondary: '#111116',
          elevated: '#16161C',
        },
      },
      fontFamily: {
        display: ['"Chakra Petch"', 'sans-serif'],
        sans: ['"IBM Plex Mono"', 'monospace'],
        mono: ['"IBM Plex Mono"', 'monospace'],
      },
      boxShadow: {
        amber: '0 0 12px rgba(255,107,0,0.4), 0 0 35px rgba(255,107,0,0.12)',
        'amber-sm': '0 0 6px rgba(255,107,0,0.3)',
        cyan: '0 0 12px rgba(0,212,168,0.4), 0 0 35px rgba(0,212,168,0.12)',
        'cyan-sm': '0 0 6px rgba(0,212,168,0.3)',
        card: '0 0 0 1px rgba(255,255,255,0.03), 0 4px 24px rgba(0,0,0,0.6)',
        soft: '0 4px 16px rgba(0,0,0,0.5)',
      },
      animation: {
        'glow-amber': 'glow-amber 2s ease-in-out infinite',
        'glow-cyan': 'glow-cyan 2s ease-in-out infinite',
        'fade-slide': 'fade-slide 0.3s ease-out forwards',
        'status-pulse': 'status-pulse 1.8s ease-in-out infinite',
      },
      keyframes: {
        'glow-amber': {
          '0%,100%': { boxShadow: '0 0 6px rgba(255,107,0,0.3)' },
          '50%': { boxShadow: '0 0 18px rgba(255,107,0,0.65), 0 0 40px rgba(255,107,0,0.2)' },
        },
        'glow-cyan': {
          '0%,100%': { boxShadow: '0 0 6px rgba(0,212,168,0.3)' },
          '50%': { boxShadow: '0 0 18px rgba(0,212,168,0.65), 0 0 40px rgba(0,212,168,0.2)' },
        },
        'fade-slide': {
          '0%': { opacity: '0', transform: 'translateY(6px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'status-pulse': {
          '0%,100%': { opacity: '1', transform: 'scale(1)' },
          '50%': { opacity: '0.5', transform: 'scale(0.8)' },
        },
      },
    },
  },
  plugins: [],
}
