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

#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const VERSION = process.env.WHODB_VERSION || 'latest';
const REPO = 'clidey/whodb';

// Map Node.js platform/arch to Go platform/arch
const PLATFORM_MAP = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64',
};

async function getLatestVersion() {
  return new Promise((resolve, reject) => {
    https.get(
      `https://api.github.com/repos/${REPO}/releases/latest`,
      { headers: { 'User-Agent': 'whodb-mcp-installer' } },
      (res) => {
        let data = '';
        res.on('data', (chunk) => (data += chunk));
        res.on('end', () => {
          try {
            const release = JSON.parse(data);
            resolve(release.tag_name || 'v1.0.0');
          } catch (e) {
            reject(e);
          }
        });
      }
    ).on('error', reject);
  });
}

async function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    const follow = (url) => {
      https.get(url, { headers: { 'User-Agent': 'whodb-mcp-installer' } }, (res) => {
        if (res.statusCode === 302 || res.statusCode === 301) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: ${res.statusCode}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on('finish', () => {
          file.close();
          resolve();
        });
      }).on('error', reject);
    };
    follow(url);
  });
}

async function main() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];

  if (!platform || !arch) {
    console.error(`Unsupported platform: ${process.platform}/${process.arch}`);
    process.exit(1);
  }

  const binDir = path.join(__dirname, 'bin');
  if (!fs.existsSync(binDir)) {
    fs.mkdirSync(binDir, { recursive: true });
  }

  const version = VERSION === 'latest' ? await getLatestVersion() : VERSION;
  const ext = platform === 'windows' ? '.exe' : '';
  const binaryName = `whodb-cli-${platform}-${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/${version}/${binaryName}`;

  const destName = platform === 'windows' ? 'whodb-mcp.exe' : 'whodb-mcp';
  const dest = path.join(binDir, destName);

  console.log(`Downloading WhoDB CLI ${version} for ${platform}/${arch}...`);

  try {
    await downloadFile(url, dest);

    // Make executable on Unix
    if (platform !== 'windows') {
      fs.chmodSync(dest, 0o755);
    }

    console.log('WhoDB MCP server installed successfully!');
  } catch (err) {
    console.error('Download failed:', err.message);
    console.error(`\nYou can manually download from: https://github.com/${REPO}/releases`);
    process.exit(1);
  }
}

main();
