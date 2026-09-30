/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package appcapture

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestBrowserScriptReadOnlyDocument runs the embedded isReadOnlyDocument helper
// under Node against read-only documents and every known way of smuggling a
// write: decoy queries, operationName selection, comment and string interleaving,
// separators, and malformed documents that must fail closed.
func TestBrowserScriptReadOnlyDocument(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is not installed")
	}
	start := strings.Index(browserScript, "const isReadOnlyDocument =")
	if start < 0 {
		t.Fatal("isReadOnlyDocument helper not found in capture.mjs")
	}
	end := strings.Index(browserScript[start:], "\n};\n")
	if end < 0 {
		t.Fatal("isReadOnlyDocument helper is not terminated")
	}
	helper := browserScript[start : start+end+4]

	cases := []struct {
		name     string
		document any
		readOnly bool
	}{
		{"plain query", "query Apps { apps { id } }", true},
		{"shorthand query", "{ apps { id } }", true},
		{"leading whitespace and comment", "  # c\n query A { a }", true},
		{"leading BOM", "\uFEFFquery A { a }", true},
		{"mutation word inside string", `query A { rows(filter: "mutation pending") { id } }`, true},
		{"mutation word inside block string", "query A { rows(desc: \"\"\"a mutation\n here\"\"\") { id } }", true},
		{"mutation word inside comment", "query A { a } # mutation B { del }", true},
		{"escaped quote inside string", `query A { rows(f: "say \" mutation") { id } }`, true},
		{"hash inside string", `query A { rows(f: "a # mutation") { id } }`, true},
		{"quote inside comment", `query A { a } # it's "quoted" mutation in comment`, true},
		{"field named mutationLog", "query A { mutationLog { id } }", true},
		{"fragment named mutationFields", "query A { ...mutationFields } fragment mutationFields on Query { a }", true},
		{"variable default containing mutation", `query A($f: String = "mutation") { a(f: $f) }`, true},

		{"plain mutation", "mutation M { del(id: 1) }", false},
		{"mutation with leading whitespace", "   \n\tmutation M { x }", false},
		{"decoy query then mutation", "query A { __typename } mutation B { del(id: 1) }", false},
		{"decoy with comment then mutation", "query A { __typename } # x\nmutation B { del(id: 1) }", false},
		{"block string opener inside comment", "query A { __typename } # \"\"\"\nmutation B { del(id: 1) }\n# \"\"\"", false},
		{"quote inside comment across lines", "query A { __typename } # \"\nmutation B { del(id: 1) }\n# \"", false},
		{"escaped block string close", "query A { f(a: \"\"\"x\\\"\"\" y\"\"\") } mutation B { del }", false},
		{"unclosed block string", "query A { f(a: \"\"\"x) } mutation B { del }", false},
		{"unclosed string", "query A { f(a: \"x) } mutation B { del }", false},
		{"raw newline inside string", "query A { f(a: \"x\n\") } mutation B { del }", false},
		{"comma separator", "query A{a},mutation B{del}", false},
		{"tab and CRLF separators", "query A{a}\r\n\tmutation B{del}", false},
		{"BOM separator", "query A{a}\uFEFFmutation B{del}", false},
		{"mutation nested in braces", "query A { a { mutation B { del } } }", false},
		{"subscription", "query A { a } subscription S { s }", false},
		{"empty document", "", false},
		{"missing document", nil, false},
		{"batched array body", []string{"mutation M { x }"}, false},
		{"object body", map[string]string{"query": "mutation M { x }"}, false},
	}

	documents := make([]any, len(cases))
	for i, c := range cases {
		documents[i] = c.document
	}
	input, err := json.Marshal(documents)
	if err != nil {
		t.Fatal(err)
	}
	script := helper + `
const documents = JSON.parse(require("node:fs").readFileSync(0, "utf8"));
process.stdout.write(JSON.stringify(documents.map(isReadOnlyDocument)));
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", "--eval", script)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node: %v: %s", err, stderr.String())
	}
	var results []bool
	if err := json.Unmarshal(out, &results); err != nil {
		t.Fatalf("decode results: %v: %s", err, out)
	}
	if len(results) != len(cases) {
		t.Fatalf("got %d results for %d cases", len(results), len(cases))
	}
	for i, c := range cases {
		if results[i] != c.readOnly {
			t.Errorf("%s: isReadOnlyDocument = %v, want %v", c.name, results[i], c.readOnly)
		}
	}
}
