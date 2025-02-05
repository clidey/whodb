/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  darkMode: 'selector',
  theme: {
    extend: {
      fontFamily: {
        title: `"Megrim", -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
        'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
        sans-serif;`,
      },
      colors: {
        brand: "#BF853E",
      },
      animation: {
        fade: 'fadeOut 300ms ease-in-out',
        boxy: 'boxy 1.5s infinite',
      },
      keyframes: () => ({
        fadeOut: {
          '0%': {
            opacity: 0,
          },
          '100%': {
            opacity: 100,
          },
        },
        boxy: {
          '0%': { backgroundSize: '35px 15px,15px 15px,15px 35px,35px 35px' },
          '25%': { backgroundSize: '35px 35px,15px 35px,15px 15px,35px 15px' },
          '50%': { backgroundSize: '15px 35px,35px 35px,35px 15px,15px 15px' },
          '75%': { backgroundSize: '15px 15px,35px 15px,35px 35px,15px 35px' },
          '100%': { backgroundSize: '35px 15px,15px 15px,15px 35px,35px 35px' },
        },
      }),
    },
  },
  plugins: [],
}

