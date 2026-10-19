package config

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// WithPlatformRefreshLock serializes refresh-token rotation across CLI processes
// for the same hosted account. The lock file contains no credentials.
func (c *Config) WithPlatformRefreshLock(ctx context.Context, host, account string, refresh func() error) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "refresh-locks")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	name := fmt.Sprintf("%x.lock", sha256.Sum256([]byte(host+"\x00"+account)))
	// #nosec G304 -- the filename is a SHA-256 digest inside the config directory.
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		locked, err := tryRefreshLock(file)
		if err != nil {
			return err
		}
		if locked {
			defer unlockRefresh(file)
			return refresh()
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
