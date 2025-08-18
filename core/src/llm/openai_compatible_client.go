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
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
)

func getOpenAICompatibleModels() ([]string, error) {
	if len(env.CustomModels) > 0 {
		log.Logger.Infof("Using %d custom models for OpenAI-compatible service", len(env.CustomModels))
		return env.CustomModels, nil
	}
	log.Logger.Info("No custom models configured for OpenAI-compatible service")
	return []string{}, nil
}
