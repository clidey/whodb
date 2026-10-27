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
	"testing"
)

func TestTrackedTUIQueryPreservesResultsAndErrors(t *testing.T) {
	model := setupConnectedModelForTest(t)
	t.Cleanup(func() { model.dbManager.Disconnect() })
	result, err := model.executeTrackedQuery(context.Background(), "SELECT 42 AS answer", "editor")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || len(result.Rows[0]) != 1 || result.Rows[0][0] != "42" {
		t.Fatalf("tracked query changed the result: %#v", result)
	}
	if _, err := model.executeTrackedQuery(context.Background(), "SELECT missing_column FROM users", "history"); err == nil {
		t.Fatal("tracked query swallowed the database error")
	}
}
