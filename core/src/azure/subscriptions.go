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
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"

	"github.com/clidey/whodb/core/src/log"
)

// Subscription represents an Azure subscription discovered via the ARM API.
type Subscription struct {
	ID          string
	DisplayName string
	State       string
	TenantID    string
}

// DiscoverSubscriptions lists Azure subscriptions accessible with the given credential.
// This is the Azure equivalent of AWS profile discovery — it helps the UI show
// available subscriptions to pick from.
func DiscoverSubscriptions(ctx context.Context, cred azcore.TokenCredential) ([]Subscription, error) {
	client, err := armsubscriptions.NewClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", HandleAzureError(err))
	}

	var subscriptions []Subscription
	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return subscriptions, HandleAzureError(err)
		}
		for _, sub := range page.Value {
			if sub == nil || sub.SubscriptionID == nil {
				continue
			}
			s := Subscription{
				ID:          *sub.SubscriptionID,
				DisplayName: derefString(sub.DisplayName),
				State:       string(*sub.State),
				TenantID:    derefString(sub.TenantID),
			}
			subscriptions = append(subscriptions, s)
		}
	}

	log.Debugf("Azure subscriptions: discovered %d subscriptions", len(subscriptions))
	return subscriptions, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
