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

package mongodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// Chat generates MongoDB queries using AI and executes read-only ones immediately.
// Mutations require user confirmation before execution.
func (p *MongoDBPlugin) Chat(config *engine.PluginConfig, schema string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	client, err := DB(config)
	if err != nil {
		log.WithError(err).Error("Failed to connect to MongoDB for chat")
		return nil, err
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer disconnectClient(client)

	db := client.Database(schema)
	collections, err := db.ListCollectionNames(ctx, map[string]any{})
	if err != nil {
		log.WithError(err).Error("Failed to list MongoDB collections for chat")
		return nil, err
	}

	tableDetails := strings.Builder{}
	for _, collName := range collections {
		if strings.HasPrefix(collName, "system.") {
			continue
		}

		tableDetails.WriteString(fmt.Sprintf("collection: %v\n", collName))

		columns, err := p.GetColumnsForTable(config, schema, collName)
		if err != nil {
			log.WithError(err).WithField("collection", collName).Warn("Failed to get columns for chat context")
			continue
		}

		for _, col := range columns {
			tableDetails.WriteString(fmt.Sprintf("- %v (%v)\n", col.Name, col.Type))
		}
	}

	return common.ExecuteChatQuery(context.Background(), "MongoDB", schema, tableDetails.String(), previousConversation, query, config, p)
}
