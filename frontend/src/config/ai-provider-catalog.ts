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

export type AIProviderStatus = "available" | "coming-soon";

export type AIProviderCatalogEntry = {
  id: string;
  label: string;
  iconKey?: string;
  status: AIProviderStatus;
  requiresEndpoint?: boolean;
  defaultEndpoint?: string;
};

const providerCatalog: AIProviderCatalogEntry[] = [
  { id: "OpenAI", label: "OpenAI", iconKey: "OpenAI", status: "available", defaultEndpoint: "https://api.openai.com/v1" },
  { id: "Anthropic", label: "Anthropic", iconKey: "Anthropic", status: "available", defaultEndpoint: "https://api.anthropic.com/v1" },
  { id: "Gemini", label: "Gemini API / AI Studio (API key)", iconKey: "Gemini", status: "available", defaultEndpoint: "https://generativelanguage.googleapis.com/v1beta/openai/" },
  { id: "Ollama", label: "Ollama", iconKey: "Ollama", status: "available", defaultEndpoint: "http://localhost:11434/api" },
  { id: "LMStudio", label: "LM Studio", iconKey: "LMStudio", status: "available", defaultEndpoint: "http://localhost:1234/v1" },
];

/** Returns AI provider catalog entries visible in this edition. */
export function getAIProviderCatalog(): AIProviderCatalogEntry[] {
  return providerCatalog;
}

/** Returns AI provider catalog entries that are executable today. */
export function getAvailableAIProviderCatalog(): AIProviderCatalogEntry[] {
  return getAIProviderCatalog().filter(provider => provider.status === "available");
}

/** Finds an AI provider catalog entry by provider identifier. */
export function findAIProviderCatalogEntry(id: string): AIProviderCatalogEntry | undefined {
  return providerCatalog.find(provider => provider.id === id);
}
