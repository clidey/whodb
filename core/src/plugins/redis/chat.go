//go:build !arm && !riscv64

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

package redis

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// Chat generates Redis commands using AI and executes read-only ones immediately.
// Mutations require user confirmation before execution.
func (p *RedisPlugin) Chat(config *engine.PluginConfig, schema string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	storageUnits, err := p.GetStorageUnits(config, schema)
	if err != nil {
		log.WithError(err).Error("Failed to get Redis storage units for chat")
		return nil, err
	}

	tableDetails := strings.Builder{}
	for _, su := range storageUnits {
		keyType := ""
		for _, attr := range su.Attributes {
			if attr.Key == "Type" {
				keyType = attr.Value
				break
			}
		}

		tableDetails.WriteString(fmt.Sprintf("key: %v (type: %v)\n", su.Name, keyType))

		columns, err := p.GetColumnsForTable(config, schema, su.Name)
		if err != nil {
			continue
		}

		for _, col := range columns {
			tableDetails.WriteString(fmt.Sprintf("- %v\n", col.Name))
		}
	}

	return common.ExecuteChatQuery(context.Background(), "Redis", schema, tableDetails.String(), previousConversation, query, config, p)
}
