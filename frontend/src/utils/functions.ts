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

import sampleSize from "lodash/sampleSize";

/**
 * Formats a number using locale-aware grouping separators (e.g. 1,000,000 in en-US, 10,00,000 in hi-IN).
 * @param value - The number to format
 * @param language - App locale string in underscore format (e.g. "en_US", "hi_IN")
 */
export function formatNumber(value: number, language: string): string {
    return new Intl.NumberFormat(language.replace('_', '-')).format(value);
}

/**
 * Checks if a string can be parsed as a numeric value.
 * @param str - The string to check
 * @returns True if the string represents a valid number
 */
export function isNumeric(str: string) {
    return !isNaN(Number(str));
}

/**
 * Returns n random items from an array.
 * @param array - The source array
 * @param n - Number of items to return (default: 3)
 * @returns Array of n randomly selected items
 * @throws Error if n is greater than the array length
 */
export function chooseRandomItems<T>(array: T[], n: number = 3): T[] {
    if (n > array.length) {
        throw new Error("n cannot be greater than the array length");
    }
    return sampleSize(array, n);
}

/**
 * Storage unit attribute keys whose Value is a raw byte count and should be
 * auto-scaled for display. Plugins emit bytes as a numeric string; the
 * frontend owns presentation.
 */
const BYTE_SIZE_KEYS: ReadonlySet<string> = new Set(["Total Size", "Data Size"]);

/**
 * Formats a byte count as a human-readable string, auto-scaling to B/KB/MB/GB/TB/PB.
 * Uses 1024-based (binary) steps with decimal-suffix labels, which matches the
 * convention users see in most file managers.
 * @param bytes - The byte count to format. Non-finite or negative values are returned as-is.
 */
export function formatBytes(bytes: number): string {
    if (!Number.isFinite(bytes) || bytes < 0) {
        return String(bytes);
    }
    if (bytes < 1024) {
        return `${bytes} B`;
    }
    const units = ["KB", "MB", "GB", "TB", "PB"] as const;
    let value = bytes / 1024;
    let unitIndex = 0;
    while (value >= 1024 && unitIndex < units.length - 1) {
        value /= 1024;
        unitIndex++;
    }
    const precision = value < 10 ? 2 : value < 100 ? 1 : 0;
    return `${value.toFixed(precision)} ${units[unitIndex]}`;
}

/**
 * Formats a storage-unit attribute value for display. Size-keyed attributes
 * (emitted as raw bytes by the backend) are auto-scaled via formatBytes();
 * all other attributes are lowercased to match existing display conventions.
 * Falls through to the raw value if size parsing fails, keeping pre-migration
 * backends forward-compatible.
 */
export function formatAttributeValue(key: string, value: string | null | undefined): string {
    if (value == null) {
        return "";
    }
    if (BYTE_SIZE_KEYS.has(key)) {
        const parsed = Number(value);
        if (Number.isFinite(parsed) && parsed >= 0) {
            return formatBytes(parsed);
        }
        return value;
    }
    return value.toLowerCase();
}
