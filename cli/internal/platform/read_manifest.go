package platform

import (
	"bytes"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
	"github.com/vektah/gqlparser/v2/parser"
)

// platformReadDocument limits fixed read projections to fields advertised by the host.
func (c *Client) platformReadDocument(operation, query string) string {
	if c.manifest == nil {
		return query
	}
	document, err := parser.ParseQuery(&ast.Source{Input: query})
	if err != nil {
		return query
	} // Fixed documents are checked against the schema in tests.
	types := map[string]map[string]string{}
	for _, typ := range c.manifest.Types {
		fields := map[string]string{}
		for _, f := range typ.Fields {
			fields[f.Name] = strings.Trim(f.Type, "[]!")
		}
		types[typ.Name] = fields
	}
	var selectFields func(ast.SelectionSet, string) ast.SelectionSet
	selectFields = func(selections ast.SelectionSet, typeName string) ast.SelectionSet {
		fields, known := types[typeName]
		if !known {
			return selections
		}
		out := ast.SelectionSet{}
		for _, selection := range selections {
			f, ok := selection.(*ast.Field)
			if !ok {
				out = append(out, selection)
				continue
			}
			childType, supported := fields[f.Name]
			if !supported && f.Name != "__typename" {
				continue
			}
			if len(f.SelectionSet) > 0 {
				f.SelectionSet = selectFields(f.SelectionSet, childType)
			}
			out = append(out, f)
		}
		if len(out) == 0 {
			return ast.SelectionSet{&ast.Field{Name: "__typename"}}
		}
		return out
	}
	for _, op := range c.manifest.Operations {
		if (operation != "" && op.Name != operation) || op.Kind != "Query" {
			continue
		}
		for _, d := range document.Operations {
			for _, selection := range d.SelectionSet {
				if f, ok := selection.(*ast.Field); ok && f.Name == op.Name {
					f.SelectionSet = selectFields(f.SelectionSet, strings.Trim(op.Returns, "[]!"))
				}
			}
		}
	}
	var buffer bytes.Buffer
	formatter.NewFormatter(&buffer).FormatQueryDocument(document)
	return buffer.String()
}
