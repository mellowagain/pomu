import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    watch: {
      usePolling: true
    }
  },
  plugins: [svelte({
    onwarn: (warning, handler) => {
      if (warning.code.startsWith('a11y-'))
        return;

      handler(warning);
    }
  })],
  optimizeDeps: {
    exclude: ['pomu']
  }
})
