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

import type { IAIModelType } from './ai-models';
import { ensureModelTypesArray, ensureModelsArray } from '../utils/ai-models-helper';
import { featureFlags } from '../config/features';
import { withBasePath } from '../utils/base-path';
import { stripProfileSecrets } from '../utils/credential-secrets';


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
    } catch {
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
    const aiModelsState: Record<string, string> = {};
    
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
      schema: databaseState.schema ?? '""',
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
          const validModelTypes = modelTypes.filter((mt: IAIModelType | null) => mt?.id && mt.modelType);
          if (validModelTypes.length !== modelTypes.length) {
            modelTypes = validModelTypes.length > 0 ? validModelTypes : ensureModelTypesArray(null);
            aiModelsState.modelTypes = JSON.stringify(modelTypes);
            needsUpdate = true;
          }
        }
      } catch {
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
      } catch {
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
 * Clear chat state to fix IChatMessage type mismatch
 */
function clearChatStateV2(): void {
  try {
    // Clear the persisted chat (houdini) state
    localStorage.removeItem('persist:houdini');
  } catch (error) {
    console.error('Error clearing chat state:', error);
  }
}

/**
 * Apply extension settings defaults
 */
function applyEESettingsDefaultsV4(): void {
  try {
    // Only run when settings page is hidden (extended mode)
    if (featureFlags.settingsPage) {
      return;
    }

    const persistedSettingsState = localStorage.getItem('persist:settings');
    if (!persistedSettingsState) {
      return;
    }

    const settingsState = JSON.parse(persistedSettingsState);
    let needsUpdate = false;

    // Update whereConditionMode to extension default ('sheet')
    if (settingsState.whereConditionMode) {
      try {
        const whereConditionMode = JSON.parse(settingsState.whereConditionMode);
        if (whereConditionMode !== 'sheet') {
          settingsState.whereConditionMode = JSON.stringify('sheet');
          needsUpdate = true;
        }
      } catch {
        // If parsing fails, set to extension default
        settingsState.whereConditionMode = JSON.stringify('sheet');
        needsUpdate = true;
      }
    } else {
      // Initialize with extension default if missing
      settingsState.whereConditionMode = JSON.stringify('sheet');
      needsUpdate = true;
    }

    // Update disableAnimations to extension default (true)
    if (settingsState.disableAnimations) {
      try {
        const disableAnimations = JSON.parse(settingsState.disableAnimations);
        if (disableAnimations !== true) {
          settingsState.disableAnimations = JSON.stringify(true);
          needsUpdate = true;
        }
      } catch {
        // If parsing fails, set to extension default
        settingsState.disableAnimations = JSON.stringify(true);
        needsUpdate = true;
      }
    } else {
      // Initialize with extension default if missing
      settingsState.disableAnimations = JSON.stringify(true);
      needsUpdate = true;
    }

    if (needsUpdate) {
      localStorage.setItem('persist:settings', JSON.stringify(settingsState));
    }
  } catch (error) {
    console.error('Error applying settings defaults:', error);
  }
}

/**
 * Migrate language codes from old format (en, es, fr, de) to regional variants
 * (en_US, es_ES, fr_FR, de_DE).
 */
function migrateLanguageCodesV5(): void {
  try {
    const persistedSettingsState = localStorage.getItem('persist:settings');
    if (!persistedSettingsState) {
      return;
    }

    const settingsState = JSON.parse(persistedSettingsState);
    if (!settingsState.language) {
      return;
    }

    const languageMap: Record<string, string> = {
      'en': 'en_US',
      'es': 'es_ES',
      'fr': 'fr_FR',
      'de': 'de_DE',
    };

    const currentLanguage = JSON.parse(settingsState.language);
    const mapped = languageMap[currentLanguage];
    if (mapped) {
      settingsState.language = JSON.stringify(mapped);
      localStorage.setItem('persist:settings', JSON.stringify(settingsState));
    }
  } catch (error) {
    console.error('Error migrating language codes:', error);
  }
}

/**
 * Remove the orphaned persisted database metadata state now that metadata is
 * owned by Apollo session state instead of Redux persistence.
 */
function clearDatabaseMetadataStateV6(): void {
  try {
    localStorage.removeItem('persist:databaseMetadata');
  } catch (error) {
    console.error('Error clearing database metadata state:', error);
  }
}

/**
 * Clear persisted auth state again after the source-first Authorization header
 * contract corrections so stale logged-in sessions from earlier source-refactor
 * builds do not survive.
 */
function clearAuthStateV8(): void {
  try {
    localStorage.removeItem('persist:auth');
  } catch (error) {
    console.error('Error clearing auth state:', error);
  }
}

const PASSWORD_KEY = 'password';

/**
 * Migrate an already-logged-in browser user to the server-side session cookie.
 * If persisted credentials carry a password and no session cookie exists yet,
 * this posts LoginSource once (awaited, so it completes before the app renders
 * and any query races it), then rewrites persist:auth with all secrets stripped
 * — keeping the connection list and logged-in state. Best-effort: on failure it
 * simply leaves the user to log in again. Skipped on desktop/webview.
 */
export async function migrateAuthSessionCookieV9(): Promise<void> {
  try {
    if (typeof window === 'undefined' || getMigrationVersion() >= 9) {
      return;
    }
    // Desktop/webview keeps the Authorization-header flow; do not migrate.
    const wailsGo = (window as any).go;
    const hasWailsBindings = Boolean(wailsGo?.main?.App) || Boolean(wailsGo?.common?.App);
    const isDesktop = hasWailsBindings
      || !['http:', 'https:'].includes(window.location.protocol);

    const raw = localStorage.getItem('persist:auth');
    const parsed = raw ? JSON.parse(raw) : null;
    const current = parsed?.current ? JSON.parse(parsed.current) : null;
    const values: Array<{ Key: string; Value: string }> = current?.Values ?? [];
    const hasPassword = Array.isArray(values)
      && values.some(v => String(v?.Key ?? '').toLowerCase() === PASSWORD_KEY && v?.Value);

    if (!isDesktop && current && hasPassword) {
      const res = await fetch(withBasePath('/api/query'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          operationName: 'LoginSource',
          query: 'mutation LoginSource($credentials: SourceLoginInput!) { LoginSource(credentials: $credentials) { Status } }',
          variables: {
            credentials: {
              Id: current.Id,
              SourceType: current.SourceType ?? current.Type,
              Values: values,
              AccessToken: current.AccessToken,
            },
          },
        }),
      });
      // Only strip secrets once the cookie is established, so a failed mint
      // leaves the credentials intact for a manual retry/login.
      if (res.ok && parsed) {
        if (parsed.current) parsed.current = JSON.stringify(stripProfileSecrets(current));
        if (parsed.profiles) {
          const profiles = JSON.parse(parsed.profiles);
          parsed.profiles = JSON.stringify(
            Array.isArray(profiles) ? profiles.map(stripProfileSecrets) : profiles
          );
        }
        localStorage.setItem('persist:auth', JSON.stringify(parsed));
      }
    }
  } catch (error) {
    console.error('Error migrating auth to session cookie:', error);
  } finally {
    setMigrationVersion(9);
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
    clearChatStateV2();
    setMigrationVersion(2);
  }

  if (currentVersion < 4) {
    applyEESettingsDefaultsV4();
    setMigrationVersion(4);
  }

  if (currentVersion < 5) {
    migrateLanguageCodesV5();
    setMigrationVersion(5);
  }

  if (currentVersion < 6) {
    clearDatabaseMetadataStateV6();
    setMigrationVersion(6);
  }

  if (currentVersion < 8) {
    clearAuthStateV8();
    setMigrationVersion(8);
  }

  // V9 (auth → session cookie) runs separately via migrateAuthSessionCookieV9,
  // which is awaited before the app renders so the cookie beats any query.

  // Always ensure AI models state is valid
  ensureValidAIModelsState();
}
