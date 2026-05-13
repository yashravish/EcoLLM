import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        eco: {
          950: '#030605',
          900: '#060D09',
          800: '#0A1510',
          700: '#0F1E14',
          600: '#162A1C',
          500: '#1E3826',
          400: '#2D5A3D',
          300: '#4B8A60',
          200: '#7DC49A',
          100: '#B5E4C7',
          50:  '#E4F5EC',
        },
        accent: {
          DEFAULT: '#00E87A',
          dim: '#00A855',
          bright: '#4DFFAA',
        },
        // keep for backwards compat
        brand: {
          50:  '#f0fdf4',
          100: '#dcfce7',
          500: '#22c55e',
          600: '#16a34a',
          700: '#15803d',
          900: '#14532d',
        },
      },
      fontFamily: {
        sans:  ['var(--font-outfit)', 'system-ui', 'sans-serif'],
        mono:  ['var(--font-outfit)', 'system-ui', 'sans-serif'],
        data:  ['var(--font-outfit)', 'system-ui', 'sans-serif'],
      },
      keyframes: {
        'fade-in': {
          from: { opacity: '0', transform: 'translateY(6px)' },
          to:   { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-in': {
          from: { opacity: '0', transform: 'translateX(-8px)' },
          to:   { opacity: '1', transform: 'translateX(0)' },
        },
      },
      animation: {
        'fade-in':  'fade-in 0.3s ease-out both',
        'slide-in': 'slide-in 0.25s ease-out both',
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
      },
    },
  },
  plugins: [],
};

export default config;
