// tailwind.config.js
module.exports = {
    content: [
    "./internal/web/*.templ",
  ],
    darkMode: 'class',
    plugins: [],
    theme: {
      extend: {
        colors: {
          primary: {
            DEFAULT: '#3b82f6',
            foreground: '#ffffff',
          },
          secondary: {
            DEFAULT: '#f3f4f6',
            foreground: '#1f2937',
          },
          destructive: {
            DEFAULT: '#ef4444',
            foreground: '#ffffff',
          },
          muted: {
            DEFAULT: '#f3f4f6',
            foreground: '#6b7280',
          },
          accent: {
            DEFAULT: '#f3f4f6',
            foreground: '#1f2937',
          },
          border: '#e5e7eb',
        },
      },
    },
  }
