/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import tailwindcss from '@tailwindcss/vite'
import yamlPlugin from './plugins/vite-plugin-yaml';

// Resolve app meta (title, description) at build time
const htmlMetaPlugin = () => {
  const title = 'Clidey WhoDB';
  const description = 'WhoDB is the next-generation database explorer';
  return {
    name: 'html-meta',
    transformIndexHtml(html: string) {
      return html
        .replace('%VITE_APP_TITLE%', title)
        .replace('%VITE_APP_DESCRIPTION%', description);
    }
  };
};

// https://vitejs.dev/config/
export default defineConfig(async () => {
  // Dynamically import istanbul plugin only in test mode
  let istanbulPlugin = null;
  if (process.env.NODE_ENV === 'test') {
    try {
      const { default: istanbul } = await import('vite-plugin-istanbul');
      // @ts-ignore
      istanbulPlugin = istanbul({
        requireEnv: false,
        include: ['src/**/*.{js,jsx,ts,tsx}'],
        exclude: [
          'node_modules',
          '**/*.d.ts',
          '**/*.test.{js,jsx,ts,tsx}',
          '**/*.spec.{js,jsx,ts,tsx}',
          'src/generated/**',
          'src/index.tsx'
        ],
        cwd: process.cwd(),
      });
    } catch (e) {
      console.warn('Failed to load vite-plugin-istanbul:', e);
    }
  }

  return {
    plugins: [
      yamlPlugin(),
      react(),
      tailwindcss(),
      htmlMetaPlugin(),
      istanbulPlugin
    ].filter(Boolean),

    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
        '@graphql': path.resolve(__dirname, './src/generated/graphql.tsx'),
      },
    },
    server: {
      port: parseInt(process.env.VITE_PORT || '3000'),
      open: process.env.NODE_ENV !== 'test',
      proxy: {
        '/api': {
          target: `http://localhost:${process.env.VITE_BACKEND_PORT || '8080'}`,
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: 'build',
      sourcemap: process.env.NODE_ENV === 'production' ? 'hidden' : true,
      chunkSizeWarningLimit: 1000,
    },
    define: {
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
      '__APP_VERSION__': JSON.stringify(process.env.VITE_APP_VERSION || 'development'),
    },
  };
});
