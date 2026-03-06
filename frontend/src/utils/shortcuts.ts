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

import { getEffectiveIsMac } from "./platform";

/**
 * Centralized keyboard shortcut definitions for the WhoDB frontend.
 *
 * This is the single source of truth for all keyboard shortcuts.
 * The CLI has its own centralized keymap in cli/internal/tui/keymap.go.
 *
 * Some shortcuts also have Wails desktop menu accelerators defined in
 * desktop-common/app.go (Toggle Sidebar, Refresh, Export, Import,
 * Execute Query, Clear Editor, Mock Data). Those accelerators must be
 * updated separately since they live in Go code.
 */

type Modifier = "mod" | "shift" | "alt" | "ctrl";

/** A single shortcut definition. */
export interface ShortcutDef {
    /** The key to match against event.key (case-insensitive). */
    key: string;
    /** Modifier keys required. "mod" means Cmd on Mac, Ctrl on Win/Linux. */
    modifiers: Modifier[];
    /** Display keys for UI rendering (passed to formatShortcut / getKeyDisplay). */
    displayKeys: string[];
}

/** Platform-variant shortcut (Mac uses Ctrl+N, Win uses Alt+N for nav). */
export interface PlatformShortcutDef {
    mac: ShortcutDef;
    win: ShortcutDef;
}

/** Resolve a PlatformShortcutDef to the active platform's ShortcutDef. */
export function resolveShortcut(def: PlatformShortcutDef): ShortcutDef {
    return getEffectiveIsMac() ? def.mac : def.win;
}

function platformNav(num: string): PlatformShortcutDef {
    return {
        mac: { key: num, modifiers: ["ctrl"], displayKeys: ["Ctrl", num] },
        win: { key: num, modifiers: ["alt"], displayKeys: ["Alt", num] },
    };
}

function mod(key: string, display?: string): ShortcutDef {
    return { key, modifiers: ["mod"], displayKeys: ["Mod", display ?? key.toUpperCase()] };
}

function modShift(key: string, display?: string): ShortcutDef {
    return { key, modifiers: ["mod", "shift"], displayKeys: ["Mod", "Shift", display ?? key.toUpperCase()] };
}

function plain(key: string, display?: string): ShortcutDef {
    return { key, modifiers: [], displayKeys: [display ?? key] };
}

function shift(key: string, display?: string): ShortcutDef {
    return { key, modifiers: ["shift"], displayKeys: ["Shift", display ?? key] };
}

/**
 * All keyboard shortcuts used in the WhoDB frontend.
 * Grouped by category for readability; the object itself is flat.
 */
export const SHORTCUTS = {
    // ── Global ───────────────────────────────────────────────
    showShortcuts:   { key: "?", modifiers: [], displayKeys: ["Shift", "?"] } as ShortcutDef,
    commandPalette:  mod("k"),
    closeDialogs:    plain("Escape"),
    toggleSidebar:   mod("b"),

    // ── Navigation (platform-variant) ────────────────────────
    navFirst:  platformNav("1"),
    navSecond: platformNav("2"),
    navThird:  platformNav("3"),
    navFourth: platformNav("4"),

    // ── Table Navigation ─────────────────────────────────────
    moveDown:   plain("ArrowDown"),
    moveUp:     plain("ArrowUp"),
    moveFirst:  plain("Home"),
    moveLast:   plain("End"),
    pageDown:   plain("PageDown"),
    pageUp:     plain("PageUp"),
    nextPage:   mod("ArrowRight"),
    prevPage:   mod("ArrowLeft"),

    // ── Selection ────────────────────────────────────────────
    toggleSelect:     plain(" ", "Space"),
    extendSelectDown: shift("ArrowDown"),
    extendSelectUp:   shift("ArrowUp"),
    selectAll:        mod("a"),

    // ── Table Actions ────────────────────────────────────────
    editRow:       plain("Enter"),
    deleteRow:     mod("Delete"),
    deleteRowAlt:  mod("Backspace"),
    editRowAlt:    mod("e"),
    mockData:      modShift("g"),
    refresh:       mod("r"),
    exportData:    modShift("e"),
    importData:    mod("i"),

    // ── Editor ───────────────────────────────────────────────
    executeQuery: mod("Enter"),
    clearEditor:  mod("u"),
} as const;

/**
 * Check whether a keyboard event matches a shortcut definition.
 *
 * Handles platform-specific "mod" resolution (Cmd on Mac, Ctrl on Win/Linux).
 * For PlatformShortcutDef, call resolveShortcut() first.
 */
export function matchesShortcut(
    event: KeyboardEvent | React.KeyboardEvent,
    def: ShortcutDef,
): boolean {
    const isMac = getEffectiveIsMac();

    // Check key match (case-insensitive)
    if (event.key.toLowerCase() !== def.key.toLowerCase()) return false;

    // Determine required modifier state
    const needsMod   = def.modifiers.includes("mod");
    const needsShift = def.modifiers.includes("shift");
    const needsAlt   = def.modifiers.includes("alt");
    const needsCtrl  = def.modifiers.includes("ctrl");

    // "mod" means metaKey on Mac, ctrlKey on Windows/Linux
    const modPressed = isMac ? event.metaKey : event.ctrlKey;

    // Check required modifiers are pressed
    if (needsMod && !modPressed) return false;
    if (needsShift && !event.shiftKey) return false;
    if (needsAlt && !event.altKey) return false;
    if (needsCtrl && !event.ctrlKey) return false;

    // Check that extra modifiers are NOT pressed.
    // "mod" consumes metaKey on Mac or ctrlKey on Win — we only check the
    // remaining modifier keys that weren't explicitly required.
    if (!needsMod && modPressed) return false;
    if (!needsShift && event.shiftKey) {
        // Exception: showShortcuts uses "?" which requires Shift on most layouts,
        // but we define it without "shift" modifier since event.key already is "?".
        // Similarly, letters can arrive uppercased when Shift is pressed for
        // Shift combos — those are handled above. For plain / mod-only shortcuts
        // we want to reject stray Shift to avoid false positives.
        // However, the "?" shortcut needs special handling — its key IS "?",
        // which browsers only emit when Shift is held. We skip the Shift check
        // for that specific case.
        if (def.key !== "?") return false;
    }
    if (!needsAlt && event.altKey) return false;
    // Reject stray ctrlKey when neither "ctrl" nor "mod" (on non-Mac) requires it.
    // On Mac, ctrlKey is independent of metaKey (which handles "mod"), so we also
    // reject stray ctrlKey when "ctrl" wasn't explicitly required.
    if (!needsCtrl && event.ctrlKey && !needsMod) return false;

    return true;
}
