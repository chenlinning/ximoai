/** @type {import('tailwindcss').Config} */
const shadeScale = (name) => ({
  50: `rgb(var(--color-${name}-50) / <alpha-value>)`,
  100: `rgb(var(--color-${name}-100) / <alpha-value>)`,
  200: `rgb(var(--color-${name}-200) / <alpha-value>)`,
  300: `rgb(var(--color-${name}-300) / <alpha-value>)`,
  400: `rgb(var(--color-${name}-400) / <alpha-value>)`,
  500: `rgb(var(--color-${name}-500) / <alpha-value>)`,
  600: `rgb(var(--color-${name}-600) / <alpha-value>)`,
  700: `rgb(var(--color-${name}-700) / <alpha-value>)`,
  800: `rgb(var(--color-${name}-800) / <alpha-value>)`,
  900: `rgb(var(--color-${name}-900) / <alpha-value>)`,
  950: `rgb(var(--color-${name}-950) / <alpha-value>)`
})

export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        inherit: 'inherit',
        current: 'currentColor',
        transparent: 'transparent',
        black: 'rgb(var(--color-black) / <alpha-value>)',
        white: 'rgb(var(--color-white) / <alpha-value>)',
        slate: shadeScale('slate'),
        gray: shadeScale('gray'),
        red: shadeScale('red'),
        orange: shadeScale('orange'),
        amber: shadeScale('amber'),
        yellow: shadeScale('yellow'),
        lime: shadeScale('lime'),
        green: shadeScale('green'),
        emerald: shadeScale('emerald'),
        teal: shadeScale('teal'),
        cyan: shadeScale('cyan'),
        sky: shadeScale('sky'),
        blue: shadeScale('blue'),
        indigo: shadeScale('indigo'),
        violet: shadeScale('violet'),
        purple: shadeScale('purple'),
        pink: shadeScale('pink'),
        rose: shadeScale('rose'),
        primary: shadeScale('primary'),
        accent: shadeScale('accent'),
        dark: shadeScale('dark'),
        stripe: shadeScale('stripe'),
        airwallex: shadeScale('airwallex'),
        alipay: shadeScale('alipay'),
        wxpay: shadeScale('wxpay'),
        easypay: shadeScale('easypay')
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 8px 32px rgb(var(--color-shadow) / 0.06)',
        'glass-sm': '0 4px 16px rgb(var(--color-shadow) / 0.04)',
        glow: '0 0 20px rgb(var(--color-primary-500) / 0.25)',
        'glow-lg': '0 0 40px rgb(var(--color-primary-500) / 0.35)',
        card: '0 1px 3px rgb(var(--color-shadow) / 0.04)',
        'card-hover': '0 4px 16px rgb(var(--color-shadow) / 0.08)',
        'inner-glow': 'inset 0 1px 0 rgb(var(--color-highlight) / 0.1)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary':
          'linear-gradient(135deg, rgb(var(--color-primary-500)) 0%, rgb(var(--color-primary-600)) 100%)',
        'gradient-dark':
          'linear-gradient(135deg, rgb(var(--color-dark-800)) 0%, rgb(var(--color-dark-900)) 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgb(var(--color-highlight) / 0.1) 0%, rgb(var(--color-highlight) / 0.05) 100%)',
        'mesh-gradient':
          'radial-gradient(at 40% 20%, rgb(var(--color-primary-500) / 0.08) 0px, transparent 50%), radial-gradient(at 80% 0%, rgb(var(--color-accent-300) / 0.1) 0px, transparent 50%), radial-gradient(at 0% 50%, rgb(var(--color-primary-500) / 0.05) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'glow 2s ease-in-out infinite alternate'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {
          '0%': { boxShadow: '0 0 20px rgb(var(--color-primary-500) / 0.25)' },
          '100%': { boxShadow: '0 0 30px rgb(var(--color-primary-500) / 0.4)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
