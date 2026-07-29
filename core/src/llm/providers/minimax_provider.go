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

package providers

const (
	MiniMax_LLMType LLMType = "MiniMax"
)

// MiniMaxDefaultEndpoint is the global OpenAI-compatible base URL for MiniMax.
// Deployments in mainland China can point WHODB_MINIMAX_ENDPOINT at
// https://api.minimaxi.com/v1 instead; both regions speak the same protocol.
const MiniMaxDefaultEndpoint = "https://api.minimax.io/v1"

// NewMiniMaxProvider creates a first-class MiniMax provider backed by the
// OpenAI-compatible adapter. Models (e.g. MiniMax-M3, MiniMax-M2.7) are
// discovered from the provider's /models endpoint.
func NewMiniMaxProvider() *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(MiniMax_LLMType, MiniMaxDefaultEndpoint)
}
