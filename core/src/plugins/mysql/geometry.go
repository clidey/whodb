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

package mysql

import (
	"encoding/binary"
	"fmt"

	"github.com/twpayne/go-geom/encoding/wkb"
	"github.com/twpayne/go-geom/encoding/wkt"
)

// FormatGeometryValue formats MySQL geometry data for display.
// MySQL stores geometry as WKB with a 4-byte SRID prefix.
func (p *MySQLPlugin) FormatGeometryValue(rawBytes []byte, columnType string) string {
	if len(rawBytes) == 0 {
		return ""
	}

	// MySQL geometry format: 4-byte SRID (little-endian) + WKB data
	if len(rawBytes) > 4 {
		srid := binary.LittleEndian.Uint32(rawBytes[:4])
		wkbData := rawBytes[4:]

		if geom, err := wkb.Unmarshal(wkbData); err == nil {
			if wktStr, err := wkt.Marshal(geom); err == nil {
				if srid != 0 {
					return fmt.Sprintf("%s (SRID:%d)", wktStr, srid)
				}
				return wktStr
			}
		}
	}

	// If WKB decoding fails, return as string
	return string(rawBytes)
}
