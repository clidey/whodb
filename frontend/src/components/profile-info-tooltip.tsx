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

import classNames from "classnames";
import { FC, useState, useCallback, useRef, useEffect } from "react";
import { Icons } from "./icons";
import { ClassNames } from "./classes";
import { LocalLoginProfile } from "../store/auth";

interface ProfileInfoTooltipProps {
  profile: LocalLoginProfile;
  className?: string;
}

function extractPortFromHostname(hostname: string): string {
  const parts = hostname.split(':');
  if (parts.length > 1) {
    const port = parts[parts.length - 1];
    // Check if the last part is numeric (a port)
    if (/^\d+$/.test(port)) {
      return port;
    }
  }
  return 'Default';
}

function getLastAccessedTime(profileId: string): string {
  try {
    const lastAccessed = localStorage.getItem(`whodb_profile_last_accessed_${profileId}`);
    if (lastAccessed) {
      const date = new Date(lastAccessed);
      return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
  } catch (error) {
    console.warn('Failed to get last accessed time:', error);
  }
  return 'Never';
}

export const ProfileInfoTooltip: FC<ProfileInfoTooltipProps> = ({ profile, className }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [isHovered, setIsHovered] = useState(false);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  const port = extractPortFromHostname(profile.Hostname);
  const lastAccessed = getLastAccessedTime(profile.Id);

  const showTooltip = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    setIsVisible(true);
  }, []);

  const hideTooltip = useCallback(() => {
    timeoutRef.current = setTimeout(() => {
      if (!isHovered) {
        setIsVisible(false);
      }
    }, 100);
  }, [isHovered]);

  const handleMouseEnter = useCallback(() => {
    setIsHovered(true);
    showTooltip();
  }, [showTooltip]);

  const handleMouseLeave = useCallback(() => {
    setIsHovered(false);
    hideTooltip();
  }, [hideTooltip]);

  const handleFocus = useCallback(() => {
    showTooltip();
  }, [showTooltip]);

  const handleBlur = useCallback(() => {
    setIsVisible(false);
  }, []);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      setIsVisible(!isVisible);
    } else if (e.key === 'Escape') {
      setIsVisible(false);
    }
  }, [isVisible]);

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, []);

  return (
    <div className={classNames("relative inline-block", className)}>
      <button
        className="flex items-center justify-center w-4 h-4 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-1 rounded-full transition-colors"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
        onFocus={handleFocus}
        onBlur={handleBlur}
        onKeyDown={handleKeyDown}
        aria-label={`Profile information for ${profile.Id}`}
        aria-describedby={`tooltip-${profile.Id}`}
        tabIndex={0}
      >
        <div className="w-4 h-4">
          {Icons.Information}
        </div>
      </button>
      
      {isVisible && (
        <div
          id={`tooltip-${profile.Id}`}
          role="tooltip"
          className={classNames(
            "absolute z-50 px-3 py-2 text-xs font-medium bg-white border border-gray-200 rounded-lg shadow-lg",
            "dark:bg-[#2C2F33] dark:border-white/20 dark:text-gray-200",
            "min-w-[180px] right-0 bottom-full mb-2",
            "animate-fade"
          )}
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => setIsHovered(false)}
        >
          <div className="space-y-1">
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Port:</span>
              <span className={ClassNames.Text}>{port}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600 dark:text-gray-400">Last Accessed:</span>
              <span className={ClassNames.Text}>{lastAccessed}</span>
            </div>
          </div>
          {/* Tooltip arrow */}
          <div className="absolute top-full right-4 w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-gray-200 dark:border-t-white/20"></div>
        </div>
      )}
    </div>
  );
};

// Utility function to update last accessed time
export function updateProfileLastAccessed(profileId: string): void {
  try {
    localStorage.setItem(`whodb_profile_last_accessed_${profileId}`, new Date().toISOString());
  } catch (error) {
    console.warn('Failed to update last accessed time:', error);
  }
}