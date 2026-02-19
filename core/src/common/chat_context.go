//go:build !arm && !riscv64

package common

import (
	"context"
	"strings"

	"github.com/clidey/whodb/core/baml_client"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// ManageConversationContext compresses previousConversation when it exceeds
// model-proportional thresholds. Returns the (possibly compacted) conversation.
func ManageConversationContext(
	ctx context.Context,
	previousConversation string,
	externalModel *engine.ExternalModel,
	databaseType string,
) string {
	if previousConversation == "" {
		return ""
	}

	maxChars := estimateContextChars(externalModel)
	reserved := 8000 // reserved for schema + tool history + prompt
	available := maxChars - reserved
	if available < 4000 {
		available = 4000
	}

	compactThreshold := available / 2
	if len(previousConversation) < compactThreshold {
		return previousConversation
	}

	log.Logger.Infof("Agentic chat: compacting conversation (%d chars, threshold %d)", len(previousConversation), compactThreshold)

	callOpts := SetupAIClientWithLogging(externalModel)
	summary, err := baml_client.SummarizeConversation(ctx, previousConversation, databaseType, callOpts...)
	if err != nil {
		log.Logger.WithError(err).Warn("Agentic chat: summarization failed, hard-truncating")
		target := available * 15 / 100
		if target > len(previousConversation) {
			return previousConversation
		}
		return "[Earlier conversation truncated]\n" + previousConversation[len(previousConversation)-target:]
	}

	return "[Conversation summary]\n" + summary
}

// estimateContextChars returns an estimated context window size in characters
// for the given model configuration.
func estimateContextChars(m *engine.ExternalModel) int {
	if m == nil {
		return 64000
	}

	model := strings.ToLower(m.Model)

	switch m.Type {
	case "Ollama":
		// Most Ollama models default to 32K context
		if strings.Contains(model, "llama-3") || strings.Contains(model, "llama3") {
			return 128000
		}
		return 32000

	case "OpenAI", "ChatGPT":
		if strings.Contains(model, "gpt-4o") || strings.Contains(model, "gpt-4-turbo") {
			return 500000
		}
		if strings.Contains(model, "gpt-4") {
			return 32000
		}
		if strings.Contains(model, "o1") || strings.Contains(model, "o3") || strings.Contains(model, "o4") {
			return 800000
		}
		return 128000

	case "Anthropic":
		return 800000

	default:
		// OpenAI-Compatible and unknown providers
		if strings.Contains(model, "gemini") {
			return 4000000
		}
		if strings.Contains(model, "claude") {
			return 800000
		}
		return 64000
	}
}
