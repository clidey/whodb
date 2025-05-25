import classNames from "classnames";
import { FC, useState, useRef, useEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { Icons } from "./icons";
import { ClassNames } from "./classes";
import { LocalLoginProfile } from "../store/auth";
import { databaseTypeDropdownItems } from "../pages/auth/login";

interface ProfileInfoTooltipProps {
  profile: LocalLoginProfile;
  className?: string;
}

// Profile ID validation: only allow alphanumeric, hyphens, underscores, max 64 chars
function isValidProfileId(profileId: string): boolean {
  return typeof profileId === 'string' && 
         profileId.length > 0 && 
         profileId.length <= 64 && 
         /^[a-zA-Z0-9_-]+$/.test(profileId);
}


function getPortFromAdvanced(profile: LocalLoginProfile): string | null {
  const dbType = profile.Type;
  const defaultPortItem = databaseTypeDropdownItems.find(item => item.id === dbType);
  
  if (!defaultPortItem?.extra?.Port) {
    return null; // No default port found, hide this info
  }
  
  const defaultPort = defaultPortItem.extra.Port;

  if (profile.Advanced) {
    const portObj = profile.Advanced.find(item => item.Key === 'Port');
    return portObj?.Value || defaultPort;
  }

  return defaultPort;
}

function getLastAccessedTime(profileId: string): string | null {
  if (!isValidProfileId(profileId)) {
    return null; // Invalid profile ID, hide this info
  }
  
  try {
    const lastAccessed = localStorage.getItem(`whodb_profile_last_accessed_${profileId}`);
    if (lastAccessed) {
      const date = new Date(lastAccessed);
      if (isNaN(date.getTime())) {
        return null; // Invalid date, hide this info
      }
      const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
      const formattedTimeZone = timeZone.replace(/_/g, ' ').split('/').join(' / ');
      return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} (${formattedTimeZone})`;
    }
  } catch (error) {
    // Silently fail - return null to hide this info
  }
  return null;
}

// Portal container reuse - create once and reuse
let tooltipPortalContainer: HTMLDivElement | null = null;

function getTooltipPortalContainer(): HTMLDivElement {
  if (!tooltipPortalContainer) {
    tooltipPortalContainer = document.createElement('div');
    tooltipPortalContainer.id = 'whodb-tooltip-portal';
    document.body.appendChild(tooltipPortalContainer);
  }
  return tooltipPortalContainer;
}

export const ProfileInfoTooltip: FC<ProfileInfoTooltipProps> = ({ profile, className }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [tooltipPos, setTooltipPos] = useState<{ top: number; left: number } | null>(null);
  const btnRef = useRef<HTMLButtonElement | null>(null);

  const port = getPortFromAdvanced(profile);
  const lastAccessed = getLastAccessedTime(profile.Id);

  // If no information is available, don't render the component
  const hasInfo = port !== null || lastAccessed !== null;
  if (!hasInfo) {
    return null;
  }

  // Show tooltip to the right of the icon
  const showTooltip = useCallback(() => {
    if (btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      setTooltipPos({
        top: rect.top + rect.height / 2,
        left: rect.right + 12, // 12px gap to the right
      });
    }
    setIsVisible(true);
  }, []);

  // Hide tooltip
  const hideTooltip = useCallback(() => {
    setIsVisible(false);
  }, []);

  // Memoized event handlers to prevent recreation
  const handleClickAway = useCallback((event: MouseEvent) => {
    if (
      btnRef.current &&
      !btnRef.current.contains(event.target as Node)
    ) {
      setIsVisible(false);
    }
  }, []);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (event.key === "Escape") setIsVisible(false);
  }, []);

  // Optimized event listeners - only add when visible, use stable handlers
  useEffect(() => {
    if (!isVisible) return;
    
    document.addEventListener("mousedown", handleClickAway);
    document.addEventListener("keydown", handleKeyDown);
    
    return () => {
      document.removeEventListener("mousedown", handleClickAway);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isVisible, handleClickAway, handleKeyDown]);

  // Memoize portal container to prevent recreation
  const portalContainer = useMemo(() => getTooltipPortalContainer(), []);

  const tooltip = isVisible && tooltipPos
    ? createPortal(
        <div
          id={`tooltip-${profile.Id}`}
          role="tooltip"
          className={classNames(
            "fixed z-[9999] px-3 py-2 text-xs font-medium bg-white border border-gray-200 rounded-lg shadow-lg",
            "dark:bg-[#2C2F33] dark:border-white/20 dark:text-gray-200",
            "min-w-[180px]",
            "animate-fade"
          )}
          style={{
            top: tooltipPos.top,
            left: tooltipPos.left,
            transform: "translateY(-50%)",
          }}
        >
          <div className="space-y-1">
            {port !== null && (
              <div className="flex justify-between">
                <span className="text-gray-600 dark:text-gray-400">Port:</span>
                <span className={ClassNames.Text}>{port}</span>
              </div>
            )}
            {lastAccessed !== null && (
              <div className="flex justify-between">
                <span className="text-gray-600 dark:text-gray-400">Last Logged In:&nbsp;</span>
                <span className={ClassNames.Text}>{lastAccessed}</span>
              </div>
            )}
          </div>
          <div
            className="absolute top-1/2 left-0 -translate-x-full -translate-y-1/2"
            style={{}}
          >
            <div className="w-0 h-0 border-t-4 border-b-4 border-r-4 border-t-transparent border-b-transparent border-r-gray-200 dark:border-r-white/20"></div>
          </div>
        </div>,
        portalContainer
      )
    : null;

  return (
    <div className={classNames("relative", className)}>
      <button
        ref={btnRef}
        className="flex items-center justify-center w-4 h-4 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 focus:outline-none focus:ring-1 focus:ring-blue-400 focus:ring-offset-2 focus:ring-offset-gray-900 rounded-full transition-colors"
        onClick={isVisible ? hideTooltip : showTooltip}
        aria-label={`Profile information for ${profile.Id}`}
        aria-describedby={`tooltip-${profile.Id}`}
        tabIndex={0}
        type="button"
      >
        <div className="w-4 h-4">{Icons.Information}</div>
      </button>
      {tooltip}
    </div>
  );
};

// Utility function to update last accessed time with validation
export function updateProfileLastAccessed(profileId: string): void {
  if (!isValidProfileId(profileId)) {
    return; // Silently fail for invalid profile IDs
  }
  
  try {
    localStorage.setItem(`whodb_profile_last_accessed_${profileId}`, new Date().toISOString());
  } catch (error) {
    // Silently fail - localStorage may be full or disabled
  }
}