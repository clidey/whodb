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

package elasticsearch

import (
	"context"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
)

// Chat generates Elasticsearch queries using AI and executes read-only ones immediately.
// Mutations require user confirmation before execution.
func (p *ElasticSearchPlugin) Chat(config *engine.PluginConfig, _ string, previousConversation string, query string) ([]*engine.ChatMessage, error) {
	databases, err := p.GetDatabases(config)
	if err != nil {
		log.WithError(err).Error("Failed to list Elasticsearch indices for chat")
		return nil, err
	}

	tableDetails := strings.Builder{}
	for _, indexName := range databases {
		if strings.HasPrefix(indexName, ".") {
			continue
		}

		tableDetails.WriteString(fmt.Sprintf("index: %v\n", indexName))

		columns, err := p.GetColumnsForTable(config, "", indexName)
		if err != nil {
			log.WithError(err).WithField("index", indexName).Warn("Failed to get columns for chat context")
			continue
		}

		for _, col := range columns {
			tableDetails.WriteString(fmt.Sprintf("- %v (%v)\n", col.Name, col.Type))
		}
	}

	callCtx := context.Background()
	return common.DBChatBAML(callCtx, "Elasticsearch", "", tableDetails.String(), previousConversation, query, config, p)
}
