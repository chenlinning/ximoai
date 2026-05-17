/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 主色调 - 橄榄绿 #5A8B67
        // 灵感：宣纸、墨、建筑师绘图桌上的标注笔
        primary: {
          50: '#f2f5f3',
          100: '#dfe7e1',
          200: '#c0cfbf',
          300: '#9db29a',
          400: '#7d9579',
          500: '#5A8B67',
          600: '#4a7355',
          700: '#3d5e45',
          800: '#344d3b',
          900: '#2c4033',
          950: '#162019'
        },
        // 辅助色 - 纯中性暖灰，不带任何彩色倾向
        accent: {
          50: '#FAF9F6',
          100: '#F2F0EB',
          200: '#E8E5DF',
          300: '#D4D0C8',
          400: '#B0AAA0',
          500: '#8A8580',
          600: '#6E6A65',
          700: '#55524E',
          800: '#3E3C39',
          900: '#2A2927',
          950: '#1A1A1A'
        },
        // 覆盖 Tailwind 默认 gray 为暖灰色，使所有 gray-* 类自动使用主题配色
        gray: {
          50: '#FAF9F6',
          100: '#F2F0EB',
          200: '#E8E5DF',
          300: '#D4D0C8',
          400: '#B0AAA0',
          500: '#8A8580',
          600: '#6E6A65',
          700: '#55524E',
          800: '#3E3C39',
          900: '#2A2927',
          950: '#1A1A1A'
        },
        // 深色模式 - 纯暖灰黑，不是冷蓝黑
        dark: {
          50: '#FAF9F6',
          100: '#EDEBE8',
          200: '#D8D4CE',
          300: '#B5B0A8',
          400: '#9A9590',
          500: '#7A7570',
          600: '#4A4845',
          700: '#3A3A3A',
          800: '#2C2C2C',
          900: '#232323',
          950: '#171717'
        }
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
        glass: '0 8px 32px rgba(0, 0, 0, 0.06)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.04)',
        card: '0 1px 3px rgba(0, 0, 0, 0.04)',
        'card-hover': '0 4px 16px rgba(0, 0, 0, 0.08)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, #5A8B67 0%, #4a7355 100%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite'
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
        }
      },
      backdropBlur: {
      }
    }
  },
  plugins: []
}
