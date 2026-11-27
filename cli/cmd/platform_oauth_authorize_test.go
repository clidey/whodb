package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/clidey/whodb/cli/pkg/output"
)

func TestPresentOAuthAuthorizationLinkOnlyPrintsWhenNeeded(t *testing.T) {
	const link = "https://app.whodb.test/api/secrets/oauth/handoff?state=Zm9v"
	for _, test := range []struct {
		name                   string
		format                 output.Format
		noBrowser, failBrowser bool
		wantBrowser, wantLink  bool
	}{
		{name: "browser success", format: output.FormatAuto, wantBrowser: true},
		{name: "browser failure", format: output.FormatAuto, failBrowser: true, wantBrowser: true, wantLink: true},
		{name: "explicit link", format: output.FormatPlain, noBrowser: true, wantLink: true},
		{name: "JSON output", format: output.FormatJSON, wantLink: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			opened := false
			err := presentOAuthAuthorizationLink(cmd, test.format, test.noBrowser, link, func(target string) error {
				opened = true
				if target != link {
					t.Fatalf("opening unexpected link: %q", target)
				}
				if test.failBrowser {
					return errors.New("browser unavailable")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if opened != test.wantBrowser || strings.Contains(stdout.String(), link) != test.wantLink || strings.Contains(stderr.String(), link) {
				t.Fatalf("browser opened=%t, stdout=%q, stderr=%q", opened, stdout.String(), stderr.String())
			}
			if test.failBrowser && !strings.Contains(stderr.String(), "Could not open the browser") {
				t.Fatal("missing browser failure message")
			}
		})
	}
}
