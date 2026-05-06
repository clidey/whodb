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

import type { SourceTypeItem } from "@/config/source-types";

interface DiscoveredConnectionPrefillSource {
    DatabaseType: string;
    ProviderType?: string;
    Metadata?: Array<{
        Key: string;
        Value: string;
    }>;
}

/** Matches AWS managed database hostnames (RDS, ElastiCache, DocumentDB, Redshift, Neptune, Timestream) */
const AWS_HOSTNAME_PATTERNS = [
    /\.rds\.amazonaws\.com$/i,
    /\.cache\.amazonaws\.com$/i,
    /\.docdb\.amazonaws\.com$/i,
    /\.redshift\.amazonaws\.com$/i,
    /\.neptune\.amazonaws\.com$/i,
    /\.timestream-influxdb\.\w+\.amazonaws\.com$/i,
];

/** Checks if a hostname belongs to an AWS managed database service */
export function isAwsHostname(hostname: string | undefined | null): boolean {
    if (!hostname) return false;
    return AWS_HOSTNAME_PATTERNS.some(pattern => pattern.test(hostname));
}

/** Matches Azure managed database hostnames */
const AZURE_HOSTNAME_PATTERNS = [
    /\.postgres\.database\.azure\.com$/i,
    /\.mysql\.database\.azure\.com$/i,
    /\.redis\.cache\.windows\.net$/i,
    /\.mongo\.cosmos\.azure\.com$/i,
    /\.redisenterprise\.cache\.azure\.net$/i,
    /\.managedcassandra\.cosmos\.azure\.com$/i,
];

/** Checks if a hostname belongs to an Azure managed database service */
export function isAzureHostname(hostname: string | undefined | null): boolean {
    if (!hostname) return false;
    return AZURE_HOSTNAME_PATTERNS.some(pattern => pattern.test(hostname));
}

/** Matches GCP managed database hostnames (Cloud SQL, AlloyDB, Memorystore, Firestore) */
const GCP_HOSTNAME_PATTERNS = [
    /\.cloudsql\.goog$/i,
    /\.alloydb\.goog$/i,
    /\.memorystore\.goog$/i,
    /\.firestore\.goog$/i,
];

/** Checks if a hostname belongs to a GCP managed database service */
export function isGcpHostname(hostname: string | undefined | null): boolean {
    if (!hostname) return false;
    return GCP_HOSTNAME_PATTERNS.some(pattern => pattern.test(hostname));
}

/**
 * Data structure for prefilling the login form from a discovered cloud connection.
 */
export interface ConnectionPrefillData {
    databaseType: string;
    hostname: string;
    database?: string;
    port?: string;
    advanced: Record<string, string>;
}

/** Get metadata value from a discovered connection */
const getMetadataValue = (conn: DiscoveredConnectionPrefillSource, key: string): string | undefined => {
    if (!conn?.Metadata) return undefined;
    return conn.Metadata.find(m => m.Key === key)?.Value;
};

function providerMatches(allowedProviders: string[], providerType: string | undefined): boolean {
    if (allowedProviders.length === 0) {
        return true;
    }
    if (!providerType) {
        return false;
    }

    const normalizedProvider = providerType.toLowerCase();
    return allowedProviders.some(candidate => candidate.toLowerCase() === normalizedProvider);
}

function conditionsMatch(
    conn: DiscoveredConnectionPrefillSource,
    conditions: NonNullable<SourceTypeItem["discoveryPrefill"]>["AdvancedDefaults"][number]["Conditions"]
): boolean {
    return conditions.every(condition => getMetadataValue(conn, condition.Key) === condition.Value);
}

function resolveAdvancedDefaultValue(
    conn: DiscoveredConnectionPrefillSource,
    advancedDefault: NonNullable<SourceTypeItem["discoveryPrefill"]>["AdvancedDefaults"][number]
): string {
    if (advancedDefault.MetadataKey) {
        const metadataValue = getMetadataValue(conn, advancedDefault.MetadataKey);
        if (metadataValue) {
            return metadataValue;
        }
    }

    if (advancedDefault.Value) {
        return advancedDefault.Value;
    }

    return advancedDefault.DefaultValue;
}

function buildAdvancedPrefill(
    conn: DiscoveredConnectionPrefillSource,
    sourceType: SourceTypeItem | undefined
): Record<string, string> {
    const advanced: Record<string, string> = {};
    const port = getMetadataValue(conn, "port");
    if (port) {
        advanced["Port"] = port;
    }

    for (const advancedDefault of sourceType?.discoveryPrefill?.AdvancedDefaults ?? []) {
        if (!providerMatches(advancedDefault.ProviderTypes, conn.ProviderType)) {
            continue;
        }
        if (!conditionsMatch(conn, advancedDefault.Conditions)) {
            continue;
        }

        const value = resolveAdvancedDefaultValue(conn, advancedDefault);
        if (!value) {
            continue;
        }
        advanced[advancedDefault.Key] = value;
    }

    return advanced;
}

/**
 * Build prefill data from any cloud provider's discovered connection.
 * Applies backend-owned discovery-prefill metadata from the source catalog.
 */
export function buildConnectionPrefill(
    conn: DiscoveredConnectionPrefillSource,
    sourceType?: SourceTypeItem
): ConnectionPrefillData {
    const endpoint = getMetadataValue(conn, "endpoint") || "";
    const port = getMetadataValue(conn, "port");
    const database = getMetadataValue(conn, "databaseName") || getMetadataValue(conn, "bucket");

    return {
        databaseType: conn.DatabaseType,
        hostname: endpoint,
        database,
        port,
        advanced: buildAdvancedPrefill(conn, sourceType),
    };
}
