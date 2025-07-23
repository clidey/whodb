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

import { IAIModelType } from '../store/ai-models';

/**
 * Ensures modelTypes is always an array with at least default values
 */
export function ensureModelTypesArray(modelTypes: any): IAIModelType[] {
  if (Array.isArray(modelTypes)) {
    return modelTypes;
  }
  
  // Return default Ollama model type
  return [{
    id: "Ollama",
    modelType: "Ollama",
  }];
}

/**
 * Ensures models is always an array
 */
export function ensureModelsArray(models: any): string[] {
  if (Array.isArray(models)) {
    return models;
  }
  
  return [];
}