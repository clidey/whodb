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

package env

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
)

var IsDevelopment = os.Getenv("ENVIRONMENT") == "dev"
var IsEnterpriseEdition = false // Set at startup by the entry point

// ActiveDatabases lists database type names available in this build (populated from registered plugins at startup).
var ActiveDatabases []string

// GetIsDesktopMode returns true if running in desktop mode.
// This is a function (not a variable) so it reads the env var each time,
// allowing the desktop app to set WHODB_DESKTOP after package initialization.
func GetIsDesktopMode() bool {
	return os.Getenv("WHODB_DESKTOP") == "true"
}

// GetIsCLIMode returns true if running as the CLI/TUI application.
func GetIsCLIMode() bool {
	return os.Getenv("WHODB_CLI") == "true"
}

// GetIsLocalMode returns true if running locally (desktop or CLI) where
// full filesystem access is expected, as opposed to server mode.
func GetIsLocalMode() bool {
	return GetIsDesktopMode() || GetIsCLIMode()
}

var OllamaHost = os.Getenv("WHODB_OLLAMA_HOST")
var OllamaPort = os.Getenv("WHODB_OLLAMA_PORT")

var AnthropicAPIKey = os.Getenv("WHODB_ANTHROPIC_API_KEY")
var AnthropicEndpoint = os.Getenv("WHODB_ANTHROPIC_ENDPOINT")
var AnthropicName = os.Getenv("WHODB_ANTHROPIC_NAME")

var GeminiEndpoint = os.Getenv("WHODB_GEMINI_ENDPOINT")

var OpenAIAPIKey = os.Getenv("WHODB_OPENAI_API_KEY")
var OpenAIEndpoint = os.Getenv("WHODB_OPENAI_ENDPOINT")
var OpenAIName = os.Getenv("WHODB_OPENAI_NAME")

var OllamaName = os.Getenv("WHODB_OLLAMA_NAME")

var LMStudioBaseURL = os.Getenv("WHODB_LMSTUDIO_BASE_URL")
var LMStudioAPIKey = os.Getenv("WHODB_LMSTUDIO_API_KEY")
var LMStudioName = os.Getenv("WHODB_LMSTUDIO_NAME")

var AllowedOrigins = common.FilterList(strings.Split(os.Getenv("WHODB_ALLOWED_ORIGINS"), ","), func(item string) bool {
	return item != ""
})

var LogLevel = os.Getenv("WHODB_LOG_LEVEL")

var AccessLogFile = os.Getenv("WHODB_ACCESS_LOG_FILE") // where to store the http access logs
var LogFile = os.Getenv("WHODB_LOG_FILE")              // where to store all other non-http logs
var LogFormat = os.Getenv("WHODB_LOG_FORMAT")          // only option right now is "json". leave blank for default format

// BasePath is the optional URL path prefix used when WhoDB serves its bundled
// frontend, for example "/whodb".
var BasePath = getBasePath()

// Default log paths used when the AccessLogFile and LogFile vars are set to "default".
const DefaultLogDir = "/var/log/whodb"
const DefaultLogFile = DefaultLogDir + "/whodb.log"
const DefaultAccessLogFile = DefaultLogDir + "/whodb.access.log"

// GetDisableUpdateCheck returns true if update checking is disabled.
func GetDisableUpdateCheck() bool {
	return os.Getenv("WHODB_DISABLE_UPDATE_CHECK") == "true"
}

var DisableMockDataGeneration = os.Getenv("WHODB_DISABLE_MOCK_DATA_GENERATION")

var ApplicationEnvironment = os.Getenv("WHODB_APPLICATION_ENVIRONMENT")

// BlockInternalAIEndpoints enables SSRF protection on user-supplied AI provider
// endpoints (rejects loopback/link-local/private/metadata destinations). Off by
// default so self-hosted deployments can use local model servers (Ollama, etc.);
// multi-tenant/hosted deployments should set this to "true".
var BlockInternalAIEndpoints = os.Getenv("WHODB_BLOCK_INTERNAL_AI_ENDPOINTS") == "true"

var ApplicationVersion string

var PosthogAPIKey = "phc_hbXcCoPTdxm5ADL8PmLSYTIUvS6oRWFM2JAK8SMbfnH"
var PosthogHost = "https://us.i.posthog.com"

// IsAWSProviderEnabled controls whether AWS provider functionality is available.
// disabled by default for now until official release
var IsAWSProviderEnabled = os.Getenv("WHODB_ENABLE_AWS_PROVIDER") == "true"

// IsAzureProviderEnabled controls whether Azure provider functionality is available.
var IsAzureProviderEnabled = os.Getenv("WHODB_ENABLE_AZURE_PROVIDER") == "true"

// IsGCPProviderEnabled controls whether GCP provider functionality is available.
var IsGCPProviderEnabled = os.Getenv("WHODB_ENABLE_GCP_PROVIDER") == "true"

// IsNewUIEnabled controls whether the new UI visuals are available.
var IsNewUIEnabled = os.Getenv("WHODB_ENABLE_NEW_UI") == "true"

// DisableCredentialForm controls whether the credential form is disabled.
var DisableCredentialForm = os.Getenv("WHODB_DISABLE_CREDENTIAL_FORM") == "true"

var BridgeURL = os.Getenv("WHODB_BRIDGE_URL")

// MaxPageSize is the maximum number of rows that can be requested in a single
// page via the Row resolver. Configurable via WHODB_MAX_PAGE_SIZE (default 10000).
var MaxPageSize = getMaxPageSize()

type ChatProvider struct {
	Type       string
	Name       string // Display name/alias for the provider
	APIKey     string
	Endpoint   string
	ProviderId string
	ClientType string // BAML client type (openai-generic, anthropic, aws-bedrock) - only for generic providers
	IsGeneric  bool   // True for generic/custom providers, false for built-in providers
	Icon       string // Optional URL to a custom icon for this provider
}

// GenericProviderConfig holds configuration for a generic AI provider.
type GenericProviderConfig struct {
	ProviderId string
	Name       string
	ClientType string // "openai-generic", "anthropic", etc.
	BaseURL    string
	APIKey     string
	Models     []string
	Icon       string // Optional URL to a custom icon
}

var GenericProviders []GenericProviderConfig

// AddGenericProvider adds a generic provider to the GenericProviders list.
// Used by dynamic provider registration.
func AddGenericProvider(config GenericProviderConfig) {
	GenericProviders = append(GenericProviders, config)
}

// ResolvedCredentials holds provider credentials resolved from environment config.
type ResolvedCredentials struct {
	ModelType string // Provider type as sent by the frontend (ProviderId)
	Token     string // API key
	Endpoint  string // Base URL
}

// GetOllamaEndpoint returns the resolved Ollama API endpoint, accounting for
// WHODB_OLLAMA_HOST/PORT overrides and Docker/WSL2 environments.
func GetOllamaEndpoint() string {
	host := "localhost"
	port := "11434"
	if OllamaHost != "" {
		host = OllamaHost
	}
	if OllamaPort != "" {
		port = OllamaPort
	}
	return common.ResolveLocalURL(fmt.Sprintf("http://%s/api", net.JoinHostPort(host, port)))
}

func GetAnthropicEndpoint() string {
	if AnthropicEndpoint != "" {
		return AnthropicEndpoint
	}
	return "https://api.anthropic.com/v1"
}

func GetOpenAIEndpoint() string {
	if OpenAIEndpoint != "" {
		return OpenAIEndpoint
	}
	return "https://api.openai.com/v1"
}

func GetGeminiEndpoint() string {
	if GeminiEndpoint != "" {
		return GeminiEndpoint
	}
	return "https://generativelanguage.googleapis.com/v1beta/openai/"
}

// GetLMStudioEndpoint returns the configured LM Studio base URL, or the
// default localhost:1234 endpoint if no override is set. The result is
// Docker/WSL2-aware so callers don't need to wrap it with ResolveLocalURL.
func GetLMStudioEndpoint() string {
	if LMStudioBaseURL != "" {
		return common.ResolveLocalURL(LMStudioBaseURL)
	}
	return common.ResolveLocalURL("http://localhost:1234/v1")
}

func getMaxPageSize() int {
	val := os.Getenv("WHODB_MAX_PAGE_SIZE")
	if val == "" {
		return 10000
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return 10000
	}
	return n
}

func getBasePath() string {
	basePath := strings.TrimSpace(os.Getenv("WHODB_BASE_PATH"))
	if basePath == "" || basePath == "/" {
		return ""
	}

	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	basePath = strings.TrimRight(basePath, "/")
	if basePath == "" || basePath == "/" {
		return ""
	}

	if !isValidBasePath(basePath) {
		panic(fmt.Sprintf("invalid WHODB_BASE_PATH %q: must be a slash-prefixed path made of non-empty segments containing only letters, numbers, '.', '_' or '-'", basePath))
	}

	return basePath
}

func isValidBasePath(basePath string) bool {
	if !strings.HasPrefix(basePath, "/") {
		return false
	}

	segments := strings.Split(basePath, "/")[1:]
	if len(segments) == 0 {
		return false
	}

	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}

		for _, char := range segment {
			switch {
			case char >= 'a' && char <= 'z':
			case char >= 'A' && char <= 'Z':
			case char >= '0' && char <= '9':
			case char == '.', char == '_', char == '-':
			default:
				return false
			}
		}
	}

	return true
}
