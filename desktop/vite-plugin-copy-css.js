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

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

// Allow disabling via env if needed
const isDisabled = process.env.WHODB_DISABLE_FRONTEND_CSS_COPY === '1';

export function copyCssFromFrontend() {
  return {
    name: 'copy-css-from-frontend',
    resolveId(id) {
      if (id === 'virtual:frontend-build-css') {
        return '\0virtual:frontend-build-css';
      }
    },
    load(id) {
      if (id === '\0virtual:frontend-build-css') {
        const cssFiles = listIndexHtmlCssFiles();
        if (cssFiles.length === 0) {
          return '/* No frontend build CSS found */';
        }
        const imports = cssFiles
          .map((absPath) => `import ${JSON.stringify(absPath)};`)
          .join('\n');
        return imports;
      }
    },
    async buildStart() {
      if (isDisabled) return;
      await copyCssFiles();
    },
    async handleHotUpdate({ file }) {
      if (isDisabled) return;
      // Watch for changes in frontend build directory
      if (file.includes('/frontend/build/assets/') && file.endsWith('.css')) {
        await copyCssFiles();
      }
    }
  };
}

export default copyCssFromFrontend;

async function copyCssFiles() {
  // Project root: desktop dir is this file's directory (ESM-safe)
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const projectRoot = path.resolve(__dirname, '..');
  const frontendBuildDir = path.resolve(projectRoot, 'frontend/build');
  const desktopDistDir = path.resolve(projectRoot, 'desktop/dist');

  if (!fs.existsSync(frontendBuildDir)) {
    console.log('⚠️  Frontend build directory not found at', frontendBuildDir, '- skipping CSS copy');
    return;
  }

  try {
    const cssFiles = listIndexHtmlCssFiles().map((absPath) => {
      // convert to path relative to frontend build dir
      return path.relative(frontendBuildDir, absPath);
    });

    if (cssFiles.length === 0) {
      console.log('⚠️  No index.html stylesheet CSS files found in frontend build');
      return;
    }

    const desktopAssetsDir = path.join(desktopDistDir, 'assets');
    if (!fs.existsSync(desktopAssetsDir)) {
      fs.mkdirSync(desktopAssetsDir, { recursive: true });
    }

    for (const cssFile of cssFiles) {
      const srcPath = path.join(frontendBuildDir, cssFile);
      const destPath = path.join(desktopDistDir, cssFile);

      fs.copyFileSync(srcPath, destPath);
      console.log(`📋 Copied CSS: ${cssFile}`);
    }

    console.log(`✅ Successfully copied ${cssFiles.length} CSS file(s) from frontend to desktop`);
  } catch (error) {
    console.error('❌ Error copying CSS files:', error);
  }
}

function listIndexHtmlCssFiles() {
  const __filename = fileURLToPath(import.meta.url);
  const __dirname = path.dirname(__filename);
  const projectRoot = path.resolve(__dirname, '..');
  const frontendBuildDir = path.resolve(projectRoot, 'frontend/build');
  const indexHtml = path.join(frontendBuildDir, 'index.html');

  if (!fs.existsSync(indexHtml)) {
    return [];
  }

  const html = fs.readFileSync(indexHtml, 'utf8');
  // Very simple href extractor for <link rel="stylesheet" href="/assets/xxx.css">
  const hrefRegex = /<link[^>]*rel=["']stylesheet["'][^>]*href=["']([^"']+\.css)["'][^>]*>/gi;
  const matches = [];
  let m;
  while ((m = hrefRegex.exec(html)) !== null) {
    matches.push(m[1]);
  }

  const absPaths = matches
    .map((href) => href.replace(/^\//, ''))
    .map((rel) => path.resolve(frontendBuildDir, rel));

  return absPaths.filter((p) => fs.existsSync(p));
}
