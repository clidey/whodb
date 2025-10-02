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

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// File permissions
	filePermissionUserRW = 0644 // User read/write, group/others read
	dirPermissionUserRWX = 0755 // User read/write/execute, group/others read/execute
)

// App struct
type App struct {
	ctx            context.Context
	edition        string // "ce" or "ee"
	windowSettings WindowSettings
}

// WindowSettings stores window state
type WindowSettings struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximized bool `json:"maximized"`
}

// NewApp creates a new App application struct
func NewApp(edition string) *App {
	return &App{
		edition: edition,
	}
}

// Startup is called when the app starts
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.RestoreWindowState()
	a.SetupApplicationMenu()
	a.SetupSystemTray()
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	a.SaveWindowState()
}

// DomReady is called when the frontend is loaded
func (a *App) DomReady(ctx context.Context) {
	// Frontend is ready
}

// OpenURL opens a URL in the system's default browser
func (a *App) OpenURL(url string) error {
	runtime.BrowserOpenURL(a.ctx, url)
	return nil
}

// File Operations

// SaveFile shows native save dialog and saves data
func (a *App) SaveFile(data string, defaultName string) (string, error) {
	options := runtime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "Save File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "CSV Files (*.csv)",
				Pattern:     "*.csv",
			},
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
			{
				DisplayName: "SQL Files (*.sql)",
				Pattern:     "*.sql",
			},
			{
				DisplayName: "Excel Files (*.xlsx)",
				Pattern:     "*.xlsx",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	}

	filepath, err := runtime.SaveFileDialog(a.ctx, options)
	if err != nil {
		return "", err
	}

	if filepath == "" {
		return "", nil // User cancelled
	}

	// Write the data to file
	err = os.WriteFile(filepath, []byte(data), filePermissionUserRW)
	if err != nil {
		return "", err
	}

	return filepath, nil
}

// SaveBinaryFile saves binary data to a file
func (a *App) SaveBinaryFile(data []byte, defaultName string) (string, error) {
	options := runtime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Title:           "Save File",
	}

	filepath, err := runtime.SaveFileDialog(a.ctx, options)
	if err != nil {
		return "", err
	}

	if filepath == "" {
		return "", nil // User cancelled
	}

	err = os.WriteFile(filepath, data, filePermissionUserRW)
	if err != nil {
		return "", err
	}

	return filepath, nil
}

// SelectDirectory shows native directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	options := runtime.OpenDialogOptions{
		Title: "Select Directory",
	}

	dir, err := runtime.OpenDirectoryDialog(a.ctx, options)
	if err != nil {
		return "", err
	}

	return dir, nil
}

// SelectSQLiteDatabase shows native file dialog for selecting SQLite database files
func (a *App) SelectSQLiteDatabase() (string, error) {
	options := runtime.OpenDialogOptions{
		Title: "Select SQLite Database",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "SQLite Database Files (*.db,*.sqlite,*.sqlite3,*.db3)",
				Pattern:     "*.db;*.sqlite;*.sqlite3;*.db3",
			},
		},
	}

	filepath, err := runtime.OpenFileDialog(a.ctx, options)
	if err != nil {
		return "", err
	}

	if filepath == "" {
		return "", nil
	}

	// Validate file extension
	ext := strings.ToLower(filepath)
	dotIndex := strings.LastIndex(ext, ".")
	if dotIndex == -1 || dotIndex == len(ext)-1 {
		return "", fmt.Errorf("invalid file type: only .db, .sqlite, .sqlite3, and .db3 files are allowed")
	}
	ext = ext[dotIndex+1:]

	validExtensions := map[string]bool{
		"db":      true,
		"sqlite":  true,
		"sqlite3": true,
		"db3":     true,
	}

	if !validExtensions[ext] {
		return "", fmt.Errorf("invalid file type: only .db, .sqlite, .sqlite3, and .db3 files are allowed")
	}

	return filepath, nil
}

// Clipboard Operations

// CopyToClipboard copies text to system clipboard
func (a *App) CopyToClipboard(text string) error {
	return runtime.ClipboardSetText(a.ctx, text)
}

// GetFromClipboard gets text from system clipboard
func (a *App) GetFromClipboard() (string, error) {
	return runtime.ClipboardGetText(a.ctx)
}

// Window Management

// SaveWindowState saves current window position, size, and zoom
func (a *App) SaveWindowState() error {
	// Get current window state
	x, y := runtime.WindowGetPosition(a.ctx)
	width, height := runtime.WindowGetSize(a.ctx)
	maximized := runtime.WindowIsMaximised(a.ctx)

	a.windowSettings = WindowSettings{
		X:         x,
		Y:         y,
		Width:     width,
		Height:    height,
		Maximized: maximized,
	}

	// Save to local storage
	settingsPath := a.getSettingsPath()
	data, err := json.MarshalIndent(a.windowSettings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, filePermissionUserRW)
}

// RestoreWindowState restores window position, size, and zoom
func (a *App) RestoreWindowState() error {
	settingsPath := a.getSettingsPath()
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// File doesn't exist, use defaults
		return nil
	}

	err = json.Unmarshal(data, &a.windowSettings)
	if err != nil {
		return err
	}

	// Apply window settings
	runtime.WindowSetPosition(a.ctx, a.windowSettings.X, a.windowSettings.Y)
	runtime.WindowSetSize(a.ctx, a.windowSettings.Width, a.windowSettings.Height)
	if a.windowSettings.Maximized {
		runtime.WindowMaximise(a.ctx)
	}

	return nil
}

// MinimizeWindow minimizes the window
func (a *App) MinimizeWindow() {
	runtime.WindowMinimise(a.ctx)
}

// MaximizeWindow maximizes the window
func (a *App) MaximizeWindow() {
	runtime.WindowToggleMaximise(a.ctx)
}

// ShowAboutDialog shows an About dialog with app information
func (a *App) ShowAboutDialog() {
	version := "Community Edition"
	if a.edition == "ee" {
		version = "Enterprise Edition"
	}

	message := fmt.Sprintf(`WhoDB %s

The AI-First Database Management System

Build: Desktop Application
Platform: %s/%s

© 2025 Clidey, Inc.
All rights reserved.

Website: https://whodb.com
GitHub: https://github.com/clidey/whodb
Documentation: https://whodb.com/docs`,
		version,
		goruntime.GOOS,
		goruntime.GOARCH)

	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "About WhoDB",
		Message: message,
	})
}

// getSettingsPath returns the path to store window settings
func (a *App) getSettingsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to temp directory if home directory cannot be determined
		homeDir = os.TempDir()
	}
	suffix := ""
	if a.edition == "ee" {
		suffix = "-ee"
	}
	configDir := filepath.Join(homeDir, ".whodb"+suffix)
	os.MkdirAll(configDir, dirPermissionUserRWX)
	return filepath.Join(configDir, "window-settings.json")
}

// SetupApplicationMenu creates and sets the application menu
func (a *App) SetupApplicationMenu() {
	appMenu := menu.NewMenu()

	// File Menu
	fileMenu := appMenu.AddSubmenu("File")
	fileMenu.AddText("New Connection", keys.CmdOrCtrl("n"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggle-sidebar-new-connection")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Export Data", keys.CmdOrCtrl("e"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:export-data")
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("Quit", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		runtime.Quit(a.ctx)
	})

	// Edit Menu
	editMenu := appMenu.AddSubmenu("Edit")
	editMenu.AddText("Copy", keys.CmdOrCtrl("c"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:copy")
	})
	editMenu.AddText("Paste", keys.CmdOrCtrl("v"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:paste")
	})
	editMenu.AddText("Select All", keys.CmdOrCtrl("a"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:select-all")
	})
	editMenu.AddSeparator()
	editMenu.AddText("Find", keys.CmdOrCtrl("f"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:find")
	})

	// View Menu
	viewMenu := appMenu.AddSubmenu("View")
	viewMenu.AddText("Refresh", keys.CmdOrCtrl("r"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:refresh")
	})
	viewMenu.AddText("Toggle Sidebar", keys.CmdOrCtrl("b"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:toggle-sidebar")
	})

	// Database Menu
	dbMenu := appMenu.AddSubmenu("Database")
	dbMenu.AddText("Execute Query", keys.CmdOrCtrl("return"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:execute-query")
	})
	dbMenu.AddText("New Scratchpad Page", keys.CmdOrCtrl("t"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:new-scratchpad-page")
	})
	dbMenu.AddSeparator()
	dbMenu.AddText("Disconnect", keys.CmdOrCtrl("d"), func(_ *menu.CallbackData) {
		runtime.EventsEmit(a.ctx, "menu:disconnect")
	})

	// Window Menu
	windowMenu := appMenu.AddSubmenu("Window")
	windowMenu.AddText("Minimize", keys.CmdOrCtrl("m"), func(_ *menu.CallbackData) {
		a.MinimizeWindow()
	})
	windowMenu.AddText("Maximize", nil, func(_ *menu.CallbackData) {
		a.MaximizeWindow()
	})

	// Help Menu
	helpMenu := appMenu.AddSubmenu("Help")
	helpMenu.AddText("Documentation", nil, func(_ *menu.CallbackData) {
		a.OpenURL("https://whodb.com/docs")
	})
	helpMenu.AddText("Report Issue", nil, func(_ *menu.CallbackData) {
		a.OpenURL("https://github.com/clidey/whodb/issues")
	})
	helpMenu.AddSeparator()

	// Edition-specific about text
	aboutText := "About WhoDB"
	if a.edition == "ee" {
		aboutText = "About WhoDB Enterprise"
	}
	helpMenu.AddText(aboutText, nil, func(_ *menu.CallbackData) {
		a.ShowAboutDialog()
	})

	runtime.MenuSetApplicationMenu(a.ctx, appMenu)
}

// Dialog Operations

// ShowMessageDialog shows a native message dialog
func (a *App) ShowMessageDialog(title, message string, dialogType string) (string, error) {
	options := runtime.MessageDialogOptions{
		Title:   title,
		Message: message,
	}

	switch dialogType {
	case "info":
		options.Type = runtime.InfoDialog
	case "warning":
		options.Type = runtime.WarningDialog
	case "error":
		options.Type = runtime.ErrorDialog
	case "question":
		options.Type = runtime.QuestionDialog
		options.Buttons = []string{"Yes", "No"}
		options.DefaultButton = "Yes"
		options.CancelButton = "No"
	}

	result, err := runtime.MessageDialog(a.ctx, options)
	return result, err
}

// ShowConfirmDialog shows a confirmation dialog
func (a *App) ShowConfirmDialog(title, message string) (bool, error) {
	result, err := a.ShowMessageDialog(title, message, "question")
	return result == "Yes", err
}

// System Tray

// SetupSystemTray creates and configures the system tray
func (a *App) SetupSystemTray() {
	// Note: Wails v2 does not have built-in system tray support
	// This would need to be implemented with platform-specific code
	// or wait for Wails v3 which has tray support planned
	// For now, we'll skip the actual implementation
	// but keep the structure in place for future enhancement
}

// ShowNotification shows a desktop notification
func (a *App) ShowNotification(title, message string) {
	// This would use platform-specific notification APIs
	// For now, we can use message dialogs as a fallback
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Title:   title,
		Message: message,
		Type:    runtime.InfoDialog,
	})
}
