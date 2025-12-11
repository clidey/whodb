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

// Type declaration for Navigator.userAgentData (experimental API)
interface NavigatorUAData {
    platform: string;
}

declare global {
    interface Navigator {
        userAgentData?: NavigatorUAData;
    }
}

/**
 * Platform detection utilities.
 * Computed once at module load time for performance.
 */

function detectMacPlatform(): boolean {
    if (typeof navigator === 'undefined') return false;

    // Modern API (Chrome 90+, Edge 90+, Opera 76+)
    if (navigator.userAgentData) {
        return navigator.userAgentData.platform === 'macOS';
    }

    // Fallback using userAgent (widely available)
    // This catches Mac, iPhone, iPad, iPod
    return /Macintosh|Mac OS X|iPhone|iPad|iPod/i.test(navigator.userAgent);
}

/**
 * Whether the current platform is macOS/iOS.
 * Use this to determine which modifier key to display (⌘ vs Ctrl).
 */
export const isMacPlatform = detectMacPlatform();

/**
 * Check if the command/control key is pressed based on platform.
 * On Mac: checks metaKey (⌘)
 * On Windows/Linux: checks ctrlKey
 */
export function isModKeyPressed(event: KeyboardEvent | React.KeyboardEvent): boolean {
    return isMacPlatform ? event.metaKey : event.ctrlKey;
}

/**
 * Map of shortcut key names to their platform-specific display symbols.
 * Used for rendering keyboard shortcuts in UI.
 */
const keyDisplayMap: Record<string, { mac: string; win: string }> = {
    Mod: { mac: "⌘", win: "Ctrl" },
    Alt: { mac: "⌥", win: "Alt" },
    Shift: { mac: "⇧", win: "Shift" },
    Delete: { mac: "⌫", win: "Del" },
    Backspace: { mac: "⌫", win: "Backspace" },
    Enter: { mac: "↵", win: "Enter" },
    Space: { mac: "Space", win: "Space" },
    Escape: { mac: "Esc", win: "Esc" },
    ArrowUp: { mac: "↑", win: "↑" },
    ArrowDown: { mac: "↓", win: "↓" },
    ArrowLeft: { mac: "←", win: "←" },
    ArrowRight: { mac: "→", win: "→" },
    Home: { mac: "Home", win: "Home" },
    End: { mac: "End", win: "End" },
    PageUp: { mac: "PgUp", win: "PgUp" },
    PageDown: { mac: "PgDn", win: "PgDn" },
};

/**
 * Convert a shortcut key name to its platform-specific display symbol.
 */
export function getKeyDisplay(key: string): string {
    const mapping = keyDisplayMap[key];
    if (mapping) {
        return isMacPlatform ? mapping.mac : mapping.win;
    }
    return key;
}

/**
 * Render an array of shortcut keys as a formatted string.
 * On Mac: symbols are joined without separator (⌘⇧E)
 * On Windows/Linux: keys are joined with + (Ctrl+Shift+E)
 */
export function formatShortcut(keys: string[]): string {
    const displayKeys = keys.map(getKeyDisplay);
    return isMacPlatform ? displayKeys.join("") : displayKeys.join("+");
}
