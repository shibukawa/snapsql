import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler']],
      },
    }),
    tailwindcss(),
  ],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  // @ts-expect-error - test is from vitest
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: [],
    css: true,
    coverage: {
      reporter: ['text', 'html'],
    },
    typecheck: {
      tsconfig: './tsconfig.vitest.json',
    },
  },
})
