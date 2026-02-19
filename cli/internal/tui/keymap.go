// Copyright 2024 Clidey
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/clidey/whodb/cli/pkg/styles"
)

// GlobalKeys contains keybindings used across multiple views
type GlobalKeys struct {
	Quit     key.Binding
	Help     key.Binding
	NextView key.Binding
	Back     key.Binding
}

// BrowserKeys contains keybindings for the browser view
type BrowserKeys struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	Select     key.Binding
	Filter     key.Binding
	Schema     key.Binding
	Refresh    key.Binding
	Editor     key.Binding
	AIChat     key.Binding
	History    key.Binding
	Disconnect key.Binding
}

// EditorKeys contains keybindings for the editor view
type EditorKeys struct {
	Execute      key.Binding
	Autocomplete key.Binding
	Clear        key.Binding
	Export       key.Binding
}

// ResultsKeys contains keybindings for the results view
type ResultsKeys struct {
	Up         key.Binding
	Down       key.Binding
	ColLeft    key.Binding
	ColRight   key.Binding
	NextPage   key.Binding
	PrevPage   key.Binding
	Where      key.Binding
	Columns    key.Binding
	Export     key.Binding
	PageSize   key.Binding
	CustomSize key.Binding
}

// HistoryKeys contains keybindings for the history view
type HistoryKeys struct {
	Edit     key.Binding
	Rerun    key.Binding
	ClearAll key.Binding
}

// ChatKeys contains keybindings for the AI chat view
type ChatKeys struct {
	CycleFieldUp   key.Binding
	CycleFieldDown key.Binding
	ChangeLeft     key.Binding
	ChangeRight    key.Binding
	Send           key.Binding
	SelectPrevMsg  key.Binding
	SelectNextMsg  key.Binding
	FocusInput     key.Binding
	RevokeConsent  key.Binding
	LoadModels     key.Binding
}

// SchemaKeys contains keybindings for the schema view
type SchemaKeys struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	ViewData key.Binding
	Filter   key.Binding
	Refresh  key.Binding
}

// ColumnsKeys contains keybindings for the columns view
type ColumnsKeys struct {
	Up         key.Binding
	Down       key.Binding
	Toggle     key.Binding
	SelectAll  key.Binding
	SelectNone key.Binding
	Apply      key.Binding
}

// WhereListKeys contains keybindings for the where view in list mode
type WhereListKeys struct {
	Up       key.Binding
	Down     key.Binding
	Add      key.Binding
	EditCond key.Binding
	Delete   key.Binding
	Apply    key.Binding
}

// WhereAddKeys contains keybindings for the where view in add mode
type WhereAddKeys struct {
	Prev    key.Binding
	Next    key.Binding
	Change  key.Binding
	Confirm key.Binding
}

// ExportKeys contains keybindings for the export view
type ExportKeys struct {
	Prev        key.Binding
	Next        key.Binding
	OptionLeft  key.Binding
	OptionRight key.Binding
	Export      key.Binding
}

// ConnectionListKeys contains keybindings for connection view in list mode
type ConnectionListKeys struct {
	Up         key.Binding
	Down       key.Binding
	Connect    key.Binding
	New        key.Binding
	DeleteConn key.Binding
	QuitEsc    key.Binding
}

// ConnectionFormKeys contains keybindings for connection view in form mode
type ConnectionFormKeys struct {
	Navigate    key.Binding
	TypeLeft    key.Binding
	TypeRight   key.Binding
	ConnectForm key.Binding
}

// FilterKeys contains keybindings for filter mode (shared by browser, schema)
type FilterKeys struct {
	CancelFilter key.Binding
	ApplyFilter  key.Binding
}

// SchemaSelectKeys contains keybindings for schema selection (browser)
type SchemaSelectKeys struct {
	NavLeft      key.Binding
	NavRight     key.Binding
	SelectSchema key.Binding
}

// Keymap contains all keybinding groups
type Keymap struct {
	Global         GlobalKeys
	Browser        BrowserKeys
	Editor         EditorKeys
	Results        ResultsKeys
	History        HistoryKeys
	Chat           ChatKeys
	Schema         SchemaKeys
	Columns        ColumnsKeys
	WhereList      WhereListKeys
	WhereAdd       WhereAddKeys
	Export         ExportKeys
	ConnectionList ConnectionListKeys
	ConnectionForm ConnectionFormKeys
	Filter         FilterKeys
	SchemaSelect   SchemaSelectKeys
}

// Keys is the top-level keymap containing all keybindings
var Keys = Keymap{
	Global: GlobalKeys{
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		NextView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	},
	Browser: BrowserKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view data"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/", "f"),
			key.WithHelp("[/]", "filter"),
		),
		Schema: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "schema"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "refresh"),
		),
		Editor: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "editor"),
		),
		AIChat: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "ai chat"),
		),
		History: key.NewBinding(
			key.WithKeys("ctrl+h"),
			key.WithHelp("ctrl+h", "history"),
		),
		Disconnect: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "disconnect"),
		),
	},
	Editor: EditorKeys{
		Execute: key.NewBinding(
			key.WithKeys("alt+enter"),
			key.WithHelp(styles.KeyExecute, "run query"),
		),
		Autocomplete: key.NewBinding(
			key.WithKeys("ctrl+@"),
			key.WithHelp("ctrl+space", "autocomplete"),
		),
		Clear: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "clear"),
		),
		Export: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("[e]", "export results"),
		),
	},
	Results: ResultsKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		ColLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "col left"),
		),
		ColRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "col right"),
		),
		NextPage: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n/p", "page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("n/p", "page"),
		),
		Where: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "where"),
		),
		Columns: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "columns"),
		),
		Export: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export"),
		),
		PageSize: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "page size"),
		),
		CustomSize: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("shift+s", "custom size"),
		),
	},
	History: HistoryKeys{
		Edit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "edit"),
		),
		Rerun: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "re-run"),
		),
		ClearAll: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("shift+d", "clear all"),
		),
	},
	Chat: ChatKeys{
		CycleFieldUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑/↓", "cycle fields"),
		),
		CycleFieldDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↑/↓", "cycle fields"),
		),
		ChangeLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←/→", "change selection"),
		),
		ChangeRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("←/→", "change selection"),
		),
		Send: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm/send/view"),
		),
		SelectPrevMsg: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("ctrl+p/n", "select message"),
		),
		SelectNextMsg: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+p/n", "select message"),
		),
		FocusInput: key.NewBinding(
			key.WithKeys("ctrl+i", "/"),
			key.WithHelp("ctrl+i", "focus input"),
		),
		RevokeConsent: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("ctrl+r", "revoke consent"),
		),
		LoadModels: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "load models"),
		),
	},
	Schema: SchemaKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "expand"),
		),
		ViewData: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("[v]", "view data"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/", "f"),
			key.WithHelp("[/]", "filter"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("[r]", "refresh"),
		),
	},
	Columns: ColumnsKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "prev"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "x"),
			key.WithHelp("space", "toggle"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("[a]", "all"),
		),
		SelectNone: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("[n]", "none"),
		),
		Apply: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply"),
		),
	},
	WhereList: WhereListKeys{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next"),
		),
		Add: key.NewBinding(
			key.WithKeys("ctrl+a"),
			key.WithHelp("ctrl+a", "add new"),
		),
		EditCond: key.NewBinding(
			key.WithKeys("ctrl+e"),
			key.WithHelp("ctrl+e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete"),
		),
		Apply: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply"),
		),
	},
	WhereAdd: WhereAddKeys{
		Prev: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "prev"),
		),
		Next: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "next"),
		),
		Change: key.NewBinding(
			key.WithKeys("left", "right"),
			key.WithHelp("← →", "change"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "add"),
		),
	},
	Export: ExportKeys{
		Prev: key.NewBinding(
			key.WithKeys("up", "k", "shift+tab"),
			key.WithHelp("↑/k", "prev"),
		),
		Next: key.NewBinding(
			key.WithKeys("down", "j", "tab"),
			key.WithHelp("↓/j", "next"),
		),
		OptionLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "prev option"),
		),
		OptionRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next option"),
		),
		Export: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	},
	ConnectionList: ConnectionListKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k", "shift+tab"),
			key.WithHelp("↑/k/shift+tab", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j", "tab"),
			key.WithHelp("↓/j/tab", "down"),
		),
		Connect: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
		New: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("[n]", "new"),
		),
		DeleteConn: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("[d]", "delete"),
		),
		QuitEsc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
	},
	ConnectionForm: ConnectionFormKeys{
		Navigate: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("↑/↓/tab", "navigate"),
		),
		TypeLeft: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←/→", "change type"),
		),
		TypeRight: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("←/→", "change type"),
		),
		ConnectForm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "connect"),
		),
	},
	Filter: FilterKeys{
		CancelFilter: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel filter"),
		),
		ApplyFilter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "apply filter"),
		),
	},
	SchemaSelect: SchemaSelectKeys{
		NavLeft: key.NewBinding(
			key.WithKeys("left", "up", "h", "k"),
		),
		NavRight: key.NewBinding(
			key.WithKeys("right", "down", "l", "j"),
		),
		SelectSchema: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select schema"),
		),
	},
}

// RenderBindingHelp renders help text for the given key bindings
func RenderBindingHelp(bindings ...key.Binding) string {
	var pairs []string
	for _, b := range bindings {
		h := b.Help()
		pairs = append(pairs, h.Key, h.Desc)
	}
	return styles.RenderHelp(pairs...)
}
