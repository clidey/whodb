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

import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { AwsProvider, CloudProviderStatus, DiscoveredConnection } from '@graphql';

/**
 * Cloud Provider with local state tracking.
 * Currently supports AWS, with GCP/Azure planned.
 */
export type LocalCloudProvider = AwsProvider & {
  /** True if provider was configured via environment variable */
  IsEnvironmentDefined?: boolean;
};

/**
 * Discovered connection with provider source information.
 */
export type LocalDiscoveredConnection = DiscoveredConnection;

export interface IProvidersState {
  /** List of configured cloud providers (AWS, GCP, Azure, etc.) */
  cloudProviders: LocalCloudProvider[];
  /** List of all discovered connections from all providers */
  discoveredConnections: LocalDiscoveredConnection[];
  /** UI state: Add/edit modal visibility */
  isProviderModalOpen: boolean;
  /** UI state: Provider being edited (null for add mode) */
  editingProviderId: string | null;
  /** Loading state for provider operations */
  isLoading: boolean;
  /** Error message from last operation */
  error: string | null;
}

const initialState: IProvidersState = {
  cloudProviders: [],
  discoveredConnections: [],
  isProviderModalOpen: false,
  editingProviderId: null,
  isLoading: false,
  error: null,
};

export const providersSlice = createSlice({
  name: 'providers',
  initialState,
  reducers: {
    /**
     * Set the full list of cloud providers (typically after fetching from API).
     */
    setCloudProviders: (state, action: PayloadAction<LocalCloudProvider[]>) => {
      state.cloudProviders = action.payload;
      state.error = null;
    },

    /**
     * Add a new cloud provider.
     */
    addCloudProvider: (state, action: PayloadAction<LocalCloudProvider>) => {
      const existingIndex = state.cloudProviders.findIndex(p => p.Id === action.payload.Id);
      if (existingIndex >= 0) {
        state.cloudProviders[existingIndex] = action.payload;
      } else {
        state.cloudProviders.push(action.payload);
      }
      state.error = null;
    },

    /**
     * Update an existing cloud provider.
     */
    updateCloudProvider: (state, action: PayloadAction<LocalCloudProvider>) => {
      const index = state.cloudProviders.findIndex(p => p.Id === action.payload.Id);
      if (index >= 0) {
        state.cloudProviders[index] = action.payload;
      }
      state.error = null;
    },

    /**
     * Remove a cloud provider by ID.
     */
    removeCloudProvider: (state, action: PayloadAction<{ id: string }>) => {
      state.cloudProviders = state.cloudProviders.filter(p => p.Id !== action.payload.id);
      // Also remove associated discovered connections
      state.discoveredConnections = state.discoveredConnections.filter(
        c => c.ProviderID !== action.payload.id
      );
      state.error = null;
    },

    /**
     * Update provider status (e.g., after test or refresh).
     */
    setProviderStatus: (state, action: PayloadAction<{ id: string; status: CloudProviderStatus; error?: string }>) => {
      const provider = state.cloudProviders.find(p => p.Id === action.payload.id);
      if (provider) {
        provider.Status = action.payload.status;
        if (action.payload.error) {
          state.error = action.payload.error;
        }
      }
    },

    /**
     * Update provider after discovery refresh.
     */
    updateProviderDiscovery: (state, action: PayloadAction<{
      id: string;
      discoveredCount: number;
      lastDiscoveryAt: string;
    }>) => {
      const provider = state.cloudProviders.find(p => p.Id === action.payload.id);
      if (provider) {
        provider.DiscoveredCount = action.payload.discoveredCount;
        provider.LastDiscoveryAt = action.payload.lastDiscoveryAt;
        provider.Status = CloudProviderStatus.Connected;
      }
    },

    /**
     * Set the full list of discovered connections.
     */
    setDiscoveredConnections: (state, action: PayloadAction<LocalDiscoveredConnection[]>) => {
      state.discoveredConnections = action.payload;
    },

    /**
     * Update discovered connections for a specific provider.
     */
    setProviderConnections: (state, action: PayloadAction<{
      providerId: string;
      connections: LocalDiscoveredConnection[];
    }>) => {
      // Remove old connections for this provider
      state.discoveredConnections = state.discoveredConnections.filter(
        c => c.ProviderID !== action.payload.providerId
      );
      // Add new connections
      state.discoveredConnections.push(...action.payload.connections);
    },

    /**
     * Open the provider modal for adding a new provider.
     */
    openAddProviderModal: (state) => {
      state.isProviderModalOpen = true;
      state.editingProviderId = null;
      state.error = null;
    },

    /**
     * Open the provider modal for editing an existing provider.
     */
    openEditProviderModal: (state, action: PayloadAction<{ id: string }>) => {
      state.isProviderModalOpen = true;
      state.editingProviderId = action.payload.id;
      state.error = null;
    },

    /**
     * Close the provider modal.
     */
    closeProviderModal: (state) => {
      state.isProviderModalOpen = false;
      state.editingProviderId = null;
    },

    /**
     * Set loading state for async operations.
     */
    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },

    /**
     * Set error message.
     */
    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },

    /**
     * Clear all provider state (e.g., on logout).
     */
    clearProviders: (state) => {
      state.cloudProviders = [];
      state.discoveredConnections = [];
      state.isProviderModalOpen = false;
      state.editingProviderId = null;
      state.isLoading = false;
      state.error = null;
    },
  },
});

export const ProvidersActions = providersSlice.actions;
export const providersReducers = providersSlice.reducer;
