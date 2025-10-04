/**
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

import { IAIModelType } from '../types/ai';
import { ensureModelTypesArray, ensureModelsArray } from '../utils/ai-models-helper';

// Define the old database state structure for migration purposes
interface OldDatabaseState {
  schema: string;
  current?: IAIModelType;
  modelTypes?: IAIModelType[];
  currentModel?: string;
  models?: string[];
}

/**
 * Migrates AI-related data from the old database store to the new aiModels store
 */
export function migrateAIModelsFromDatabase(): void {
  try {
    // Check if migration has already been performed
    const migrationKey = 'aiModels_migration_v1_completed';
    if (localStorage.getItem(migrationKey) === 'true') {
      return;
    }

    // Get the persisted database state
    const persistedDatabaseState = localStorage.getItem('persist:database');
    if (!persistedDatabaseState) {
      // No database state to migrate
      localStorage.setItem(migrationKey, 'true');
      return;
    }

    // Parse the persisted state
    const databaseState = JSON.parse(persistedDatabaseState);
    
    // Redux persist stores each field as a JSON string, so we need to parse them
    let current, modelTypes, currentModel, models;
    
    try {
      current = databaseState.current ? JSON.parse(databaseState.current) : undefined;
      modelTypes = databaseState.modelTypes ? JSON.parse(databaseState.modelTypes) : undefined;
      currentModel = databaseState.currentModel ? JSON.parse(databaseState.currentModel) : undefined;
      models = databaseState.models ? JSON.parse(databaseState.models) : undefined;
    } catch (e) {
      // If parsing fails, the data might be in a different format
      current = databaseState.current;
      modelTypes = databaseState.modelTypes;
      currentModel = databaseState.currentModel;
      models = databaseState.models;
    }
    
    // Check if there's AI-related data to migrate
    if (!modelTypes && !current && !currentModel && !models) {
      // No AI data to migrate
      localStorage.setItem(migrationKey, 'true');
      return;
    }

    // Create the new aiModels state
    const aiModelsState: any = {};
    
    // Ensure we have at least the default model types if none exist
    const defaultModelTypes = ["Ollama"].map(modelType => ({
      id: modelType,
      modelType,
    }));
    
    if (current !== undefined) {
      aiModelsState.current = JSON.stringify(current);
    }
    
    if (modelTypes !== undefined && Array.isArray(modelTypes)) {
      aiModelsState.modelTypes = JSON.stringify(modelTypes);
    } else {
      // Ensure we always have a valid modelTypes array
      aiModelsState.modelTypes = JSON.stringify(defaultModelTypes);
    }
    
    if (currentModel !== undefined) {
      aiModelsState.currentModel = JSON.stringify(currentModel);
    }
    
    if (models !== undefined && Array.isArray(models)) {
      aiModelsState.models = JSON.stringify(models);
    } else {
      aiModelsState.models = JSON.stringify([]);
    }

    // Add Redux persist metadata
    aiModelsState._persist = JSON.stringify({ version: -1, rehydrated: true });

    // Save the new aiModels state
    localStorage.setItem('persist:aiModels', JSON.stringify(aiModelsState));

    // Clean up the old database state - keep only the schema
    const cleanedDatabaseState = {
      schema: databaseState.schema || '""',
      _persist: JSON.stringify({ version: -1, rehydrated: true })
    };
    localStorage.setItem('persist:database', JSON.stringify(cleanedDatabaseState));

    // Mark migration as completed
    localStorage.setItem(migrationKey, 'true');
      } catch (error) {
    console.error('Error during AI models migration:', error);
    // Don't mark as completed so it can be retried on next load
  }
}

/**
 * Ensure AI models state has valid array values
 */
function ensureValidAIModelsState(): void {
  try {
    const persistedAIModelsState = localStorage.getItem('persist:aiModels');
    if (!persistedAIModelsState) {
      return;
    }

    const aiModelsState = JSON.parse(persistedAIModelsState);
    let needsUpdate = false;

    // Check and fix modelTypes
    if (aiModelsState.modelTypes) {
      try {
        let modelTypes = JSON.parse(aiModelsState.modelTypes);
        if (!Array.isArray(modelTypes)) {
          modelTypes = ensureModelTypesArray(modelTypes);
          aiModelsState.modelTypes = JSON.stringify(modelTypes);
          needsUpdate = true;
        } else {
          // Ensure each modelType has required properties
          const validModelTypes = modelTypes.filter((mt: any) => mt && mt.id && mt.modelType);
          if (validModelTypes.length !== modelTypes.length) {
            modelTypes = validModelTypes.length > 0 ? validModelTypes : ensureModelTypesArray(null);
            aiModelsState.modelTypes = JSON.stringify(modelTypes);
            needsUpdate = true;
          }
        }
      } catch (e) {
        aiModelsState.modelTypes = JSON.stringify(ensureModelTypesArray(null));
        needsUpdate = true;
      }
    } else {
      // Initialize with default if missing
      aiModelsState.modelTypes = JSON.stringify(ensureModelTypesArray(null));
      needsUpdate = true;
    }

    // Check and fix models
    if (aiModelsState.models) {
      try {
        const models = JSON.parse(aiModelsState.models);
        if (!Array.isArray(models)) {
          aiModelsState.models = JSON.stringify(ensureModelsArray(models));
          needsUpdate = true;
        }
      } catch (e) {
        aiModelsState.models = JSON.stringify(ensureModelsArray(null));
        needsUpdate = true;
      }
    } else {
      // Initialize with empty array if missing
      aiModelsState.models = JSON.stringify([]);
      needsUpdate = true;
    }

    // Ensure _persist metadata exists
    if (!aiModelsState._persist) {
      aiModelsState._persist = JSON.stringify({ version: -1, rehydrated: true });
      needsUpdate = true;
    }

    if (needsUpdate) {
      localStorage.setItem('persist:aiModels', JSON.stringify(aiModelsState));
    }
  } catch (error) {
    console.error('Error ensuring valid AI models state:', error);
  }
}

/**
 * Get the current migration version
 */
function getMigrationVersion(): number {
  const version = localStorage.getItem('whodb_migration_version');
  return version ? parseInt(version, 10) : 0;
}

/**
 * Set the migration version
 */
function setMigrationVersion(version: number): void {
  localStorage.setItem('whodb_migration_version', version.toString());
}

/**
 * Migrate from old AI models system to new provider-based system
 */
function migrateToAIProviders(): void {
  try {
    // Check if migration has already been performed
    const migrationKey = 'aiProviders_migration_v2_completed';
    if (localStorage.getItem(migrationKey) === 'true') {
      return;
    }

    // Get the persisted aiModels state
    const persistedAIModelsState = localStorage.getItem('persist:aiModels');
    if (!persistedAIModelsState) {
      // No AI models state to migrate, create empty providers state
      const aiProvidersState = {
        providers: JSON.stringify([]),
        models: JSON.stringify([]),
        _persist: JSON.stringify({ version: -1, rehydrated: true })
      };
      localStorage.setItem('persist:aiProviders', JSON.stringify(aiProvidersState));
      localStorage.setItem(migrationKey, 'true');
      return;
    }

    // Parse the persisted state
    const aiModelsState = JSON.parse(persistedAIModelsState);

    // Parse individual fields
    let current, modelTypes, currentModel, models;
    try {
      current = aiModelsState.current ? JSON.parse(aiModelsState.current) : undefined;
      modelTypes = aiModelsState.modelTypes ? JSON.parse(aiModelsState.modelTypes) : [];
      currentModel = aiModelsState.currentModel ? JSON.parse(aiModelsState.currentModel) : undefined;
      models = aiModelsState.models ? JSON.parse(aiModelsState.models) : [];
    } catch (e) {
      modelTypes = [];
      models = [];
    }

    // Convert old modelTypes to new providers format
    const providers: any[] = [];
    let currentProviderId: string | undefined;

    if (Array.isArray(modelTypes)) {
      modelTypes.forEach((modelType: any) => {
        if (modelType && modelType.id && modelType.modelType) {
          // Create a provider from the old model type
          const provider = {
            id: modelType.id,
            name: modelType.modelType + (modelType.isEnvironmentDefined ? ' (Environment)' : ''),
            type: modelType.modelType,
            isEnvironmentDefined: modelType.isEnvironmentDefined || false,
            isUserDefined: !modelType.isEnvironmentDefined,
            settings: {}
          };

          // If this was the current model type, save its ID
          if (current && current.id === modelType.id) {
            currentProviderId = provider.id;
          }

          providers.push(provider);
        }
      });
    }

    // Create the new aiProviders state
    const aiProvidersState: any = {
      providers: JSON.stringify(providers),
      models: JSON.stringify(models || []),
      currentModel: JSON.stringify(currentModel || null),
      _persist: JSON.stringify({ version: -1, rehydrated: true })
    };

    if (currentProviderId) {
      aiProvidersState.currentProviderId = JSON.stringify(currentProviderId);
    }

    // Save the new aiProviders state
    localStorage.setItem('persist:aiProviders', JSON.stringify(aiProvidersState));

    // Remove the old aiModels state
    localStorage.removeItem('persist:aiModels');

    // Mark migration as completed
    localStorage.setItem(migrationKey, 'true');
  } catch (error) {
    console.error('Error during AI providers migration:', error);
  }
}

/**
 * Run all necessary migrations
 */
export function runMigrations(): void {
  const currentVersion = getMigrationVersion();

  // Run migrations in order based on version
  if (currentVersion < 1) {
    migrateAIModelsFromDatabase();
    setMigrationVersion(1);
  }

  if (currentVersion < 2) {
    migrateToAIProviders();
    setMigrationVersion(2);
  }

  // Always ensure AI models state is valid
  ensureValidAIModelsState();
}
