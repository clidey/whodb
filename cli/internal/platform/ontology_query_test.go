package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOntologyQueryCompletenessContract(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "current", true: "legacy"}[legacy], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					Query string `json:"query"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				if strings.Contains(request.Query, "partial") == legacy || strings.Contains(request.Query, "scannedRows") == legacy {
					t.Errorf("incorrect selection: %s", request.Query)
				}
				w.Header().Set("Content-Type", "application/json")
				if legacy {
					_, _ = w.Write([]byte(`{"data":{"OntologyQuery":{"columns":["count"],"rows":[["1000"]],"total":1}}}`))
					return
				}
				_, _ = w.Write([]byte(`{"data":{"OntologyQuery":{"columns":["count"],"rows":[["1000"]],"total":1,"partial":true,"scannedRows":1001}}}`))
			}))
			defer server.Close()
			client, err := NewClient(server.URL, "token")
			if err != nil {
				t.Fatal(err)
			}
			fields := []PlatformManifestField{{Name: "columns"}, {Name: "rows"}, {Name: "total"}}
			if !legacy {
				fields = append(fields, PlatformManifestField{Name: "partial"}, PlatformManifestField{Name: "scannedRows"})
			}
			client.SetPlatformManifest(&PlatformManifest{Operations: []PlatformManifestOperation{{Kind: "Query", Name: "OntologyQuery"}}, Types: []PlatformManifestType{{Name: "DatasetQueryResult", Fields: fields}}})
			result, err := client.OntologyQuery(context.Background(), "project", OntologyQueryInput{Entity: "Sales"})
			if err != nil {
				t.Fatal(err)
			}
			if legacy {
				if result.Partial != nil {
					t.Fatal("legacy result incorrectly claimed completeness")
				}
				return
			}
			if result.Partial == nil || !*result.Partial || result.ScannedRows == nil || *result.ScannedRows != 1001 {
				t.Fatalf("lost metadata: %+v", result)
			}
		})
	}
}
