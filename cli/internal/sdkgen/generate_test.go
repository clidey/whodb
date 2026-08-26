package sdkgen

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

var update = flag.Bool("update", false, "rewrite golden files")

func fixtureOntologies() []platformapi.Ontology {
	return []platformapi.Ontology{
		{
			APIName:     "user",
			DisplayName: "User",
			Description: "A registered user.",
			PrimaryKey:  "id",
			Properties: []platformapi.OntologyProperty{
				{APIName: "id", DisplayName: "ID", DataType: "UUID", IsRequired: true},
				{APIName: "email", DisplayName: "Email", DataType: "String", IsRequired: true},
				{APIName: "age", DisplayName: "Age", DataType: "Integer"},
				{APIName: "active", DisplayName: "Active", DataType: "Boolean", IsRequired: true},
				{APIName: "joined_at", DisplayName: "Joined", DataType: "Timestamp"},
			},
			Links: []platformapi.OntologyLink{
				{APIName: "orders", TargetOntologyAPIName: "order", Cardinality: "ONE_TO_MANY"},
			},
		},
		{
			APIName:     "order",
			DisplayName: "Order",
			PrimaryKey:  "order_id",
			Properties: []platformapi.OntologyProperty{
				{APIName: "order_id", DisplayName: "Order ID", DataType: "UUID", IsRequired: true},
				{APIName: "total", DisplayName: "Total", DataType: "Double", IsRequired: true},
			},
		},
	}
}

func TestModelIsDeterministic(t *testing.T) {
	first := BuildModel(fixtureOntologies())
	second := BuildModel(fixtureOntologies())
	if first.Hash() == "" || first.Hash() != second.Hash() {
		t.Fatalf("model hash must be stable and non-empty, got %q vs %q", first.Hash(), second.Hash())
	}
}

func TestIdentifier(t *testing.T) {
	cases := map[string]string{
		"user":         "User",
		"user_account": "UserAccount",
		"UserAccount":  "UserAccount",
		"9lives":       "E9lives",
		"":             "Entity",
	}
	for input, want := range cases {
		if got := identifier(input); got != want {
			t.Errorf("identifier(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestValidateModelRejectsIdentifierCollision(t *testing.T) {
	model := &Model{Entities: []Entity{{APIName: "user_account"}, {APIName: "UserAccount"}}}
	if err := validateModel(model); err == nil {
		t.Fatal("expected identifier-collision error")
	}
}

func TestGenerateGolden(t *testing.T) {
	model := BuildModel(fixtureOntologies())
	for _, testCase := range []struct {
		language Language
		golden   string
		outFile  string
	}{
		{LanguageTypeScript, "typescript.golden.ts", "whodb.generated.ts"},
		{LanguagePython, "python.golden.py", "whodb_generated.py"},
	} {
		t.Run(string(testCase.language), func(t *testing.T) {
			outDir := t.TempDir()
			written, err := Generate(model, testCase.language, outDir)
			if err != nil {
				t.Fatal(err)
			}
			if len(written) != 1 || filepath.Base(written[0]) != testCase.outFile {
				t.Fatalf("unexpected outputs: %v", written)
			}
			got, err := os.ReadFile(written[0])
			if err != nil {
				t.Fatal(err)
			}
			goldenPath := filepath.Join("testdata", testCase.golden)
			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with -update to create): %v", err)
			}
			if string(got) != string(want) {
				t.Errorf("output differs from %s — run: go test ./internal/sdkgen -update\nfirst diff near: %s", goldenPath, firstDiff(string(got), string(want)))
			}
		})
	}
}

func firstDiff(a, b string) string {
	aLines, bLines := strings.Split(a, "\n"), strings.Split(b, "\n")
	for i := range aLines {
		if i >= len(bLines) || aLines[i] != bLines[i] {
			return aLines[i]
		}
	}
	return "(b longer than a)"
}
