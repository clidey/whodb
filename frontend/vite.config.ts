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
import fs from 'fs';
import tailwindcss from '@tailwindcss/vite'

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
        const originalId = id.replace('\0virtual:ee-fallback:', '');
        // Return empty CSS for CSS files, minimal JS for other modules
        if (originalId.endsWith('.css')) {
            return '';
        }
      return 'export default {};';
    }
  }
});

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
        include: [
          'src/**/*.{js,jsx,ts,tsx}',
          // Include EE sources when testing EE edition
          ...(process.env.VITE_BUILD_EDITION === 'ee' && eeExists ? [
            '../ee/frontend/src/**/*.{js,jsx,ts,tsx}'
          ] : [])
        ],
        exclude: [
          'node_modules',
          '**/*.d.ts',
          '**/*.test.{js,jsx,ts,tsx}',
          '**/*.spec.{js,jsx,ts,tsx}',
          'src/generated/**',
          '../ee/frontend/src/generated/**',
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
      react(),
      tailwindcss(),
      eeModulePlugin(),
      istanbulPlugin
    ].filter(Boolean),

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
    port: parseInt(process.env.VITE_PORT || '3000'),
    open: process.env.NODE_ENV !== 'test',
    proxy: {
      '/api': {
        target: `http://localhost:${process.env.VITE_BACKEND_PORT || '8080'}`,
        changeOrigin: true,
      },
    },
  },
    publicDir: eeExists ? path.resolve(__dirname, '../ee/frontend/public') : undefined,
  build: {
    outDir: 'build',
    sourcemap: process.env.NODE_ENV === 'production' ? 'hidden' : true,
      // Removed manual chunking to avoid dependency order issues
      // Let Vite handle chunking automatically to prevent React context errors
      chunkSizeWarningLimit: 1000, // Increase warning limit since we're not manually splitting
  },
    define: {
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
      'process.env.BUILD_EDITION': JSON.stringify(process.env.VITE_BUILD_EDITION),
    },
  };
});