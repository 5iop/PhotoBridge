/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Cloudflare orange palette
        primary: {
          50: '#fff7ed',
          100: '#ffedd5',
          200: '#fed7aa',
          300: '#fdba74',
          400: '#fb923c',
          500: '#f6821f',  // Cloudflare signature orange
          600: '#ea580c',
          700: '#c2410c',
          800: '#9a3412',
          900: '#7c2d12',
        },
        // Cloudflare dark theme grays
        dark: {
          100: '#404040',
          200: '#363636',
          300: '#2c2c2c',
          400: '#242424',
          500: '#1d1d1d',
          600: '#171717',
          700: '#111111',
          800: '#0a0a0a',
          900: '#050505',
        }
      },
    },
  },
  plugins: [],
}
