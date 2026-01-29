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

import { HealthActions } from '@/store/health';
import { reduxStore } from '@/store';
import { GetHealthDocument } from '@graphql';
import { graphqlClient } from '@/config/graphql-client';

type HealthResponse = {
    server: 'healthy' | 'error' | 'unavailable';
    database: 'healthy' | 'error' | 'unavailable';
};

type HealthCheckConfig = {
    initialInterval: number;
    maxInterval: number;
    backoffMultiplier: number;
};

const DEFAULT_CONFIG: HealthCheckConfig = {
    initialInterval: 5000, // Start with 5 seconds
    maxInterval: 60000, // Max 60 seconds
    backoffMultiplier: 1.5, // Exponential backoff multiplier
};

/**
 * HealthCheckService manages periodic health checks with exponential backoff.
 * Monitors both server and database connectivity status.
 */
class HealthCheckService {
    private intervalId: NodeJS.Timeout | null = null;
    private currentInterval: number;
    private config: HealthCheckConfig;
    private consecutiveFailures: number = 0;
    private isRunning: boolean = false;
    private wasDown: boolean = false; // Track if server was previously down
    private wasDatabaseDown: boolean = false; // Track if database was previously down

    constructor(config: Partial<HealthCheckConfig> = {}) {
        this.config = { ...DEFAULT_CONFIG, ...config };
        this.currentInterval = this.config.initialInterval;
    }

    /**
     * Performs a single health check using GraphQL Health query.
     */
    private async checkHealth(): Promise<HealthResponse | null> {
        try {
            const result = await graphqlClient.query({
                query: GetHealthDocument,
                fetchPolicy: 'network-only', // Always fetch fresh data
                errorPolicy: 'all', // Return partial data even if there are errors
            });

            // Check for GraphQL errors
            if (result.errors && result.errors.length > 0) {
                return null;
            }

            // Check if data exists
            if (!result.data || !result.data.Health) {
                return null;
            }

            const health = result.data.Health;

            return {
                server: (health.Server?.toLowerCase() || 'unavailable') as 'healthy' | 'error' | 'unavailable',
                database: (health.Database?.toLowerCase() || 'unavailable') as 'healthy' | 'error' | 'unavailable',
            };
        } catch (error) {
            // Network error or server unreachable
            return null;
        }
    }

    /**
     * Executes a health check and updates Redux state.
     * Reloads the page when connection is restored after being down.
     */
    private async performHealthCheck(): Promise<void> {
        const result = await this.checkHealth();

        if (result === null) {
            // Server is down
            reduxStore.dispatch(HealthActions.setHealthStatus({
                server: 'error',
                database: 'unavailable',
            }));
            this.consecutiveFailures++;
            this.wasDown = true;
            this.wasDatabaseDown = true;
            this.increaseInterval();
        } else {
            // Server is up
            const isServerHealthy = result.server === 'healthy';
            const isDatabaseHealthy = result.database === 'healthy';
            const isFullyHealthy = isServerHealthy && isDatabaseHealthy;

            // Track if database was in error state
            const isDatabaseError = result.database === 'error';
            if (isDatabaseError) {
                this.wasDatabaseDown = true;
            }

            reduxStore.dispatch(HealthActions.setHealthStatus({
                server: isServerHealthy ? 'healthy' : 'error',
                database: result.database,
            }));

            // Reload if server or database recovered after being down
            const serverRecovered = this.wasDown && isServerHealthy;
            const databaseRecovered = this.wasDatabaseDown && isDatabaseHealthy;

            if (serverRecovered || databaseRecovered) {
                window.location.reload();
                return;
            }

            // Reset backoff if fully healthy
            if (isFullyHealthy) {
                this.resetInterval();
            } else {
                this.consecutiveFailures++;
                this.increaseInterval();
            }
        }
    }

    /**
     * Increases the check interval using exponential backoff.
     */
    private increaseInterval(): void {
        this.currentInterval = Math.min(
            this.currentInterval * this.config.backoffMultiplier,
            this.config.maxInterval
        );
    }

    /**
     * Resets the check interval to the initial value.
     */
    private resetInterval(): void {
        if (this.consecutiveFailures > 0) {
            this.consecutiveFailures = 0;
            this.currentInterval = this.config.initialInterval;
            this.reschedule();
        }
    }

    /**
     * Reschedules the health check with the current interval.
     */
    private reschedule(): void {
        if (this.intervalId !== null) {
            clearTimeout(this.intervalId);
        }

        if (this.isRunning) {
            this.intervalId = setTimeout(() => {
                this.performHealthCheck();
                this.reschedule();
            }, this.currentInterval);
        }
    }

    /**
     * Starts the health check service.
     */
    start(): void {
        if (this.isRunning) {
            return;
        }

        this.isRunning = true;
        this.consecutiveFailures = 0;
        this.wasDown = false;
        this.wasDatabaseDown = false;
        this.currentInterval = this.config.initialInterval;

        // Perform first check immediately
        this.performHealthCheck();

        // Schedule recurring checks
        this.reschedule();
    }

    /**
     * Stops the health check service.
     */
    stop(): void {
        this.isRunning = false;
        if (this.intervalId !== null) {
            clearTimeout(this.intervalId);
            this.intervalId = null;
        }
    }

    /**
     * Forces an immediate health check without waiting for the next scheduled check.
     */
    async forceCheck(): Promise<void> {
        await this.performHealthCheck();
    }
}

// Singleton instance
export const healthCheckService = new HealthCheckService();
