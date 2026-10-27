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

package graph

import (
	"context"
	"strings"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/analytics"
)

func trackSourceQuery(ctx context.Context, start time.Time, event, sourceType string, properties map[string]any, result *model.RowsResult, err error) {
	identity := strings.TrimSpace(analytics.MetadataFromContext(ctx).DistinctID)
	if identity == "" || identity == "disabled" {
		return
	}

	duration := time.Since(start)
	properties["success"] = err == nil
	properties["outcome"] = "success"
	properties["duration_ms"] = duration.Milliseconds()
	properties["duration_bucket"] = analytics.DurationBucket(duration)
	properties["slow"] = duration >= 3*time.Second
	if sourceType != "" {
		properties["source_type"] = sourceType
	}
	if err != nil {
		properties["outcome"] = "failure"
		properties["error_code"] = analytics.ErrorCode(err)
	}
	if result != nil {
		properties["row_count_bucket"] = analytics.CountBucket(int64(len(result.Rows)))
		properties["column_count_bucket"] = analytics.CountBucket(int64(len(result.Columns)))
		if result.TotalCount >= 0 {
			properties["total_row_count_bucket"] = analytics.CountBucket(int64(result.TotalCount))
		}
		properties["empty_result"] = len(result.Rows) == 0
	}
	analytics.CaptureWithDistinctID(ctx, identity, event, properties)
}
