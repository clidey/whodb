package cmd

import "testing"

func TestProjectBundleNamesKeepLegacyAliases(t *testing.T) {
	for _, test := range []struct {
		command  string
		expected string
	}{{"export-project", "export-project"}, {"backup-project", "export-project"}, {"import-project", "import-project"}, {"restore-project", "import-project"}} {
		command, _, err := rootCmd.Find([]string{test.command})
		if err != nil {
			t.Fatal(err)
		}
		if command.Name() != test.expected {
			t.Errorf("%s resolved to %s, want %s", test.command, command.Name(), test.expected)
		}
	}
}
