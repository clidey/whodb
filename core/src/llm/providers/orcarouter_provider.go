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
	OrcaRouter_LLMType LLMType = "OrcaRouter"
)

// OrcaRouterProvider implements the AIProvider interface for the OrcaRouter
// gateway. OrcaRouter is an OpenAI-compatible gateway that exposes models from
// many providers (OpenAI, Anthropic, Google, DeepSeek, Qwen, MiniMax, xAI)
// behind a single endpoint, using provider/model model slugs.
type OrcaRouterProvider struct {
	*OpenAICompatibleProvider
}

// NewOrcaRouterProvider creates a new OrcaRouter provider instance backed by
// the OpenAI-compatible adapter.
func NewOrcaRouterProvider() *OrcaRouterProvider {
	return &OrcaRouterProvider{
		OpenAICompatibleProvider: NewOpenAICompatibleProvider(OrcaRouter_LLMType, "https://api.orcarouter.ai/v1"),
	}
}
