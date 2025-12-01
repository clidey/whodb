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

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

const rcBlockStart = "# >>> whodb-cli completion >>>"
const rcBlockEnd = "# <<< whodb-cli completion <<<"

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate autocompletion scripts for supported shells.

Examples:
  # Install completion for your current shell (recommended)
  whodb-cli completion install

  # Print completion script for bash to stdout
  whodb-cli completion bash|zsh|fish|powershell

  # Install completion for a specific shell
  whodb-cli completion install bash`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default behavior with no args: show help instead of printing a script
		if len(args) == 0 {
			return cmd.Help()
		}

		sh := strings.ToLower(strings.TrimSpace(args[0]))

		var buf bytes.Buffer
		if err := generateCompletion(sh, &buf); err != nil {
			return err
		}
		_, _ = io.Copy(os.Stdout, &buf)
		return nil
	},
}

var completionInstallCmd = &cobra.Command{
	Use:   "install [bash|zsh|fish|powershell]",
	Short: "Install shell completion script",
	Long: `Install autocompletion for your shell into standard user directories.

This writes the completion script to a common per-user location without requiring sudo.
For bash and zsh, ensure your shell loads completions from these locations as noted below.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := ""
		if len(args) == 1 {
			sh = strings.ToLower(strings.TrimSpace(args[0]))
		} else {
			sh = detectShell()
		}
		if sh == "" {
			return errors.New("could not detect shell; please specify one: bash|zsh|fish|powershell")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		var targetPath string
		var postMsg string

		switch sh {
		case "bash":
			// Standard user dir for bash-completion if installed
			targetPath = filepath.Join(home, ".local", "share", "bash-completion", "completions", "whodb-cli")
			postMsg = "Bash completion installed. Ensure bash-completion is installed and sourced. Restart your shell, or source ~/.bashrc."

		case "zsh":
			// A common per-user completions directory; user must add to fpath
			targetPath = filepath.Join(home, ".zsh", "completions", "_whodb-cli")
			postMsg = "Zsh completion installed. Add 'fpath=(~/.zsh/completions $fpath); autoload -U compinit; compinit' to your ~/.zshrc if not already configured. Then restart zsh."

		case "fish":
			// Fish automatically loads from this path
			targetPath = filepath.Join(home, ".config", "fish", "completions", "whodb-cli.fish")
			postMsg = "Fish completion installed. Restart fish or run 'exec fish'."

		case "powershell", "pwsh":
			// Installing PowerShell completions varies; for now, print instructions
			return errors.New("powershell completion install is not automated yet; run 'whodb-cli completion powershell | Out-String | Set-Content -Path $PROFILE' in PowerShell")

		default:
			return fmt.Errorf("unsupported shell: %s", sh)
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed creating directory %s: %w", filepath.Dir(targetPath), err)
		}

		var buf bytes.Buffer
		if err := generateCompletion(sh, &buf); err != nil {
			return err
		}

		if err := os.WriteFile(targetPath, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("failed writing completion to %s: %w", targetPath, err)
		}

		fmt.Fprintf(os.Stdout, "Installed %s completion to: %s\n", sh, targetPath)

		// Append shell init configuration automatically
		switch sh {
		case "bash":
			if err := ensureBashRc(targetPath); err != nil {
				return fmt.Errorf("failed updating bash rc: %w", err)
			}
		case "zsh":
			if err := ensureZshRc(); err != nil {
				return fmt.Errorf("failed updating zsh rc: %w", err)
			}
		case "fish":
			// fish should autoload from the completions directory - will test later
		}

		fmt.Fprintln(os.Stdout, postMsg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
	completionCmd.AddCommand(completionInstallCmd)
	completionCmd.AddCommand(completionUninstallCmd)
}

func detectShell() string {
	// Try SHELL env (Unix)
	if sh := os.Getenv("SHELL"); sh != "" {
		base := filepath.Base(sh)
		base = strings.ToLower(base)
		switch base {
		case "bash", "zsh", "fish":
			return base
		}
	}

	// Try PowerShell on Windows
	if runtime.GOOS == "windows" {
		if ps := os.Getenv("PSModulePath"); ps != "" {
			return "powershell"
		}
	}
	return ""
}

func generateCompletion(shell string, w io.Writer) error {
	switch shell {
	case "bash":
		return rootCmd.GenBashCompletion(w)
	case "zsh":
		return rootCmd.GenZshCompletion(w)
	case "fish":
		return rootCmd.GenFishCompletion(w, true)
	case "powershell", "pwsh":
		return rootCmd.GenPowerShellCompletion(w)
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func ensureBashRc(installPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	block := rcBlockStart + "\n" +
		"# shell completion\n" +
		"if [ -f '" + installPath + "' ]; then\n" +
		". '" + installPath + "'\n" +
		"fi\n" +
		rcBlockEnd + "\n"

	var rcFiles []string
	if runtime.GOOS == "darwin" {
		rcFiles = []string{filepath.Join(home, ".bash_profile"), filepath.Join(home, ".bashrc")}
	} else {
		rcFiles = []string{filepath.Join(home, ".bashrc")}
	}

	for _, rc := range rcFiles {
		if err := ensureBlockInFile(rc, block); err != nil {
			return err
		}
	}
	return nil
}

func ensureZshRc() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Ensure completions dir exists and is in fpath; init compinit
	block := rcBlockStart + "\n" +
		"fpath=(~/.zsh/completions $fpath)\n" +
		"autoload -U compinit\n" +
		"compinit\n" +
		rcBlockEnd + "\n"

	rc := filepath.Join(home, ".zshrc")
	return ensureBlockInFile(rc, block)
}

func ensureBlockInFile(path string, block string) error {
	// Read existing content if file exists
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory for rc file if needed
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return err
			}
			// Create new file with block
			return os.WriteFile(path, []byte(block), 0o644)
		}
		return err
	}

	content := string(data)
	if strings.Contains(content, rcBlockStart) {
		return nil
	}

	// Ensure newline separation
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += block
	return os.WriteFile(path, []byte(content), 0o644)
}

var completionUninstallCmd = &cobra.Command{
	Use:   "uninstall [bash|zsh|fish|powershell]",
	Short: "Uninstall shell completion script",
	Long:  "Remove installed completion script and shell init blocks. If shell is not provided, the system default will be used.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sh := ""
		if len(args) == 1 {
			sh = strings.ToLower(strings.TrimSpace(args[0]))
		} else {
			sh = detectShell()
		}
		if sh == "" {
			return errors.New("could not detect shell; please specify one: bash|zsh|fish|powershell")
		}

		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		var targetPath string
		switch sh {
		case "bash":
			targetPath = filepath.Join(home, ".local", "share", "bash-completion", "completions", "whodb-cli")
			if err := removeBashRc(); err != nil {
				return err
			}
		case "zsh":
			targetPath = filepath.Join(home, ".zsh", "completions", "_whodb-cli")
			if err := removeZshRc(); err != nil {
				return err
			}
		case "fish":
			targetPath = filepath.Join(home, ".config", "fish", "completions", "whodb-cli.fish")
		case "powershell", "pwsh":
			fmt.Fprintln(os.Stdout, "PowerShell uninstall is manual: remove completion lines from $PROFILE and any saved completion file.")
			return nil
		default:
			return fmt.Errorf("unsupported shell: %s", sh)
		}

		if err := removeFileIfExists(targetPath); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Removed %s completion from: %s\n", sh, targetPath)
		return nil
	},
}

func removeBashRc() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	var rcFiles []string
	if runtime.GOOS == "darwin" {
		rcFiles = []string{filepath.Join(home, ".bash_profile"), filepath.Join(home, ".bashrc")}
	} else {
		rcFiles = []string{filepath.Join(home, ".bashrc")}
	}
	for _, rc := range rcFiles {
		if err := removeBlockFromFile(rc); err != nil {
			return err
		}
	}
	return nil
}

func removeZshRc() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	rc := filepath.Join(home, ".zshrc")
	return removeBlockFromFile(rc)
}

func removeBlockFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	content := string(data)
	start := strings.Index(content, rcBlockStart)
	if start == -1 {
		return nil
	}
	end := strings.Index(content[start:], rcBlockEnd)
	if end == -1 {
		return nil
	}
	end += start + len(rcBlockEnd)

	newContent := content[:start] + content[end:]
	newContent = strings.ReplaceAll(newContent, "\n\n\n", "\n\n")
	return os.WriteFile(path, []byte(newContent), 0o644)
}

func removeFileIfExists(path string) error {
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
