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

/**
 * Advanced-field keys reserved for SSL configuration.
 */
export const SSL_KEYS = {
    MODE: 'SSL Mode',
    CA_CONTENT: 'SSL CA Content',
    CLIENT_CERT_CONTENT: 'SSL Client Cert Content',
    CLIENT_KEY_CONTENT: 'SSL Client Key Content',
    SERVER_NAME: 'SSL Server Name',
} as const;

/**
 * Returns the reserved SSL advanced-field keys as a set for fast lookups.
 *
 * @returns Reserved SSL field keys.
 */
export function getSSLAdvancedKeys(): ReadonlySet<string> {
    return new Set<string>(Object.values(SSL_KEYS));
}
