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

import type { LocalLoginProfile } from "@/store/auth";

/** The WhoDB Platform (hosted) base URL. */
export const PLATFORM_URL = "https://app.whodb.com";

/**
 * The non-secret connection identity carried to WhoDB Platform when a CE user
 * chooses to use a connection there. The password is intentionally omitted —
 * secrets must never travel in a URL.
 */
export type PlatformImportPrefill = {
    type: string;
    hostname: string;
    port: string;
    database: string;
    username: string;
    displayName: string;
};

/**
 * Extracts the non-secret prefill fields from a saved CE connection profile.
 * Port is read from the profile's advanced values where connectors store it.
 */
export const buildPlatformImportPrefill = (profile: LocalLoginProfile): PlatformImportPrefill => {
    const port = profile.Advanced?.find(value => value.Key === "Port")?.Value ?? "";
    return {
        type: profile.Type,
        hostname: profile.Hostname,
        port,
        database: profile.Database,
        username: profile.Username,
        displayName: profile.DisplayName ?? profile.Id,
    };
};

/**
 * Builds the deep-link a CE user opens to bring a connection into WhoDB
 * Platform as a managed source. Only non-secret identity fields are encoded;
 * the user re-enters the password on the hosted side.
 */
export const buildPlatformImportUrl = (profile: LocalLoginProfile): string => {
    const prefill = buildPlatformImportPrefill(profile);
    const params = new URLSearchParams({
        source: "ce",
        type: prefill.type,
        hostname: prefill.hostname,
        database: prefill.database,
        username: prefill.username,
        name: prefill.displayName,
    });
    if (prefill.port) {
        params.set("port", prefill.port);
    }
    return `${PLATFORM_URL}/import?${params.toString()}`;
};
