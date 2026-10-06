import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  server: {
    proxy: Object.fromEntries(
      ['/api', '/oidc', '/ingest'].map((path) => [
        path,
        process.env.WARNLY_API_URL || 'http://127.0.0.1:8080'
      ])
    )
  }
});
