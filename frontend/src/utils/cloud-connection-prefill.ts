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

import { LocalDiscoveredConnection } from "@/store/providers";

/**
 * Data structure for prefilling the login form from a discovered cloud connection.
 */
export interface ConnectionPrefillData {
    databaseType: string;
    hostname: string;
    port?: string;
    advanced: Record<string, string>;
}

/**
 * Prefill rule function signature.
 * Receives connection and metadata getter, returns advanced settings to apply.
 */
export type PrefillRule = (
    conn: LocalDiscoveredConnection,
    meta: (key: string) => string | undefined
) => Record<string, string>;

/** Get metadata value from a discovered connection */
const getMetadataValue = (conn: LocalDiscoveredConnection, key: string): string | undefined => {
    if (!conn?.Metadata) return undefined;
    return conn.Metadata.find(m => m.Key === key)?.Value;
};

/**
 * CE prefill rules for base database types.
 * Each rule defines how to configure advanced settings for cloud-managed instances.
 */
const basePrefillRules: Record<string, PrefillRule> = {
    // SQL databases - managed cloud services require SSL
    Postgres: () => ({ "SSL Mode": "require" }),
    MySQL: () => ({ "SSL Mode": "require" }),
    MariaDB: () => ({ "SSL Mode": "require" }),

    // ElastiCache (Redis-compatible)
    ElastiCache: (_, meta): Record<string, string> => {
        if (meta("transitEncryption") === "true") {
            return { "TLS": "true" };
        }
        return {};
    },

    // DocumentDB (MongoDB-compatible)
    DocumentDB: () => ({
        "URL Params": "?tls=true&tlsInsecure=true&replicaSet=rs0&retryWrites=false&readPreference=secondaryPreferred"
    }),
};

// Combined rules - EE rules will be merged at runtime
let prefillRules: Record<string, PrefillRule> = { ...basePrefillRules };

// Load EE prefill rules if in EE mode
if (import.meta.env.VITE_BUILD_EDITION === 'ee') {
    import('@ee/utils/cloud-prefill-rules')
        .then((eeModule) => {
            if (eeModule?.eePrefillRules) {
                prefillRules = { ...basePrefillRules, ...eeModule.eePrefillRules };
            }
        })
        .catch((error) => {
            console.error('Could not load EE prefill rules:', error);
        });
}

/**
 * Build prefill data from any cloud provider's discovered connection.
 * Applies database-specific rules to configure SSL/TLS and other advanced settings.
 */
export function buildConnectionPrefill(conn: LocalDiscoveredConnection): ConnectionPrefillData {
    const meta = (key: string) => getMetadataValue(conn, key);
    const endpoint = meta("endpoint") || "";
    const port = meta("port");

    const advanced: Record<string, string> = {};
    if (port) {
        advanced["Port"] = port;
    }

    // Apply database-specific rules (CE or EE)
    const rule = prefillRules[conn.DatabaseType];
    if (rule) {
        Object.assign(advanced, rule(conn, meta));
    }

    return {
        databaseType: conn.DatabaseType,
        hostname: endpoint,
        port,
        advanced,
    };
}
