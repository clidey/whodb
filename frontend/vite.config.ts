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
      // Return minimal module code
      return '';
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
        cypress: true,
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
          'cypress',
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
    port: 3000,
    open: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
    publicDir: eeExists ? path.resolve(__dirname, '../ee/frontend/public') : undefined,
  build: {
    outDir: 'build',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: (id) => {
          // Core vendor libraries
          if (id.includes('node_modules')) {
            // React ecosystem
            if (id.includes('react') || id.includes('react-dom') || id.includes('react-router')) {
              return 'react-vendor';
            }

            // Apollo and GraphQL
            if (id.includes('@apollo') || id.includes('graphql')) {
              return 'graphql-vendor';
            }

            // Large visualization libraries
            if (id.includes('reactflow') || id.includes('@dagrejs/dagre')) {
              return 'visualization';
            }

            // Code editor libraries
            if (id.includes('codemirror') || id.includes('@codemirror')) {
              return 'editor';
            }

            // Utility libraries
            if (id.includes('lodash')) {
              return 'utils';
            }

            // Analytics (PostHog)
            if (id.includes('posthog')) {
              return 'analytics';
            }

            // UI libraries
            if (id.includes('@clidey/ux') || id.includes('@radix-ui') || id.includes('framer-motion')) {
              return 'ui-vendor';
            }

            // Everything else in node_modules
            return 'vendor';
          }
        },
        chunkFileNames: (chunkInfo) => {
          const facadeModuleId = chunkInfo.facadeModuleId ? chunkInfo.facadeModuleId.split('/').pop() : '';
          if (chunkInfo.name.includes('vendor') || chunkInfo.name === 'utils' || chunkInfo.name === 'visualization' || chunkInfo.name === 'editor' || chunkInfo.name === 'analytics' || chunkInfo.name === 'ui-vendor') {
            return `assets/[name]-[hash].js`;
          }
          return `assets/[name]-[hash].js`;
        }
      }
    },
    chunkSizeWarningLimit: 600, // Increase warning limit to 600KB since we're splitting properly
  },
    define: {
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
      'process.env.BUILD_EDITION': JSON.stringify(process.env.VITE_BUILD_EDITION),
    },
  };
});