import adapter from '@sveltejs/adapter-static';

export default {
  kit: {
    adapter: adapter({
      pages: 'internal/server/frontend/dist',
      assets: 'internal/server/frontend/dist',
      fallback: 'index.html'
    }),
    files: {
      routes: 'frontend/src/routes',
      lib: 'frontend/src/lib',
      appTemplate: 'frontend/src/app.html'
    }
  }
};
