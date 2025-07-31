/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./src/**/*.{js,jsx,ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        'seckill-orange': '#ff4000',
        'seckill-orange-dark': '#e63600',
        'seckill-orange-light': '#ff6633',
      },
      animation: {
        'bounce-slow': 'bounce 2s infinite',
        'pulse-slow': 'pulse 3s infinite',
        'flash': 'flash 1s ease-in-out infinite alternate',
      },
      keyframes: {
        flash: {
          '0%': { opacity: '1' },
          '100%': { opacity: '0.5' },
        }
      },
      boxShadow: {
        'seckill': '0 4px 20px rgba(255, 64, 0, 0.3)',
        'seckill-hover': '0 8px 30px rgba(255, 64, 0, 0.4)',
      }
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
} 