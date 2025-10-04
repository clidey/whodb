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

package llm

import (
    "fmt"
    "io"

    "github.com/clidey/whodb/core/src/env"
    "github.com/clidey/whodb/core/src/log"
)

// getOpenAICompatibleModelsForConfig attempts to fetch models from the provider's /models endpoint.
// Falls back to env.CustomModels if request fails or returns non-200.
func getOpenAICompatibleModelsForConfig(config *ProviderConfig) ([]string, error) {
    // Best-effort request to the provider's models endpoint
    if config != nil && config.BaseURL != "" {
        url := fmt.Sprintf("%v/models", config.BaseURL)
        headers := map[string]string{
            "Content-Type": "application/json",
        }
        if config.APIKey != "" {
            headers["Authorization"] = fmt.Sprintf("Bearer %s", config.APIKey)
        }

        resp, err := sendHTTPRequest("GET", url, nil, headers)
        if err == nil {
            defer resp.Body.Close()
            if resp.StatusCode == 200 {
                // Reuse OpenAI models response parser
                return parseChatGPTModelsResponse(resp.Body)
            }
            // Drain body for logging clarity
            body, _ := io.ReadAll(resp.Body)
            log.Logger.WithField("status", resp.StatusCode).Warnf("OpenAI-compatible models endpoint returned non-OK; falling back. Body: %s", string(body))
        } else {
            log.Logger.WithError(err).Warn("Failed to fetch OpenAI-compatible models; falling back to env models")
        }
    }

    // Fallback to environment models if configured
    if len(env.CustomModels) > 0 {
        return env.CustomModels, nil
    }
    return []string{}, nil
}
