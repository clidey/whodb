import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';

// Check if EE directory exists
const eeDir = path.resolve(__dirname, '../ee/frontend/src');
const eeExists = fs.existsSync(eeDir);

// Plugin to handle missing EE modules
const eeModulePlugin = () => ({
  name: 'ee-module-fallback',
  resolveId(id: string) {
    if (id.startsWith('@ee/') && !eeExists) {
      // Return a virtual module ID for missing EE modules
      return '\0virtual:ee-fallback:' + id;
    }
  },
  load(id: string) {
    if (id.startsWith('\0virtual:ee-fallback:')) {
      // Return minimal module code
      return '';
    }
  }
});

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), eeModulePlugin()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      ...(eeExists ? { '@ee': eeDir } : {}),
      // Dynamic GraphQL import based on build edition
      '@graphql': process.env.VITE_BUILD_EDITION === 'ee' 
        ? path.resolve(__dirname, '../ee/frontend/src/generated/graphql.tsx')
        : path.resolve(__dirname, './src/generated/graphql.tsx'),
      // Handle relative imports from EE to frontend
      '../../../../../frontend/src': path.resolve(__dirname, './src'),
      '../../../frontend/src': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 3000,
    open: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'build',
    sourcemap: true,
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
    'process.env.BUILD_EDITION': JSON.stringify(process.env.VITE_BUILD_EDITION),
  },
});