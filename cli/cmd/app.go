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

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

const appDownloadURL = "https://whodb.com/"

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Open the WhoDB desktop app",
	Long: `Open the WhoDB desktop app if it is installed on this machine.

The desktop app is a separate installation from this CLI. If it is not
found, this command opens the WhoDB download page instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := launchDesktopApp(); err == nil {
			return nil
		}
		fmt.Println("WhoDB desktop app not found. Opening the download page:")
		fmt.Println("  " + appDownloadURL)
		if err := openURL(appDownloadURL); err != nil {
			return fmt.Errorf("could not open browser; visit %s to download the desktop app", appDownloadURL)
		}
		return nil
	},
}

// launchDesktopApp starts the locally installed WhoDB desktop app, returning
// an error if no installation is found for the current platform.
func launchDesktopApp() error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-a", "WhoDB").Run()
	case "windows":
		for _, dir := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles")} {
			if dir == "" {
				continue
			}
			path := filepath.Join(dir, "WhoDB", "whodb.exe")
			if _, err := os.Stat(path); err == nil {
				return exec.Command(path).Start()
			}
		}
		return fmt.Errorf("desktop app not installed")
	default:
		// The Linux desktop app is distributed as a snap named "whodb".
		if _, err := os.Stat("/snap/bin/whodb"); err == nil {
			return exec.Command("/snap/bin/whodb").Start()
		}
		return fmt.Errorf("desktop app not installed")
	}
}

// openURL opens target in the default browser for the current platform.
func openURL(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	return cmd.Start()
}

func init() {
	rootCmd.AddCommand(appCmd)
}
