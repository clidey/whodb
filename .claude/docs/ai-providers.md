# AI Provider Environment Variables

Environment variable configuration for AI chat providers. All providers configured via env vars appear in the UI with a lock icon (non-editable).

For all providers, `_MODEL` (singular) is accepted as a fallback when `_MODELS` (plural) is not set.

## CE Providers

### Built-in (OpenAI, Anthropic, Ollama)

Documented in `environment-variables.md` under "AI Provider Variables".

### Google AI (Gemini)

Parsed in `core/src/envconfig/envconfig.go:ParseGoogleAIProvider()`. Registered in `core/src/src.go:InitializeEngine()`. Uses the `GenericProvider` with `clientType: "google-ai"`.

| Variable | Required | Default | Description |
|---|---|---|---|
| `WHODB_GOOGLE_AI_API_KEY` | Yes | | API key (falls back to `GOOGLE_API_KEY`) |
| `WHODB_GOOGLE_AI_MODELS` | Yes | | Comma-separated model list (falls back to `WHODB_GOOGLE_AI_MODEL`) |
| `WHODB_GOOGLE_AI_BASE_URL` | No | `https://generativelanguage.googleapis.com/v1beta` | API endpoint |
| `WHODB_GOOGLE_AI_NAME` | No | `Google AI` | Display name in dropdown |

### Generic Providers

Documented in `environment-variables.md` under "Generic AI providers". `_MODEL` singular fallback also supported.

## EE Providers

All EE providers are registered via `init()` functions in `ee/core/src/llm/providers/` that call `RegisterEEProvider()`. The registry in `ee/core/src/llm/providers/registry.go` validates and initializes them.

### Azure OpenAI

File: `ee/core/src/llm/providers/azure_openai_provider.go`

| Variable | Required | Default | Description |
|---|---|---|---|
| `WHODB_AZURE_OPENAI_BASE_URL` | Yes | | Full Azure endpoint URL (api-version can be in query string) |
| `WHODB_AZURE_OPENAI_API_KEY` | Yes | | API key (falls back to `AZURE_OPENAI_API_KEY`) |
| `WHODB_AZURE_OPENAI_MODELS` | Yes | | Comma-separated deployment names (falls back to `WHODB_AZURE_OPENAI_MODEL`) |
| `WHODB_AZURE_OPENAI_API_VERSION` | No | Extracted from URL | API version (e.g. `2024-06-01`) |
| `WHODB_AZURE_OPENAI_NAME` | No | `Azure OpenAI` | Display name |

URL parsing: `parseAzureOpenAIURL()` strips `api-version` from query string and returns it separately. BAML auto-derives `resource_name` and `deployment_id` from the base URL.

### Vertex AI

File: `ee/core/src/llm/providers/vertex_ai_provider.go`

| Variable | Required | Default | Description |
|---|---|---|---|
| `WHODB_VERTEX_AI_REGION` | Yes | | GCP region (e.g. `us-central1`) |
| `WHODB_VERTEX_AI_MODELS` | Yes | | Comma-separated model list (falls back to `WHODB_VERTEX_AI_MODEL`) |
| `WHODB_VERTEX_AI_API_KEY` | One of two | | Express Mode API key (mutually exclusive with credentials) |
| `WHODB_VERTEX_AI_CREDENTIALS` | One of two | | Service account: raw JSON, base64, or file path |
| `WHODB_VERTEX_AI_PROJECT_ID` | With API key | | GCP project ID (inferred from credentials JSON if not set) |
| `WHODB_VERTEX_AI_BASE_URL` | No | | Custom endpoint (BAML builds from project_id + region) |
| `WHODB_VERTEX_AI_NAME` | No | `Vertex AI` | Display name |

Auth modes:
- **API Key (Express Mode)**: Set `WHODB_VERTEX_AI_API_KEY` + `WHODB_VERTEX_AI_PROJECT_ID`. Key passed as `query_params`.
- **Service Account**: Set `WHODB_VERTEX_AI_CREDENTIALS`. `decodeCredentials()` handles raw JSON, base64, or file path. `project_id` inferred from credentials JSON if not explicitly set.

### AWS Bedrock

File: `ee/core/src/llm/providers/bedrock_provider.go`

| Variable | Required | Default | Description |
|---|---|---|---|
| `AWS_BEDROCK_MODELS` | Yes | | Comma-separated model IDs (falls back to `AWS_BEDROCK_MODEL`) |
| `AWS_REGION` | No | `us-east-1` | AWS region |
| `AWS_ACCESS_KEY_ID` | No | | Explicit credentials (optional, SDK chain used if not set) |
| `AWS_SECRET_ACCESS_KEY` | No | | Explicit credentials |
| `AWS_SESSION_TOKEN` | No | | For temporary credentials |
| `AWS_PROFILE` | No | | Named AWS profile from `~/.aws/credentials` |

Auth precedence: explicit credentials > named profile > SDK credential chain (instance profiles, task roles, etc.)

### EE Generic Providers

File: `ee/core/src/llm/providers/env_providers.go`

Same pattern as CE generic providers but with `WHODB_EE_AI_<ID>_*` prefix. `_MODEL` singular fallback also supported.

## Architecture Notes

- CE providers register via `llm.RegisterGenericProviders()` which calls both `providers.RegisterProvider()` (backend) and `env.AddGenericProvider()` (frontend display)
- EE providers register via `RegisterEEProvider()` → `initializeProvider()` which handles type-specific setup (dedicated `AIProvider` implementations for bedrock/azure/vertex, generic registration for others)
- `ProviderConfig.Metadata map[string]string` (core) and `EEProviderConfig.Metadata map[string]string` (EE) carry provider-specific config (api_version, region, credentials, etc.)
- BAML's `google-ai` client type rejects `request_timeout_ms` as a top-level option — it must be in an `http {}` block
