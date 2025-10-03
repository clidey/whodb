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

import { useCallback, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import * as desktopService from '../services/desktop';
import { isDesktopApp } from '../utils/external-links';
import { useAppSelector } from '../store/hooks';
import { InternalRoutes } from '../config/routes';

// Hook for file operations
export const useDesktopFile = () => {
  const isDesktop = isDesktopApp();

  const saveFile = useCallback(async (data: string, defaultName: string) => {
    return await desktopService.saveFile(data, defaultName);
  }, []);

  const saveBinaryFile = useCallback(async (data: Uint8Array, defaultName: string) => {
    return await desktopService.saveBinaryFile(data, defaultName);
  }, []);

  const selectDirectory = useCallback(async () => {
    return await desktopService.selectDirectory();
  }, []);

  const selectSQLiteDatabase = useCallback(async () => {
    return await desktopService.selectSQLiteDatabase();
  }, []);

  return {
    isDesktop,
    saveFile,
    saveBinaryFile,
    selectDirectory,
    selectSQLiteDatabase,
  };
};

// Hook for clipboard operations
export const useDesktopClipboard = () => {
  const isDesktop = isDesktopApp();

  const copyToClipboard = useCallback(async (text: string) => {
    return await desktopService.copyToClipboard(text);
  }, []);

  const getFromClipboard = useCallback(async () => {
    return await desktopService.getFromClipboard();
  }, []);

  return {
    isDesktop,
    copyToClipboard,
    getFromClipboard,
  };
};

// Hook for window management
export const useDesktopWindow = () => {
  const isDesktop = isDesktopApp();

  const minimizeWindow = useCallback(async () => {
    await desktopService.minimizeWindow();
  }, []);

  const maximizeWindow = useCallback(async () => {
    await desktopService.maximizeWindow();
  }, []);

  return {
    isDesktop,
    minimizeWindow,
    maximizeWindow,
  };
};

// Hook for dialog operations
export const useDesktopDialog = () => {
  const isDesktop = isDesktopApp();

  const showMessage = useCallback(async (title: string, message: string, type: 'info' | 'warning' | 'error' = 'info') => {
    return await desktopService.showMessageDialog(title, message, type);
  }, []);

  const showConfirm = useCallback(async (title: string, message: string) => {
    return await desktopService.showConfirmDialog(title, message);
  }, []);

  return {
    isDesktop,
    showMessage,
    showConfirm,
  };
};

// Hook for menu events
export const useDesktopMenu = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const isDesktop = isDesktopApp();
  const { showConfirm } = useDesktopDialog();
  const currentAuth = useAppSelector(state => state.auth.current);

  useEffect(() => {
    if (!isDesktop) return;

    // Prevent zooming with keyboard shortcuts
    const handleKeydown = (e: KeyboardEvent) => {
      // Prevent Cmd/Ctrl + Plus/Minus/0 for zoom
      if ((e.metaKey || e.ctrlKey) && (e.key === '+' || e.key === '-' || e.key === '=' || e.key === '0')) {
        e.preventDefault();
        return false;
      }
    };

    // Prevent zooming with mouse wheel
    const handleWheel = (e: WheelEvent) => {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        return false;
      }
    };

    // Add event listeners to prevent zooming
    document.addEventListener('keydown', handleKeydown, true);
    document.addEventListener('wheel', handleWheel, { passive: false });

    // Setup menu event listeners with error handling
    const safeHandler = (handler: () => void | Promise<void>) => {
      return async () => {
        try {
          await handler();
        } catch (error) {
          console.error('Menu handler error:', error);
        }
      };
    };

    const handlers = {
      'menu:toggle-sidebar-new-connection': safeHandler(() => {
        if (currentAuth) {
          // User is logged in - emit event to open sidebar with add profile form
          window.dispatchEvent(new CustomEvent('menu:open-add-profile'));
        } else {
          // User is not logged in - navigate to login page
          navigate('/login');
        }
      }),
      'menu:export-data': safeHandler(() => {
        // Emit custom event that the table component can listen to
        window.dispatchEvent(new CustomEvent('menu:trigger-export'));
      }),
      'menu:copy': safeHandler(async () => {
        // Copy selected text to clipboard
        const selection = window.getSelection();
        if (selection && selection.toString()) {
          try {
            await navigator.clipboard.writeText(selection.toString());
          } catch (err) {
            console.error('Failed to copy:', err);
          }
        }
      }),
      'menu:paste': safeHandler(async () => {
        try {
          // Get text from clipboard
          const text = await desktopService.getFromClipboard();
          if (!text) return;

          // Get the currently focused element
          const activeElement = document.activeElement;

          // Check if it's an input or textarea element
          if (activeElement && (activeElement instanceof HTMLInputElement || activeElement instanceof HTMLTextAreaElement)) {
            const input = activeElement as HTMLInputElement | HTMLTextAreaElement;
            const start = input.selectionStart || 0;
            const end = input.selectionEnd || 0;
            const currentValue = input.value;

            // Insert the text at the cursor position
            const newValue = currentValue.substring(0, start) + text + currentValue.substring(end);

            // Use the native setter to update the value
            const nativeInputValueSetter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value")?.set;
            const nativeTextAreaValueSetter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, "value")?.set;

            if (input instanceof HTMLInputElement && nativeInputValueSetter) {
              nativeInputValueSetter.call(input, newValue);
            } else if (input instanceof HTMLTextAreaElement && nativeTextAreaValueSetter) {
              nativeTextAreaValueSetter.call(input, newValue);
            } else {
              input.value = newValue;
            }

            // Set cursor position after the inserted text
            const newCursorPos = start + text.length;
            input.setSelectionRange(newCursorPos, newCursorPos);

            // Trigger React's onChange by dispatching both input and change events
            const inputEvent = new Event('input', { bubbles: true, cancelable: true });
            const changeEvent = new Event('change', { bubbles: true, cancelable: true });
            input.dispatchEvent(inputEvent);
            input.dispatchEvent(changeEvent);
          } else if (activeElement && activeElement.getAttribute('contenteditable') === 'true') {
            // Handle contenteditable elements (like rich text editors)
            const selection = window.getSelection();
            if (selection && selection.rangeCount > 0) {
              const range = selection.getRangeAt(0);
              range.deleteContents();
              const textNode = document.createTextNode(text);
              range.insertNode(textNode);

              // Move cursor to the end of inserted text
              range.setStartAfter(textNode);
              range.setEndAfter(textNode);
              selection.removeAllRanges();
              selection.addRange(range);

              // Trigger input event for contenteditable
              const event = new InputEvent('input', {
                bubbles: true,
                cancelable: true,
                inputType: 'insertText',
                data: text
              });
              activeElement.dispatchEvent(event);
            }
          }
        } catch (err) {
          console.error('Failed to paste:', err);
        }
      }),
      'menu:select-all': safeHandler(() => {
        // Select all text in the focused element
        const activeElement = document.activeElement;
        if (activeElement && ('select' in activeElement)) {
          (activeElement as HTMLInputElement | HTMLTextAreaElement).select();
        } else {
          // For non-input elements, select all content
          const selection = window.getSelection();
          const range = document.createRange();
          range.selectNodeContents(document.body);
          selection?.removeAllRanges();
          selection?.addRange(range);
        }
      }),
      // Find functionality could be implemented when needed
      // 'menu:find': safeHandler(() => {
      //   window.dispatchEvent(new CustomEvent('desktop:find'));
      // }),
      'menu:refresh': safeHandler(() => {
        // For HashRouter, we should refresh data not reload the page
        // Emit an event that components can listen to for refreshing their data
        window.dispatchEvent(new CustomEvent('app:refresh-data'));
        // Alternatively, if we need to reload the window properly with Wails:
        // const wailsGo = (window as any).go;
        // if (wailsGo?.main?.App?.ReloadWindow) {
        //   wailsGo.main.App.ReloadWindow();
        // }
      }),
      'menu:toggle-sidebar': safeHandler(() => {
        // Use the proper sidebar toggle mechanism - click the trigger button
        const sidebarTrigger = document.querySelector('[data-sidebar-trigger]') as HTMLButtonElement;
        if (sidebarTrigger) {
          sidebarTrigger.click();
        } else {
          // Fallback: emit event that sidebar can listen to
          window.dispatchEvent(new CustomEvent('menu:toggle-sidebar'));
        }
      }),
      'menu:execute-query': safeHandler(() => {
        // Emit custom event that the editor component can listen to
        window.dispatchEvent(new CustomEvent('menu:trigger-execute-query'));
      }),
      'menu:disconnect': safeHandler(async () => {
        const confirm = await showConfirm('Disconnect', 'Are you sure you want to disconnect from the current database?');
        if (confirm) {
          navigate('/');
        }
      }),
      'menu:new-scratchpad-page': safeHandler(() => {
        // Only allow creating new page if already on Scratchpad page
        if (location.pathname === InternalRoutes.RawExecute.path) {
          // Emit event that the Scratchpad page will listen to
          window.dispatchEvent(new CustomEvent('menu:new-scratchpad-page'));
        }
        // If not on Scratchpad page, do nothing (shortcut is context-sensitive)
      }),
      // Window and about operations could be implemented when needed
      // 'menu:toggle-always-on-top': safeHandler(() => {
      //   window.dispatchEvent(new CustomEvent('desktop:toggle-always-on-top'));
      // }),
      // 'menu:about': safeHandler(() => {
      //   window.dispatchEvent(new CustomEvent('desktop:show-about'));
      // }),
    };

    // Register all event handlers
    Object.entries(handlers).forEach(([event, handler]) => {
      desktopService.onEvent(event, handler);
    });

    // Cleanup
    return () => {
      document.removeEventListener('keydown', handleKeydown, true);
      document.removeEventListener('wheel', handleWheel);
      Object.keys(handlers).forEach(event => {
        desktopService.offEvent(event);
      });
    };
  }, [isDesktop, navigate, location.pathname, showConfirm, currentAuth]);
};

// Hook for keyboard shortcuts (in addition to menu shortcuts)
// Currently unused but kept for future keyboard shortcuts implementation
// export const useDesktopKeyboard = () => {
//   const isDesktop = isDesktopApp();
//
//   useEffect(() => {
//     if (!isDesktop) return;
//
//     const handleKeyDown = (e: KeyboardEvent) => {
//       const isMac = navigator.userAgent.toUpperCase().indexOf('MAC') >= 0;
//       const ctrlOrCmd = isMac ? e.metaKey : e.ctrlKey;
//
//       // Additional keyboard shortcuts not in menu
//       if (ctrlOrCmd && e.shiftKey) {
//         switch (e.key.toLowerCase()) {
//           case 'p':
//             e.preventDefault();
//             // Command palette could be implemented when needed
//             break;
//         }
//       }
//     };
//
//     window.addEventListener('keydown', handleKeyDown);
//
//     return () => {
//       window.removeEventListener('keydown', handleKeyDown);
//     };
//   }, [isDesktop]);
// };