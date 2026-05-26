//go:build !darwin

package common

import "fmt"

// selectDatabaseFileDarwin is a no-op on non-darwin platforms.
func (a *App) selectDatabaseFileDarwin(dbType string) (string, error) {
	return "", fmt.Errorf("darwin-specific file picker called on non-darwin platform")
}

// ResolveDatabaseBookmark is a no-op on non-darwin platforms.
func (a *App) ResolveDatabaseBookmark(path string) (string, error) {
	return "", fmt.Errorf("security-scoped bookmarks are only available on macOS")
}

// StopAccessingDatabase is a no-op on non-darwin platforms.
func (a *App) StopAccessingDatabase(path string) {}
