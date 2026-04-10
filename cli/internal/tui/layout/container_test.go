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

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// mockPane implements Renderable for testing.
type mockPane struct {
	name   string
	width  int
	height int
}

func (m *mockPane) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}
	lines := make([]string, m.height)
	label := m.name
	if len(label) > m.width {
		label = label[:m.width]
	}
	lines[0] = label + strings.Repeat(" ", m.width-len(label))
	for i := 1; i < m.height; i++ {
		lines[i] = strings.Repeat(" ", m.width)
	}
	return strings.Join(lines, "\n")
}

func (m *mockPane) SetDimensions(w, h int) {
	m.width = w
	m.height = h
}

func TestNewLeaf(t *testing.T) {
	p := &mockPane{name: "test"}
	c := NewLeaf("Test", p)

	if !c.IsLeaf() {
		t.Error("NewLeaf should create a leaf container")
	}
	if c.Label() != "Test" {
		t.Errorf("Label() = %q, want %q", c.Label(), "Test")
	}
	if c.Content() != p {
		t.Error("Content() should return the provided Renderable")
	}
}

func TestNewSplit(t *testing.T) {
	left := NewLeaf("L", &mockPane{name: "left"})
	right := NewLeaf("R", &mockPane{name: "right"})
	c := NewSplit(Horizontal, 0.4, left, right)

	if c.IsLeaf() {
		t.Error("NewSplit should not be a leaf")
	}
	if c.Direction() != Horizontal {
		t.Error("Direction should be Horizontal")
	}
	if c.Ratio() != 0.4 {
		t.Errorf("Ratio() = %f, want 0.4", c.Ratio())
	}
	if c.Children()[0] != left || c.Children()[1] != right {
		t.Error("Children should match provided containers")
	}
}

func TestRatioClamp(t *testing.T) {
	left := NewLeaf("L", &mockPane{name: "l"})
	right := NewLeaf("R", &mockPane{name: "r"})

	c := NewSplit(Horizontal, -0.5, left, right)
	if c.Ratio() != 0.1 {
		t.Errorf("Ratio should be clamped to 0.1, got %f", c.Ratio())
	}

	c.SetRatio(1.5)
	if c.Ratio() != 0.9 {
		t.Errorf("Ratio should be clamped to 0.9, got %f", c.Ratio())
	}
}

func TestLayoutLeaf(t *testing.T) {
	p := &mockPane{name: "test"}
	c := NewLeaf("Test", p)

	c.Layout(0, 0, 80, 24)

	x, y, w, h := c.Geometry()
	if x != 0 || y != 0 || w != 80 || h != 24 {
		t.Errorf("Geometry = (%d,%d,%d,%d), want (0,0,80,24)", x, y, w, h)
	}
	// Content gets full dimensions (SetDimensions called by Layout)
	if p.width != 80 || p.height != 24 {
		t.Errorf("Content dimensions = (%d,%d), want (80,24)", p.width, p.height)
	}
}

func TestLayoutHorizontalSplit(t *testing.T) {
	left := &mockPane{name: "left"}
	right := &mockPane{name: "right"}
	c := NewSplit(Horizontal, 0.5,
		NewLeaf("Left", left),
		NewLeaf("Right", right),
	)

	c.Layout(0, 0, 101, 24) // 101 = 50 + 1 divider + 50

	lx, _, lw, lh := c.Children()[0].Geometry()
	rx, _, rw, rh := c.Children()[1].Geometry()

	if lx != 0 {
		t.Errorf("Left x = %d, want 0", lx)
	}
	if lw+1+rw != 101 {
		t.Errorf("Left(%d) + divider(1) + Right(%d) = %d, want 101", lw, rw, lw+1+rw)
	}
	if rx != lw+1 {
		t.Errorf("Right x = %d, want %d", rx, lw+1)
	}
	if lh != 24 || rh != 24 {
		t.Errorf("Heights should both be 24, got left=%d right=%d", lh, rh)
	}
}

func TestLayoutVerticalSplit(t *testing.T) {
	top := &mockPane{name: "top"}
	bottom := &mockPane{name: "bottom"}
	c := NewSplit(Vertical, 0.5,
		NewLeaf("Top", top),
		NewLeaf("Bottom", bottom),
	)

	c.Layout(0, 0, 80, 25) // 25 = 12 + 1 divider + 12

	_, ty, _, th := c.Children()[0].Geometry()
	_, by, _, bh := c.Children()[1].Geometry()

	if ty != 0 {
		t.Errorf("Top y = %d, want 0", ty)
	}
	if th+1+bh != 25 {
		t.Errorf("Top(%d) + divider(1) + Bottom(%d) = %d, want 25", th, bh, th+1+bh)
	}
	if by != th+1 {
		t.Errorf("Bottom y = %d, want %d", by, th+1)
	}
}

func TestLayoutThreePane(t *testing.T) {
	browser := &mockPane{name: "browser"}
	editor := &mockPane{name: "editor"}
	results := &mockPane{name: "results"}

	c := BuildLayout(LayoutFull, Panes{
		Browser: browser,
		Editor:  editor,
		Results: results,
	})

	c.Layout(0, 0, 160, 40)

	leaves := c.Leaves()
	if len(leaves) != 3 {
		t.Fatalf("Expected 3 leaves, got %d", len(leaves))
	}

	// All leaves should have positive dimensions
	for i, leaf := range leaves {
		_, _, w, h := leaf.Geometry()
		if w <= 0 || h <= 0 {
			t.Errorf("Leaf %d (%s) has zero dimensions: %dx%d", i, leaf.Label(), w, h)
		}
	}
}

func TestLayoutTooSmallFallback(t *testing.T) {
	left := &mockPane{name: "left"}
	right := &mockPane{name: "right"}
	c := NewSplit(Horizontal, 0.5,
		NewLeaf("Left", left),
		NewLeaf("Right", right),
	)

	// Too narrow for two panes (need MinPaneWidth*2 + 1)
	c.Layout(0, 0, 30, 24) // 30 < 20*2+1 = 41

	// First child gets all the space
	_, _, lw, _ := c.Children()[0].Geometry()
	_, _, rw, _ := c.Children()[1].Geometry()

	if lw != 30 {
		t.Errorf("Left should get full width 30, got %d", lw)
	}
	if rw != 0 {
		t.Errorf("Right should get 0 width, got %d", rw)
	}
}

func TestViewRenders(t *testing.T) {
	left := &mockPane{name: "L"}
	right := &mockPane{name: "R"}
	c := NewSplit(Horizontal, 0.5,
		NewLeaf("Left", left),
		NewLeaf("Right", right),
	)

	c.Layout(0, 0, 101, 10)
	output := c.View()

	if output == "" {
		t.Error("View() should produce output")
	}

	lines := strings.Split(output, "\n")
	if len(lines) != 10 {
		t.Errorf("Expected 10 lines, got %d", len(lines))
	}

	// First line should contain both pane headers
	firstLine := lines[0]
	if !strings.Contains(firstLine, "Left") {
		t.Error("First line should contain 'Left' label")
	}
	if !strings.Contains(firstLine, "Right") {
		t.Error("First line should contain 'Right' label")
	}
}

func TestViewSingleNoHeader(t *testing.T) {
	p := &mockPane{name: "content"}
	c := NewLeaf("Test", p)

	c.Layout(0, 0, 40, 10)
	output := c.ViewSingle()

	// ViewSingle should NOT have the header line
	if strings.Contains(output, "── Test ") {
		t.Error("ViewSingle should not include a pane header")
	}
}

func TestLeaves(t *testing.T) {
	c := NewSplit(Horizontal, 0.3,
		NewLeaf("Browser", &mockPane{name: "b"}),
		NewSplit(Vertical, 0.5,
			NewLeaf("Editor", &mockPane{name: "e"}),
			NewLeaf("Results", &mockPane{name: "r"}),
		),
	)

	leaves := c.Leaves()
	if len(leaves) != 3 {
		t.Fatalf("Expected 3 leaves, got %d", len(leaves))
	}
	names := []string{leaves[0].Label(), leaves[1].Label(), leaves[2].Label()}
	expected := []string{"Browser", "Editor", "Results"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("Leaf %d label = %q, want %q", i, name, expected[i])
		}
	}
}

func TestFitToBox(t *testing.T) {
	// Short content
	result := fitToBox("abc", 5, 3)
	lines := strings.Split(result, "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}
	if lipgloss.Width(lines[0]) != 5 {
		t.Errorf("Line 0 width = %d, want 5", lipgloss.Width(lines[0]))
	}

	// Long content
	result = fitToBox("abcdefghij\nline2\nline3\nline4", 5, 2)
	lines = strings.Split(result, "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestAutoLayout(t *testing.T) {
	tests := []struct {
		width    int
		expected LayoutName
	}{
		{60, LayoutSingle},
		{99, LayoutSingle},
		{100, LayoutExplore},
		{159, LayoutExplore},
		{160, LayoutFull},
		{200, LayoutFull},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("width=%d", tt.width), func(t *testing.T) {
			got := AutoLayout(tt.width)
			if got != tt.expected {
				t.Errorf("AutoLayout(%d) = %q, want %q", tt.width, got, tt.expected)
			}
		})
	}
}

func TestNextLayout(t *testing.T) {
	if NextLayout(LayoutSingle) != LayoutExplore {
		t.Error("After Single should be Explore")
	}
	if NextLayout(LayoutFull) != LayoutSingle {
		t.Error("After Full should wrap to Single")
	}
}
