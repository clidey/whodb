/*
 * Copyright 2025 Clidey, Inc.
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

package postgres

import (
	"github.com/twpayne/go-geom/encoding/ewkb"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// FormatGeometryValue formats PostgreSQL geometry data for display.
// PostGIS columns use EWKB format which we decode to WKT.
// Native PostgreSQL geometric types (point, line, etc.) return text representation.
func (p *PostgresPlugin) FormatGeometryValue(rawBytes []byte, columnType string) string {
	if len(rawBytes) == 0 {
		return ""
	}

	// Try PostGIS EWKB decoding first
	if geom, err := ewkb.Unmarshal(rawBytes); err == nil {
		if wktStr, err := wkt.Marshal(geom); err == nil {
			return wktStr
		}
	}

	// Native PostgreSQL geometry types return text representation as bytes
	return string(rawBytes)
}
