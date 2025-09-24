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

/**
 * Credential Manager for WhoDB Desktop
 *
 * Uses Tauri Stronghold for secure storage in desktop app,
 * falls back to sessionStorage for web builds
 */

interface StoredCredentials {
    credentials: any;
    timestamp: number;
}

class CredentialManager {
    private isDesktop: boolean;
    private strongholdClient: any = null;
    private stronghold: any = null;
    private strongholdStore: any = null;
    private masterPasswordSet: boolean = false;
    private isInitialized: boolean = false;
    private cachedCredentials: any = null;
    private cachedHeader: string | null = null;

    constructor() {
        // Check if running in Tauri desktop environment
        this.isDesktop = !!(window as any).__TAURI__;
        console.log('[CredentialManager] Running in:', this.isDesktop ? 'Desktop' : 'Web');
    }

    /**
     * Initialize Stronghold (desktop only)
     * @param masterPassword - Master password for Stronghold vault
     */
    async initialize(masterPassword?: string): Promise<void> {
        if (!this.isDesktop) {
            console.log('[CredentialManager] Web mode - skipping Stronghold initialization');
            return;
        }

        // Skip if already initialized
        if (this.isInitialized) {
            console.log('[CredentialManager] Already initialized, skipping');
            return;
        }

        try {
            // Dynamically import Stronghold only in desktop environment
            const {Client, Stronghold} = await import('@tauri-apps/plugin-stronghold');
            const {appDataDir} = await import('@tauri-apps/api/path');

            const vaultPath = `${await appDataDir()}/whodb-vault.stronghold`;
            console.log('[CredentialManager] Initializing Stronghold at:', vaultPath);

            // Initialize or load existing Stronghold
            this.stronghold = await Stronghold.load(vaultPath, masterPassword || 'default-password');
            this.strongholdClient = await this.stronghold.createClient('whodb-client');
            this.strongholdStore = this.strongholdClient.getStore();
            this.masterPasswordSet = !!masterPassword;
            this.isInitialized = true;

            console.log('[CredentialManager] Stronghold initialized successfully');
        } catch (error) {
            console.error('[CredentialManager] Failed to initialize Stronghold:', error);
            // Fall back to sessionStorage
            this.isDesktop = false;
            this.isInitialized = false;
        }
    }

    /**
     * Store credentials securely
     */
    async storeCredentials(credentials: any): Promise<void> {
        // Clear caches when storing new credentials
        this.cachedCredentials = null;
        this.cachedHeader = null;

        const data: StoredCredentials = {
            credentials,
            timestamp: Date.now()
        };

        if (this.isDesktop && this.strongholdStore) {
            // Desktop: Use Stronghold Store
            try {
                const key = 'whodb-db-credentials';
                const encoder = new TextEncoder();
                const dataBytes = Array.from(encoder.encode(JSON.stringify(data)));

                await this.strongholdStore.insert(key, dataBytes);
                await this.stronghold.save();
                console.log('[CredentialManager] Credentials stored in Stronghold');
            } catch (error) {
                console.error('[CredentialManager] Stronghold storage failed:', error);
                // Fallback to sessionStorage
                this.storeInSession(data);
            }
        } else {
            // Web or fallback: Use sessionStorage
            this.storeInSession(data);
        }
    }

    /**
     * Retrieve stored credentials
     */
    async getCredentials(): Promise<any | null> {
        // Return cached credentials if available
        if (this.cachedCredentials) {
            console.log('[CredentialManager] Using cached credentials');
            return this.cachedCredentials;
        }

        if (this.isDesktop && this.strongholdStore) {
            // Desktop: Get from Stronghold Store
            try {
                const key = 'whodb-db-credentials';
                const dataBytes = await this.strongholdStore.get(key);

                if (dataBytes && dataBytes.length > 0) {
                    const decoder = new TextDecoder();
                    const dataStr = decoder.decode(new Uint8Array(dataBytes));
                    const data: StoredCredentials = JSON.parse(dataStr);

                    // Check if credentials are expired (24 hours)
                    if (Date.now() - data.timestamp > 24 * 60 * 60 * 1000) {
                        console.log('[CredentialManager] Credentials expired');
                        await this.clearCredentials();
                        return null;
                    }

                    console.log('[CredentialManager] Retrieved credentials from Stronghold');
                    // Cache the credentials for future use
                    this.cachedCredentials = data.credentials;
                    return data.credentials;
                }
            } catch (error) {
                console.error('[CredentialManager] Stronghold retrieval failed:', error);
                // Fallback to sessionStorage
                const sessionCreds = this.getFromSession();
                if (sessionCreds) {
                    this.cachedCredentials = sessionCreds;
                }
                return sessionCreds;
            }
        } else {
            // Web or fallback: Get from sessionStorage
            const sessionCreds = this.getFromSession();
            if (sessionCreds) {
                this.cachedCredentials = sessionCreds;
            }
            return sessionCreds;
        }

        return null;
    }

    /**
     * Clear stored credentials
     */
    async clearCredentials(): Promise<void> {
        // Clear caches
        this.cachedCredentials = null;
        this.cachedHeader = null;

        if (this.isDesktop && this.strongholdStore) {
            // Desktop: Clear from Stronghold Store
            try {
                const key = 'whodb-db-credentials';

                await this.strongholdStore.delete(key);
                await this.stronghold.save();
                console.log('[CredentialManager] Credentials cleared from Stronghold');
            } catch (error) {
                console.error('[CredentialManager] Stronghold clear failed:', error);
            }
        }

        // Always clear sessionStorage as well
        sessionStorage.removeItem('whodb_temp_credentials');
        console.log('[CredentialManager] Session storage cleared');
    }

    /**
     * Get credentials as base64-encoded string for header
     */
    async getCredentialsHeader(): Promise<string | null> {
        // Return cached header if available
        if (this.cachedHeader) {
            console.log('[CredentialManager] Using cached header');
            return this.cachedHeader;
        }

        const credentials = await this.getCredentials();
        if (credentials) {
            // Cache the encoded header
            this.cachedHeader = btoa(JSON.stringify(credentials));
            console.log('[CredentialManager] Cached credentials header');
            return this.cachedHeader;
        }
        return null;
    }

    /**
     * Check if running in desktop mode with Stronghold available
     */
    isStrongholdAvailable(): boolean {
        return this.isDesktop && !!this.strongholdStore;
    }

    /**
     * Check if master password has been set (desktop only)
     */
    isMasterPasswordSet(): boolean {
        return this.masterPasswordSet;
    }

    // Private helper methods for sessionStorage fallback
    private storeInSession(data: StoredCredentials): void {
        // Use sessionStorage for web (cleared on tab close)
        // Never use localStorage for credentials
        const encoded = btoa(JSON.stringify(data));
        sessionStorage.setItem('whodb_temp_credentials', encoded);
        console.log('[CredentialManager] Credentials stored in session (temporary)');
    }

    private getFromSession(): any | null {
        const encoded = sessionStorage.getItem('whodb_temp_credentials');
        if (encoded) {
            try {
                const data: StoredCredentials = JSON.parse(atob(encoded));

                // Check expiration
                if (Date.now() - data.timestamp > 24 * 60 * 60 * 1000) {
                    console.log('[CredentialManager] Session credentials expired');
                    sessionStorage.removeItem('whodb_temp_credentials');
                    return null;
                }

                console.log('[CredentialManager] Retrieved credentials from session');
                return data.credentials;
            } catch (error) {
                console.error('[CredentialManager] Failed to parse session credentials:', error);
                return null;
            }
        }
        return null;
    }
}

// Singleton instance
export const credentialManager = new CredentialManager();