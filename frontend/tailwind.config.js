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
        // Cloudflare light theme
        cf: {
          bg: '#f4f5f7',        // Main background
          card: '#ffffff',      // Card background
          sidebar: '#fbfbfc',   // Sidebar background
          border: '#e5e7eb',    // Border color
          text: '#1d1d1d',      // Primary text
          muted: '#6b7280',     // Muted text
          hover: '#f9fafb',     // Hover state
        }
      },
    },
  },
  plugins: [],
}
