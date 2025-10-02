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

import { isDesktopApp } from '../utils/external-links';

// Wails runtime types
declare global {
  interface Window {
    go: {
      common: {
        App: {
          SaveFile: (data: string, defaultName: string) => Promise<string>;
          SaveBinaryFile: (data: number[], defaultName: string) => Promise<string>;
          SelectDirectory: () => Promise<string>;
          SelectSQLiteDatabase: () => Promise<string>;
          CopyToClipboard: (text: string) => Promise<void>;
          GetFromClipboard: () => Promise<string>;
          MinimizeWindow: () => Promise<void>;
          MaximizeWindow: () => Promise<void>;
          ShowMessageDialog: (title: string, message: string, dialogType: string) => Promise<string>;
          ShowConfirmDialog: (title: string, message: string) => Promise<boolean>;
          OpenURL: (url: string) => Promise<void>;
          ShowAboutDialog: () => Promise<void>;
        };
      };
    };
    runtime: {
      EventsOn: (event: string, callback: (...args: any[]) => void) => void;
      EventsOff: (event: string) => void;
      EventsEmit: (event: string, ...args: any[]) => void;
    };
  }
}

const isDesktop = isDesktopApp();

// File Operations - Desktop only, no browser fallbacks needed
export async function saveFile(data: string, defaultName: string): Promise<string | null> {
  if (!isDesktop || !window.go?.common?.App?.SaveFile) {
    console.warn('Save file not available in browser mode');
    return null;
  }

  try {
    const filepath = await window.go.common.App.SaveFile(data, defaultName);
    return filepath || null;
  } catch (error) {
    console.error('Failed to save file:', error);
    return null;
  }
}

export async function saveBinaryFile(data: Uint8Array, defaultName: string): Promise<string | null> {
  if (!isDesktop) {
    console.warn('Save binary file: Not in desktop mode');
    return null;
  }

  if (!window.go?.common?.App?.SaveBinaryFile) {
    console.error('Save binary file: Wails binding not available');
    return null;
  }

  try {
    // Convert Uint8Array to regular array for Wails
    // Wails doesn't handle typed arrays properly in JSON serialization
    const dataArray = Array.from(data);
    const filepath = await window.go.common.App.SaveBinaryFile(dataArray as any, defaultName);
    return filepath || null;
  } catch (error) {
    console.error('Failed to save binary file:', error);
    return null;
  }
}


export async function selectDirectory(): Promise<string | null> {
  if (!isDesktop || !window.go?.common?.App?.SelectDirectory) {
    console.warn('Select directory not available in browser mode');
    return null;
  }

  try {
    const dir = await window.go.common.App.SelectDirectory();
    return dir || null;
  } catch (error) {
    console.error('Failed to select directory:', error);
    return null;
  }
}

export async function selectSQLiteDatabase(): Promise<string | null> {
  if (!isDesktop || !window.go?.common?.App?.SelectSQLiteDatabase) {
    console.warn('Select SQLite database not available in browser mode');
    return null;
  }

  try {
    const filepath = await window.go.common.App.SelectSQLiteDatabase();
    return filepath || null;
  } catch (error) {
    console.error('Failed to select SQLite database:', error);
    return null;
  }
}

// Clipboard Operations
export async function copyToClipboard(text: string): Promise<boolean> {
  // Try desktop first
  if (isDesktop && window.go?.common?.App?.CopyToClipboard) {
    try {
      await window.go.common.App.CopyToClipboard(text);
      return true;
    } catch (error) {
      console.error('Desktop clipboard failed:', error);
    }
  }

  // Fallback to browser clipboard API (works in both browser and desktop)
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch (error) {
    console.error('Failed to copy to clipboard:', error);
  }

  return false;
}

export async function getFromClipboard(): Promise<string | null> {
  // Try desktop first
  if (isDesktop && window.go?.common?.App?.GetFromClipboard) {
    try {
      const text = await window.go.common.App.GetFromClipboard();
      return text || null;
    } catch (error) {
      console.error('Desktop clipboard failed:', error);
    }
  }

  // Fallback to browser clipboard API
  try {
    if (navigator.clipboard && window.isSecureContext) {
      return await navigator.clipboard.readText();
    }
  } catch (error) {
    console.error('Failed to read from clipboard:', error);
  }

  return null;
}

// Window Management - Desktop only
export async function minimizeWindow(): Promise<void> {
  if (!isDesktop || !window.go?.common?.App?.MinimizeWindow) {
    return;
  }

  try {
    await window.go.common.App.MinimizeWindow();
  } catch (error) {
    console.error('Failed to minimize window:', error);
  }
}

export async function maximizeWindow(): Promise<void> {
  if (!isDesktop || !window.go?.common?.App?.MaximizeWindow) {
    return;
  }

  try {
    await window.go.common.App.MaximizeWindow();
  } catch (error) {
    console.error('Failed to maximize window:', error);
  }
}


// Dialog Operations - Desktop only
export async function showMessageDialog(title: string, message: string, type: 'info' | 'warning' | 'error' | 'question' = 'info'): Promise<string | null> {
  if (!isDesktop || !window.go?.common?.App?.ShowMessageDialog) {
    // Browser fallback - use alert
    alert(`${title}\n\n${message}`);
    return 'ok';
  }

  try {
    const result = await window.go.common.App.ShowMessageDialog(title, message, type);
    return result;
  } catch (error) {
    console.error('Failed to show message dialog:', error);
    return null;
  }
}

export async function showConfirmDialog(title: string, message: string): Promise<boolean> {
  if (!isDesktop || !window.go?.common?.App?.ShowConfirmDialog) {
    // Browser fallback - use confirm
    return confirm(`${title}\n\n${message}`);
  }

  try {
    return await window.go.common.App.ShowConfirmDialog(title, message);
  } catch (error) {
    console.error('Failed to show confirm dialog:', error);
    return false;
  }
}

// Event Handling - Desktop only
export function onEvent(event: string, callback: (...args: any[]) => void): void {
  if (!isDesktop || !window.runtime?.EventsOn) {
    return;
  }

  window.runtime.EventsOn(event, callback);
}

export function offEvent(event: string): void {
  if (!isDesktop || !window.runtime?.EventsOff) {
    return;
  }

  window.runtime.EventsOff(event);
}

export function emitEvent(event: string, ...args: any[]): void {
  if (!isDesktop || !window.runtime?.EventsEmit) {
    return;
  }

  window.runtime.EventsEmit(event, ...args);
}

// Export check for desktop mode
export { isDesktop };