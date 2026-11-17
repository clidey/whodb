package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func scriptExecuteCommand() *cobra.Command {
	var language, file, sample string
	var dependencies []string
	command := typedResourceWriteCommand("run", "Run a confirmed script preview on the platform; omitted sample data uses built-in sample rows", "action", "script", "execute", func(*cobra.Command) (map[string]any, error) {
		if strings.TrimSpace(file) == "" {
			return nil, errors.New("--file is required")
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		payload := map[string]any{"language": language, "script": string(data), "dependencies": dependencies}
		if sample != "" {
			payload["sampleData"] = sample
		}
		return payload, nil
	})
	command.Args = cobra.NoArgs
	command.Flags().StringVar(&file, "file", "", "script source file")
	command.Flags().StringVar(&language, "language", "python", "script language: python, r or julia")
	command.Flags().StringSliceVar(&dependencies, "dependencies", []string{}, "curated script dependencies")
	command.Flags().StringVar(&sample, "sample-data", "", "JSON object containing columns and rows of string cells")
	command.Flags().BoolVarP(&platformWriteYes, "yes", "y", false, "approve platform script execution")
	return command
}
