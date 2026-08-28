import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import pluginVue from 'eslint-plugin-vue'

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'vite.config.js',
      'vite.config.d.ts',
      '../backend/internal/web/dist/**'
    ]
  },
  {
    files: ['**/*.{ts,mts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.recommended]
  },
  {
    files: ['**/*.vue'],
    extends: [...pluginVue.configs['flat/essential']],
    languageOptions: {
      parserOptions: {
        parser: tseslint.parser
      }
    }
  },
  {
    files: ['**/*.{js,jsx,cjs,mjs,cts,mts,ts,tsx,vue}'],
    plugins: {
      '@typescript-eslint': tseslint.plugin,
      vue: pluginVue,
    },
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node
      },
      ecmaVersion: 'latest',
      sourceType: 'module'
    },
    rules: {
      'no-constant-condition': 'off',
      'no-mixed-spaces-and-tabs': 'off',
      'no-useless-escape': 'off',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' }
      ],
      '@typescript-eslint/ban-types': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      'vue/multi-word-component-names': 'off',
      'vue/no-use-v-if-with-v-for': 'off'
    }
  }
)
