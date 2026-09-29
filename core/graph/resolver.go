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
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/source"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{}

// MapColumnsToModel converts source columns to GraphQL model columns.
func MapColumnsToModel(columnsResult []source.Column) []*model.Column {
	columns := make([]*model.Column, 0, len(columnsResult))
	for _, column := range columnsResult {
		columns = append(columns, &model.Column{
			Type:             column.Type,
			Name:             column.Name,
			MetadataFidelity: metadataFidelityToModel(column.MetadataFidelity),
			IsPrimary:        column.IsPrimary,
			IsForeignKey:     column.IsForeignKey,
			ReferencedTable:  column.ReferencedTable,
			ReferencedColumn: column.ReferencedColumn,
			Length:           column.Length,
			Precision:        column.Precision,
			Scale:            column.Scale,
		})
	}
	return columns
}

func metadataFidelityToModel(fidelity source.MetadataFidelity) model.SourceMetadataFidelity {
	return model.SourceMetadataFidelity(source.MetadataFidelityOrUnknown(fidelity))
}
