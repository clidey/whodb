import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';
import fs from 'fs';
// @ts-ignore – TS sometimes fails to resolve this package under bundler resolution
import tailwindcss from '@tailwindcss/vite'
import copyCssFromFrontend from './vite-plugin-copy-css'

// Check if EE directory exists
const eeDir = path.resolve(__dirname, '../ee/frontend/src');
const eeExists = fs.existsSync(eeDir);

// Plugin to handle EE modules in CE builds
const eeModulePlugin = () => ({
  name: 'ee-module-fallback',
  resolveId(id: string) {
    if (id.startsWith('@ee/') && process.env.VITE_BUILD_EDITION !== 'ee') {
      // Return a virtual module ID for EE modules in CE builds
      return '\0virtual:ee-fallback:' + id;
    }
  },
  load(id: string) {
    if (id.startsWith('\0virtual:ee-fallback:')) {
      // Return minimal module code for CE builds
      const modulePath = id.replace('\0virtual:ee-fallback:', '');
      
      if (modulePath.includes('charts/line-chart')) {
        return 'export const LineChart = () => null; export default LineChart;';
      }
      if (modulePath.includes('charts/pie-chart')) {
        return 'export const PieChart = () => null; export default PieChart;';
      }
      if (modulePath.includes('pages/raw-execute/index')) {
        return 'export default { ActionOptions: {}, ActionOptionIcons: {}, plugins: [] };';
      }
      if (modulePath.includes('config.tsx')) {
        return 'export const eeDatabaseTypes = []; export const eeFeatures = {}; export default {};';
      }
      if (modulePath.includes('icons')) {
        return 'export const EEIcons = { Logos: {} }; export default EEIcons;';
      }
      if (modulePath.includes('index')) {
        return 'export const isEENoSQLDatabase = () => false; export const getEEDatabaseStorageUnitLabel = () => ""; export const EEDatabaseType = {}; export type EEDatabaseTypeValue = string;';
      }
      if (modulePath.includes('index.css')) {
        return '/* EE styles not available in CE */';
      }
      
      return 'export default {};';
    }
  }
});

// https://vitejs.dev/config/
export default defineConfig(async () => {
  return {
    plugins: [
      react(),
      tailwindcss(),
      copyCssFromFrontend(),
      eeModulePlugin(),
    ],

    resolve: {
      alias: {
        // Reference frontend directly instead of copying
        '@': path.resolve(__dirname, '../frontend/src'),
        ...(eeExists ? { '@ee': eeDir } : {}),
        // Dynamic GraphQL import based on build edition
        '@graphql': process.env.VITE_BUILD_EDITION === 'ee' 
          ? path.resolve(__dirname, '../ee/frontend/src/generated/graphql.tsx')
          : path.resolve(__dirname, '../frontend/src/generated/graphql.tsx'),
        // Handle relative imports from EE to frontend
        '../../../../../frontend/src': path.resolve(__dirname, '../frontend/src'),
        '../../../frontend/src': path.resolve(__dirname, '../frontend/src'),
      },
      // Ensure singletons for React and React Router across desktop and frontend to prevent context mismatches
      dedupe: ['react', 'react-dom', 'react-router', 'react-router-dom'],
    },

    // Configure public directory to resolve assets from frontend
    publicDir: path.resolve(__dirname, '../frontend/public'),

    // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
    //
    // 1. prevent vite from obscuring rust errors
    clearScreen: false,
    // 2. tauri expects a fixed port, fail if that port is not available
    server: {
      port: 1420,
      strictPort: true,
      watch: {
        // 3. tell vite to ignore watching `src-tauri`
        ignored: ["**/src-tauri/**"],
      },
      fs: {
        // allow importing CSS assets from the frontend build directory
        allow: [
          path.resolve(__dirname, '..'),
          path.resolve(__dirname, '../frontend/build')
        ],
      },
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },

    build: {
      outDir: 'dist',
      sourcemap: false,
    },

    define: {
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
      'process.env.BUILD_EDITION': JSON.stringify(process.env.VITE_BUILD_EDITION),
    },
  };
});
