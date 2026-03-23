/*
 * Copyright 2026 Clidey, Inc.
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

import {isDesktopApp} from '../utils/external-links';

const isDesktop = isDesktopApp();

/**
 * Copies text to the clipboard. Works in all contexts:
 * 1. Desktop (Wails binding)
 * 2. Secure browser context (Clipboard API)
 * 3. Insecure browser context (execCommand fallback)
 */
export async function copyToClipboard(text: string): Promise<boolean> {
  if (isDesktop && window.go?.common?.App?.CopyToClipboard) {
    try {
      await window.go.common.App.CopyToClipboard(text);
      return true;
    } catch (error) {
      console.error('Desktop clipboard failed:', error);
    }
  }

  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch (error) {
    console.error('Failed to copy to clipboard:', error);
  }

  try {
    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.left = '-9999px';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const success = document.execCommand('copy');
    document.body.removeChild(textarea);
    return success;
  } catch (error) {
    console.error('execCommand copy fallback failed:', error);
  }

  return false;
}

