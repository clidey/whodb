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

package tui

import (
	"context"
	"time"

	"github.com/clidey/whodb/cli/pkg/analytics"
	coreanalytics "github.com/clidey/whodb/core/src/analytics"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sqlguard"
)

func (m *MainModel) executeTrackedQuery(ctx context.Context, query, origin string) (*engine.GetRowsResult, error) {
	dbType := ""
	if connection := m.dbManager.GetCurrentConnection(); connection != nil {
		dbType = connection.Type
	}
	start := time.Now()
	result, err := m.dbManager.ExecuteQueryWithContext(ctx, query)
	statementType := string(sqlguard.Classify(query).Type)
	properties := map[string]any{
		"execution_mode": "tui",
		"origin_type":    origin,
		"statement_type": statementType,
	}
	if err != nil {
		analytics.TrackQueryError(ctx, dbType, coreanalytics.ErrorCode(err), time.Since(start).Milliseconds(), properties)
	} else {
		properties["column_count"] = len(result.Columns)
		analytics.TrackQueryExecute(ctx, dbType, statementType, true, time.Since(start).Milliseconds(), len(result.Rows), properties)
	}
	return result, err
}
