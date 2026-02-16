//go:build arm || riscv64

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
	ctx "context"
	"fmt"

	"github.com/clidey/whodb/core/graph/model"
)

// generateChatTitleImpl is not supported on ARM/RISCV64 architectures
func generateChatTitleImpl(c ctx.Context, input model.GenerateChatTitleInput) (*model.GenerateChatTitleResponse, error) {
	return nil, fmt.Errorf("chat title generation not supported on this platform")
}
