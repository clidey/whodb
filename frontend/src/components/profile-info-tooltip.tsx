import classNames from "classnames";
import { FC, useState, useRef, useEffect, useCallback, useMemo } from "react";
import { createPortal } from "react-dom";
import { LocalLoginProfile } from "../store/auth";
import { databaseTypeDropdownItems } from "../config/database-types";
import { InformationCircleIcon } from "./heroicons";

interface ProfileInfoTooltipProps {
  profile: LocalLoginProfile;
  className?: string;
}

const PROFILE_ID_REGEX = /^[a-zA-Z0-9_\- ]+$/;
const PROFILE_ID_MAX_LENGTH = 64;
const TOOLTIP_OFFSET = 12;

const isValidProfileId = (profileId: string): boolean => 
  typeof profileId === 'string' && 
  profileId.length > 0 && 
  profileId.length <= PROFILE_ID_MAX_LENGTH && 
  PROFILE_ID_REGEX.test(profileId);

const getPortFromAdvanced = (profile: LocalLoginProfile): string | null => {
  const defaultPortItem = databaseTypeDropdownItems.find(item => item.id === profile.Type);
  const defaultPort = defaultPortItem?.extra?.Port;
  
  if (!defaultPort) return null;
  
  if (profile.Advanced) {
    const portObj = profile.Advanced.find(item => item.Key === 'Port');
    return portObj?.Value || defaultPort;
  }

  return defaultPort;
};

const getLastAccessedTime = (profileId: string): string | null => {
  if (!isValidProfileId(profileId)) return null;
  
  try {
    const lastAccessed = localStorage.getItem(`whodb_profile_last_accessed_${profileId}`);
    if (!lastAccessed) return null;

    const date = new Date(lastAccessed);
    if (isNaN(date.getTime())) return null;

    const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    const formattedTimeZone = timeZone.replace(/_/g, ' ').split('/').join(' / ');
    return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })} (${formattedTimeZone})`;
  } catch {
    return null;
  }
};

let tooltipPortalContainer: HTMLDivElement | null = null;

const getTooltipPortalContainer = (): HTMLDivElement => {
  if (!tooltipPortalContainer) {
    tooltipPortalContainer = document.createElement('div');
    tooltipPortalContainer.id = 'whodb-tooltip-portal';
    document.body.appendChild(tooltipPortalContainer);
  }
  return tooltipPortalContainer;
};

const TOOLTIP_CLASSES = {
  container: classNames(
    "fixed z-[9999] px-3 py-2 text-xs font-medium bg-white border border-gray-200 rounded-lg shadow-lg",
    "dark:bg-[#2C2F33] dark:border-white/20 dark:text-gray-200",
    "min-w-[180px]",
    "animate-fade"
  ),
  button: "flex items-center justify-center w-4 h-4 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 focus:outline-none focus:ring-1 focus:ring-blue-400 focus:ring-offset-2 focus:ring-offset-gray-900 rounded-full transition-colors"
};

export const ProfileInfoTooltip: FC<ProfileInfoTooltipProps> = ({ profile, className }) => {
  const [isVisible, setIsVisible] = useState(false);
  const [tooltipPos, setTooltipPos] = useState<{ top: number; left: number } | null>(null);
  const btnRef = useRef<HTMLButtonElement | null>(null);

  const port = getPortFromAdvanced(profile);
  const lastAccessed = getLastAccessedTime(profile.Id);

  if (!port && !lastAccessed) return null;

  const showTooltip = useCallback(() => {
    if (!btnRef.current) return;
    
    const rect = btnRef.current.getBoundingClientRect();
    setTooltipPos({
      top: rect.top + rect.height / 2,
      left: rect.right + TOOLTIP_OFFSET,
    });
    setIsVisible(true);
  }, []);

  const hideTooltip = useCallback(() => setIsVisible(false), []);

  const handleClickAway = useCallback((event: MouseEvent) => {
    if (btnRef.current?.contains(event.target as Node)) return;
    setIsVisible(false);
  }, []);

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (event.key === "Escape") setIsVisible(false);
  }, []);

  useEffect(() => {
    if (!isVisible) return;
    
    document.addEventListener("mousedown", handleClickAway);
    document.addEventListener("keydown", handleKeyDown);
    
    return () => {
      document.removeEventListener("mousedown", handleClickAway);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [isVisible, handleClickAway, handleKeyDown]);

  const portalContainer = useMemo(getTooltipPortalContainer, []);

  const tooltip = isVisible && tooltipPos && createPortal(
    <div
      id={`tooltip-${profile.Id}`}
      role="tooltip"
      className={TOOLTIP_CLASSES.container}
      style={{
        top: tooltipPos.top,
        left: tooltipPos.left,
        transform: "translateY(-50%)",
      }}
    >
      <div className="space-y-1">
        {port && (
          <div className="flex justify-between">
            <span className="text-gray-600 dark:text-gray-400">Port:</span>
            <span>{port}</span>
          </div>
        )}
        {lastAccessed && (
          <div className="flex justify-between">
            <span className="text-gray-600 dark:text-gray-400">Last Logged In:&nbsp;</span>
            <span>{lastAccessed}</span>
          </div>
        )}
      </div>
      <div className="absolute top-1/2 left-0 -translate-x-full -translate-y-1/2">
        <div className="w-0 h-0 border-t-4 border-b-4 border-r-4 border-t-transparent border-b-transparent border-r-gray-200 dark:border-r-white/20" />
      </div>
    </div>,
    portalContainer
  );

  return (
    <div className={classNames("relative", className)}>
      <button
        ref={btnRef}
        className={TOOLTIP_CLASSES.button}
        onClick={isVisible ? hideTooltip : showTooltip}
        aria-label={`Profile information for ${profile.Id}`}
        aria-describedby={`tooltip-${profile.Id}`}
        tabIndex={0}
        type="button"
      >
        <div className="w-4 h-4"><InformationCircleIcon className="w-4 h-4" /></div>
      </button>
      {tooltip}
    </div>
  );
};

export const updateProfileLastAccessed = (profileId: string): void => {
  if (!isValidProfileId(profileId)) return;
  
  try {
    localStorage.setItem(`whodb_profile_last_accessed_${profileId}`, new Date().toISOString());
  } catch {
    // Silently fail if localStorage is not available
  }
};