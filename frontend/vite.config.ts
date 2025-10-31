import { defineConfig } from 'vite'
import viteReact from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

import { resolve } from 'node:path'
// removed cloudflare vite plugin

// https://vitejs.dev/config/
const config: any = {
  plugins: [viteReact(), tailwindcss()],
  test: {
    globals: true,
    environment: 'jsdom',
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  build: {
    minify: 'terser',
    sourcemap: true,
  },
  server: {
    port: 3000,
  },
}

export default defineConfig(config)
