/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 注意：primary 阶梯色在 src/style.css 的 @theme 中声明为 CSS 变量，
        // 运行时覆写 <html> 上的 --color-primary-* 即可整体换肤（强调色预设）。
        // 此处不再重复定义，否则会编译成静态色值导致换肤失效。
        // 辅助色：电紫
        accent: {
          50: '#f5f3ff',
          100: '#ede9fe',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
          800: '#5b21b6',
          900: '#4c1d95',
          950: '#2e1065'
        },
        // 深色模式：冷调海军蓝
        dark: {
          50: '#f4f7fb',
          100: '#e6edf7',
          200: '#c7d4e6',
          300: '#a8b8d0',
          400: '#6b7f9e',
          500: '#4a5f80',
          600: '#1b2740',
          700: '#17233a',
          800: '#0b1220',
          900: '#070b14',
          950: '#04070f'
        },
        // 语义 Token 色（渐进式迁移用——新增类名如 text-token-muted / bg-token-surface）
        token: {
          bg: 'var(--color-bg)',
          surface: 'var(--color-surface)',
          'surface-2': 'var(--color-surface-2)',
          border: 'var(--color-border)',
          text: 'var(--color-text)',
          'text-secondary': 'var(--color-text-secondary)',
          muted: 'var(--color-text-muted)',
          'text-inverse': 'var(--color-text-inverse)',
          primary: 'rgb(var(--color-primary))',
          accent: 'rgb(var(--color-accent))',
          success: 'rgb(var(--color-success))',
          warning: 'rgb(var(--color-warning))',
          danger: 'rgb(var(--color-danger))',
          info: 'rgb(var(--color-info))',
        }
      },
      fontFamily: {
        sans: ['var(--font-sans)'],
        mono: ['var(--font-mono)']
      },
      boxShadow: {
        glass: 'var(--shadow-glass)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: 'var(--shadow-glow)',
        'glow-lg': 'var(--shadow-glow-lg)',
        neon: 'var(--shadow-neon)',
        card: 'var(--shadow-card)',
        'card-hover': 'var(--shadow-card-hover)',
        'inner-glow': 'var(--shadow-inner-glow)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(135deg, rgb(var(--brand-500)) 0%, rgb(var(--brand-600)) 100%)',
        'gradient-neon': 'linear-gradient(135deg, rgb(var(--brand-400)) 0%, #8b5cf6 100%)',
        'gradient-dark': 'linear-gradient(135deg, #0b1220 0%, #070b14 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.08) 0%, rgba(255,255,255,0.02) 100%)',
        'grid-fine':
          'linear-gradient(var(--grid-line) 1px, transparent 1px), linear-gradient(90deg, var(--grid-line) 1px, transparent 1px)',
        'mesh-gradient':
          'radial-gradient(at 38% 12%, rgb(var(--brand-400) / 0.10) 0px, transparent 52%), radial-gradient(at 84% 4%, rgba(124, 58, 237, 0.10) 0px, transparent 50%), radial-gradient(at 0% 60%, rgb(var(--brand-400) / 0.06) 0px, transparent 50%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'float-slow': 'floatSlow 9s ease-in-out infinite',
        shimmer: 'shimmer 2s linear infinite',
        scan: 'scan 7s linear infinite',
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
        floatSlow: {
          '0%, 100%': { transform: 'translate3d(0, 0, 0)' },
          '50%': { transform: 'translate3d(0, -14px, 0)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        scan: {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(1200%)' }
        },
        glow: {
          '0%': { boxShadow: '0 0 18px rgb(var(--brand-400) / 0.18)' },
          '100%': { boxShadow: '0 0 32px rgb(var(--brand-400) / 0.34)' }
        }
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': 'var(--radius-4xl)'
      }
    }
  },
  plugins: []
}
