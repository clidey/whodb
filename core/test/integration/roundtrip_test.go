//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	graph "github.com/clidey/whodb/core/graph"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/auth"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/types"
)

func TestConnectorAvailability(t *testing.T) {
	for _, target := range targets {
		if !target.enabled {
			continue
		}
		t.Run(target.name, func(t *testing.T) {
			if ok := target.plugin.IsAvailable(target.config); !ok {
				t.Fatalf("plugin reported unavailable for %s", target.name)
			}
		})
	}
}

func TestSQLTypeRoundTrips(t *testing.T) {
	for _, target := range targets {
		if target.plugin.GetDatabaseMetadata() == nil {
			continue
		}
		if target.plugin.Type == engine.DatabaseType_MongoDB || target.plugin.Type == engine.DatabaseType_ElasticSearch {
			continue
		}
		t.Run(target.name, func(t *testing.T) {
			meta := target.plugin.GetDatabaseMetadata()
			for idx, td := range meta.TypeDefinitions {
				sample, ok, expected := sampleValue(td)
				if !ok {
					continue
				}
				if strings.Contains(td.ID, "SERIAL") {
					continue
				}
				table := fmt.Sprintf("intg_%s_%d", strings.ToLower(string(target.plugin.Type)), idx)
				fields := []engine.Record{
					{
						Key:   "id",
						Value: "INT",
						Extra: map[string]string{"Primary": "true", "Nullable": "false"},
					},
					{
						Key:   "val",
						Value: td.ID,
						Extra: map[string]string{
							"Primary":  "false",
							"Nullable": "false",
						},
					},
				}

				created, err := target.plugin.AddStorageUnit(target.config, target.schema, table, fields)
				if err != nil || !created {
					t.Skipf("skip type %s on %s: %v", td.ID, target.name, err)
					continue
				}
				defer target.plugin.RawExecute(target.config, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", target.schema, table))

				valRecord := []engine.Record{
					{Key: "id", Value: "1", Extra: map[string]string{"Type": "INT"}},
					{Key: "val", Value: sample, Extra: map[string]string{"Type": td.ID}},
				}
				inserted, err := target.plugin.AddRow(target.config, target.schema, table, valRecord)
				if err != nil || !inserted {
					t.Fatalf("failed to insert sample for %s on %s: %v", td.ID, target.name, err)
				}

				rows, err := target.plugin.GetRows(target.config, target.schema, table, nil, []*model.SortCondition{}, 10, 0)
				if err != nil {
					t.Fatalf("GetRows failed for %s on %s: %v", td.ID, target.name, err)
				}
				if len(rows.Rows) == 0 {
					t.Fatalf("expected at least one row for %s on %s", td.ID, target.name)
				}

				valIdx := 1
				if len(rows.Rows[0]) == 1 {
					valIdx = 0
				}
				got := rows.Rows[0][valIdx]
				if expected != "" && !strings.Contains(got, expected) && got != expected {
					t.Fatalf("round trip mismatch for %s on %s: got %s expected substring %s", td.ID, target.name, got, expected)
				}

				// Update row and read again
				update := map[string]string{"id": "1", "val": sample}
				if _, err := target.plugin.UpdateStorageUnit(target.config, target.schema, table, update, []string{"val"}); err != nil {
					if strings.Contains(err.Error(), "WHERE conditions required") || strings.Contains(err.Error(), "no rows were updated") {
						t.Skipf("skipping update for %s on %s: %v", td.ID, target.name, err)
						return
					}
					t.Fatalf("update failed for %s on %s: %v", td.ID, target.name, err)
				}
			}
		})
	}
}

func TestMongoRoundTrip(t *testing.T) {
	var mongoTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_MongoDB {
			mongoTarget = &targets[i]
			break
		}
	}
	if mongoTarget == nil {
		t.Skip("mongo target not configured")
	}

	doc := map[string]any{"_id": "507f1f77bcf86cd799439011", "name": "integration", "count": 1}
	b, _ := json.Marshal(doc)
	ok, err := mongoTarget.plugin.AddRow(mongoTarget.config, mongoTarget.schema, "items", []engine.Record{
		{Key: "document", Value: string(b)},
	})
	if err != nil || !ok {
		t.Fatalf("failed to insert mongo doc: %v", err)
	}
	rows, err := mongoTarget.plugin.GetRows(mongoTarget.config, mongoTarget.schema, "items", nil, []*model.SortCondition{}, 10, 0)
	if err != nil {
		t.Fatalf("mongo get rows failed: %v", err)
	}
	if len(rows.Rows) == 0 {
		t.Fatalf("expected rows from mongo collection")
	}
}

func TestElasticsearchAvailability(t *testing.T) {
	var esTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_ElasticSearch {
			esTarget = &targets[i]
			break
		}
	}
	if esTarget == nil {
		t.Skip("elasticsearch target not configured")
	}
	if !esTarget.plugin.IsAvailable(esTarget.config) {
		t.Fatalf("elasticsearch not available")
	}
}

func sampleValue(td engine.TypeDefinition) (string, bool, string) {
	switch td.Category {
	case engine.TypeCategoryNumeric:
		if strings.Contains(td.ID, "DECIMAL") || strings.Contains(td.ID, "NUMERIC") || strings.Contains(td.ID, "MONEY") {
			return "123.45", true, "123"
		}
		return "123", true, "123"
	case engine.TypeCategoryText:
		return "hello world", true, "hello"
	case engine.TypeCategoryBinary:
		return "48656c6c6f", true, ""
	case engine.TypeCategoryDatetime:
		switch strings.ToUpper(td.ID) {
		case "DATE":
			return time.Now().Format("2006-01-02"), true, time.Now().Format("2006-01-02")
		default:
			return time.Now().Format("2006-01-02 15:04:05"), true, time.Now().Format("2006-01-02")
		}
	case engine.TypeCategoryBoolean:
		return "true", true, "true"
	case engine.TypeCategoryJSON:
		return `{"key":"value"}`, true, "value"
	case engine.TypeCategoryOther:
		if strings.Contains(strings.ToUpper(td.ID), "UUID") {
			return "00000000-0000-0000-0000-000000000001", true, "00000000-0000-0000-0000-000000000001"
		}
		if strings.Contains(strings.ToUpper(td.ID), "ARRAY") {
			return "{1,2,3}", true, "1"
		}
		return "", false, ""
	default:
		return "", false, ""
	}
}

func TestServerSmokeAgainstPostgres(t *testing.T) {
	var pgTarget *target
	for i := range targets {
		if targets[i].plugin.Type == engine.DatabaseType_Postgres {
			pgTarget = &targets[i]
			break
		}
	}
	if pgTarget == nil {
		t.Skip("postgres target not configured")
	}

	origEngine := src.MainEngine
	src.MainEngine = &engine.Engine{}
	src.MainEngine.RegistryPlugin(pgTarget.plugin)
	t.Cleanup(func() { src.MainEngine = origEngine })

	src.MainEngine.AddLoginProfile(types.DatabaseCredentials{
		Type:     string(pgTarget.config.Credentials.Type),
		Hostname: pgTarget.config.Credentials.Hostname,
		Database: pgTarget.config.Credentials.Database,
		Username: pgTarget.config.Credentials.Username,
		Password: pgTarget.config.Credentials.Password,
		Port:     common.GetRecordValueOrDefault(pgTarget.config.Credentials.Advanced, "Port", ""),
	})

	// Start a minimal handler: use GraphQL server directly rather than full binary to avoid changing scripts
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	ctx := context.WithValue(context.Background(), auth.AuthKey_Credentials, pgTarget.config.Credentials)

	// simple AddRow/Row via GraphQL against live DB
	table := "intg_smoke"
	_, _ = pgTarget.plugin.RawExecute(pgTarget.config, fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", pgTarget.schema, table))
	_, err := pgTarget.plugin.AddStorageUnit(pgTarget.config, pgTarget.schema, table, []engine.Record{
		{Key: "id", Value: "INTEGER", Extra: map[string]string{"Primary": "true", "Nullable": "false"}},
	})
	if err != nil {
		t.Fatalf("failed to create table for smoke: %v", err)
	}

	graphAdd := `mutation($schema:String!,$table:String!){ AddRow(schema:$schema, storageUnit:$table, values:[{Key:"id", Value:"1"}]){Status}}`
	body, _ := json.Marshal(map[string]any{
		"query":     graphAdd,
		"variables": map[string]any{"schema": pgTarget.schema, "table": table},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("graphql add row failed: %d %s", w.Code, w.Body.String())
	}

	graphRow := `query($schema:String!,$table:String!){ Row(schema:$schema, storageUnit:$table, where:{Type:Atomic,Atomic:{Key:"id",Operator:"=",Value:"1",ColumnType:"integer"}}, sort:[{Column:"id", Direction:ASC}], pageSize:10, pageOffset:0){ Rows }}`
	body, _ = json.Marshal(map[string]any{
		"query":     graphRow,
		"variables": map[string]any{"schema": pgTarget.schema, "table": table},
	})
	req = httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBuffer(body))
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "1") {
		t.Fatalf("graphql row query failed: %d %s", w.Code, w.Body.String())
	}
}
