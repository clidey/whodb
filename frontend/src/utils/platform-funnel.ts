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

/** Endpoint that stages CE connections and returns a short-lived import token. */
export const PLATFORM_IMPORT_ENDPOINT = `${PLATFORM_URL}/api/ce-import`;

/** A key/value advanced field carried alongside a connection. */
export type PlatformImportAdvanced = {
    Key: string;
    Value: string;
};

/**
 * A single connection handed to WhoDB Platform for import. Mirrors the platform
 * `CreateSourceInput`. `password` is included only when the user explicitly
 * consents; otherwise it is omitted and re-entered on the hosted side.
 */
export type PlatformImportConnection = {
    name: string;
    databaseType: string;
    hostname: string;
    port: string;
    username: string;
    password?: string;
    database: string;
    advanced: PlatformImportAdvanced[];
};

/** The staging response returned by the platform import endpoint. */
export type PlatformImportStageResult = {
    token: string;
    expiresAt: string;
};

/**
 * Maps a saved CE connection profile to the platform import shape. Port is
 * pulled out of the profile's advanced values (where connectors store it) and
 * the remaining advanced entries are carried through. The password travels only
 * when `includePassword` is true.
 */
export const buildImportConnection = (profile: LocalLoginProfile, includePassword: boolean): PlatformImportConnection => {
    const advanced = profile.Advanced ?? [];
    const port = advanced.find(value => value.Key === "Port")?.Value ?? "";
    const rest = advanced
        .filter(value => value.Key !== "Port")
        .map(value => ({ Key: value.Key, Value: value.Value }));
    return {
        name: profile.DisplayName ?? profile.Id,
        databaseType: profile.Type,
        hostname: profile.Hostname,
        port,
        username: profile.Username,
        ...(includePassword && profile.Password ? { password: profile.Password } : {}),
        database: profile.Database,
        advanced: rest,
    };
};

/**
 * Stages the selected connections on WhoDB Platform and returns a short-lived
 * token. Sent as a `text/plain` POST so the browser issues a simple request
 * with no CORS preflight; the platform endpoint is public and origin-permissive.
 */
export const postConnectionsToPlatform = async (connections: PlatformImportConnection[]): Promise<PlatformImportStageResult> => {
    const response = await fetch(PLATFORM_IMPORT_ENDPOINT, {
        method: "POST",
        headers: { "Content-Type": "text/plain" },
        body: JSON.stringify({ connections }),
    });
    if (!response.ok) {
        throw new Error(`Platform import staging failed with status ${response.status}`);
    }
    return response.json() as Promise<PlatformImportStageResult>;
};

/** Builds the landing URL a CE user opens after staging connections. */
export const buildPlatformImportLandingUrl = (token: string): string => {
    const params = new URLSearchParams({ token });
    return `${PLATFORM_URL}/import?${params.toString()}`;
};
