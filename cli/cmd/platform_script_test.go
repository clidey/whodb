package cmd

import "testing"

func TestScriptPreviewCommandDoesNotRequireAResourceID(t *testing.T) {
	command := scriptExecuteCommand()
	if err := command.Args(command, nil); err != nil {
		t.Fatalf("script preview required an id: %v", err)
	}
	if err := command.Args(command, []string{"unexpected-id"}); err == nil {
		t.Fatal("unexpected id accepted")
	}
	if err := command.RunE(command, nil); err == nil || err.Error() != "--file is required" {
		t.Fatalf("missing file did not fail before workspace access: %v", err)
	}
}
