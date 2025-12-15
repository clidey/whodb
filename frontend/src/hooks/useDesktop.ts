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

import {useCallback, useEffect} from 'react';
import {useLocation, useNavigate} from 'react-router-dom';
import * as desktopService from '../services/desktop';
import {isDesktopApp} from '../utils/external-links';
import {useAppSelector} from '../store/hooks';
import {InternalRoutes} from '../config/routes';

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
      'menu:refresh': safeHandler(() => {
        // Emit refresh event instead of page reload for HashRouter compatibility
        window.dispatchEvent(new CustomEvent('app:refresh-data'));
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
  }, [isDesktop]);
};