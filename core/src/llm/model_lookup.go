package llm

import (
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/source"
)

// ModelLookupOptions controls supported-model discovery for a configured or user-supplied provider.
type ModelLookupOptions struct {
	ProviderID          string
	ModelType           string
	Token               string
	ConfiguredProviders []env.ChatProvider
	AllowTokenFallback  bool
}

// ListSupportedModels returns configured generic models or discovers models from the selected provider.
func ListSupportedModels(options ModelLookupOptions) ([]string, error) {
	model, configuredModels := resolveModelLookup(options)
	if configuredModels != nil {
		return configuredModels, nil
	}
	return ClientForModel(&model).GetSupportedModels()
}

func resolveModelLookup(options ModelLookupOptions) (source.ExternalModel, []string) {
	model := source.ExternalModel{Type: options.ModelType}
	if options.ProviderID == "" {
		model.Token = options.Token
		return model, nil
	}

	for _, provider := range options.ConfiguredProviders {
		if provider.ProviderId != options.ProviderID {
			continue
		}
		model.Token = provider.APIKey
		if provider.IsGeneric {
			for _, genericProvider := range env.GenericProviders {
				if genericProvider.ProviderId == options.ProviderID {
					return model, genericProvider.Models
				}
			}
		}
		return model, nil
	}

	if options.AllowTokenFallback {
		model.Token = options.Token
	}
	return model, nil
}
