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

package layout

// LayoutName identifies a named layout preset.
type LayoutName string

const (
	LayoutSingle  LayoutName = "Single"
	LayoutExplore LayoutName = "Explore"
	LayoutQuery   LayoutName = "Query"
	LayoutFull    LayoutName = "Full"
)

// AllLayouts lists the available layout names in cycle order.
var AllLayouts = []LayoutName{LayoutSingle, LayoutExplore, LayoutQuery, LayoutFull}

// NextLayout returns the layout after the given one in the cycle.
func NextLayout(current LayoutName) LayoutName {
	for i, name := range AllLayouts {
		if name == current {
			return AllLayouts[(i+1)%len(AllLayouts)]
		}
	}
	return LayoutSingle
}

// AutoLayout picks the best layout based on terminal width.
func AutoLayout(width int) LayoutName {
	switch {
	case width >= 160:
		return LayoutFull
	case width >= 100:
		return LayoutExplore
	default:
		return LayoutSingle
	}
}

// Panes holds the named Renderables used to build layouts.
type Panes struct {
	Browser Renderable
	Editor  Renderable
	Results Renderable
}

// BuildLayout creates a container tree for the given layout name.
// The caller is responsible for calling Layout(x, y, w, h) on the result.
func BuildLayout(name LayoutName, p Panes) *Container {
	switch name {
	case LayoutExplore:
		// Browser (40%) | Results (60%) — horizontal split
		return NewSplit(Horizontal, 0.4,
			NewLeaf("Browser", p.Browser),
			NewLeaf("Results", p.Results),
		)

	case LayoutQuery:
		// Editor (50%) / Results (50%) — vertical split
		return NewSplit(Vertical, 0.5,
			NewLeaf("Editor", p.Editor),
			NewLeaf("Results", p.Results),
		)

	case LayoutFull:
		// Browser (30%) | Editor (50%) / Results (50%) — three pane
		return NewSplit(Horizontal, 0.3,
			NewLeaf("Browser", p.Browser),
			NewSplit(Vertical, 0.5,
				NewLeaf("Editor", p.Editor),
				NewLeaf("Results", p.Results),
			),
		)

	default:
		// Single pane — falls through to single below
	}

	return nil // Single layout is handled specially (no container tree)
}
