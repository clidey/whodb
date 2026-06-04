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
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/clidey/whodb/core/src/log"
)

// Azure AD token scopes for database authentication.
const (
	// ScopePostgreSQLMySQL is the OAuth2 scope for Azure Database for PostgreSQL and MySQL.
	ScopePostgreSQLMySQL = "https://ossrdbms-aad.database.windows.net/.default"

	// ScopeRedis is the OAuth2 scope for Azure Cache for Redis.
	ScopeRedis = "https://redis.azure.com/.default"
)

// GenerateADToken obtains a short-lived Azure AD access token for database authentication.
// The token acts as a password — the username is the Azure AD identity.
func GenerateADToken(ctx context.Context, cred azcore.TokenCredential, scope string) (string, error) {
	log.Infof("Azure AD Auth: generating token for scope %s", scope)

	token, err := cred.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scope},
	})
	if err != nil {
		log.Errorf("Azure AD Auth: token generation failed for scope %s: %v", scope, err)
		return "", HandleAzureError(err)
	}

	log.Infof("Azure AD Auth: token generated successfully (token length=%d)", len(token.Token))
	return token.Token, nil
}
