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

import React from 'react';

/**
 * Check if we're running in a desktop environment (Wails)
 */
export const isDesktopApp = (): boolean => {
  // In Wails desktop apps, the window.go object is available
  if (typeof window === 'undefined') {
    return false;
  }

  // For Wails apps, we MUST have the go bindings available
  // If they're not there, we're not really in a desktop app
  // even if other indicators suggest we might be
  return !!(window as any).go?.common?.App;
};

/**
 * Open an external URL in the system's default browser
 * Handles both desktop and web environments
 */
export const openExternalLink = async (url: string, event?: React.MouseEvent): Promise<void> => {
  // Prevent default behavior if event is provided
  if (event) {
    event.preventDefault();
    event.stopPropagation();
  }

  if (isDesktopApp()) {
    // Use Wails runtime to open the URL in the system browser
    const wailsGo = (window as any).go;
    if (wailsGo && wailsGo.main && wailsGo.main.App && wailsGo.main.App.OpenURL) {
      try {
        await wailsGo.main.App.OpenURL(url);
      } catch (error) {
        console.error('Failed to open external link:', error);
        // Fallback to window.open if Wails method fails
        const newWindow = window.open(url, '_blank');
        if (!newWindow) {
          console.warn('Unable to open external link in desktop app:', url);
          alert(`Please open this URL in your browser: ${url}`);
        }
      }
    } else {
      // Fallback if Wails runtime is not available
      const newWindow = window.open(url, '_blank');
      if (!newWindow) {
        console.warn('Unable to open external link in desktop app:', url);
        alert(`Please open this URL in your browser: ${url}`);
      }
    }
  } else {
    // In web browser, use standard behavior
    window.open(url, '_blank', 'noopener,noreferrer');
  }
};

/**
 * React component wrapper for external links
 * Usage: <ExternalLink href="https://example.com">Link Text</ExternalLink>
 */
interface ExternalLinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  href: string;
  children: React.ReactNode;
}

export const ExternalLink: React.FC<ExternalLinkProps> = ({
  href,
  children,
  onClick,
  ...props
}) => {
  const handleClick = (e: React.MouseEvent<HTMLAnchorElement>) => {
    openExternalLink(href, e);
    if (onClick) {
      onClick(e);
    }
  };

  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      onClick={handleClick}
      {...props}
    >
      {children}
    </a>
  );
};