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

package azure

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// LoadAzureCredential creates an Azure token credential from WhoDB credentials.
// Returns the credential and the subscription ID.
func LoadAzureCredential(creds *engine.Credentials) (azcore.TokenCredential, string, error) {
	azureCreds, err := ParseFromWhoDB(creds)
	if err != nil {
		return nil, "", err
	}

	credential, err := buildCredential(azureCreds)
	if err != nil {
		return nil, "", err
	}

	log.WithFields(map[string]any{
		"subscriptionID":   azureCreds.SubscriptionID,
		"authMethod":       azureCreds.AuthMethod,
		"hasResourceGroup": azureCreds.ResourceGroup != "",
	}).Debug("Azure credential loaded successfully")

	return credential, azureCreds.SubscriptionID, nil
}

// BuildCredentialFromConfig creates an Azure token credential from a parsed config.
func BuildCredentialFromConfig(config *AzureCredentialConfig) (azcore.TokenCredential, error) {
	return buildCredential(config)
}

func buildCredential(config *AzureCredentialConfig) (azcore.TokenCredential, error) {
	switch config.AuthMethod {
	case AuthMethodDefault:
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.WithError(err).Error("Failed to create default Azure credential")
			return nil, HandleAzureError(err)
		}
		return cred, nil

	case AuthMethodServicePrincipal:
		cred, err := azidentity.NewClientSecretCredential(
			config.TenantID,
			config.ClientID,
			config.ClientSecret,
			nil,
		)
		if err != nil {
			log.WithError(err).Error("Failed to create service principal credential")
			return nil, HandleAzureError(err)
		}
		return cred, nil

	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidAuthMethod, config.AuthMethod)
	}
}
