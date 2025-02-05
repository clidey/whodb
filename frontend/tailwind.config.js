// Licensed to Clidey Limited under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Clidey Limited licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

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

