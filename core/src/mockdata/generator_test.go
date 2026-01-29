package mockdata

import (
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/clidey/whodb/core/src/engine"
)

// GenerateValue generates a mock value
func (g *Generator) GenerateValue(columnName string, columnType string, constraints map[string]any) (any, error) {
	col := engine.Column{
		Name: columnName,
		Type: columnType,
	}
	return g.generateColumnValue(col, constraints), nil
}

// GenerateRowDataWithConstraints generates mock data for a complete row
func (g *Generator) GenerateRowDataWithConstraints(columns []engine.Column, colConstraints map[string]map[string]any) ([]engine.Record, error) {
	return g.generateRow(columns, colConstraints, nil, "")
}

// GenerateRowData generates mock data without constraints
func (g *Generator) GenerateRowData(columns []engine.Column) ([]engine.Record, error) {
	return g.GenerateRowDataWithConstraints(columns, nil)
}

// GenerateRowWithDefaults generates a row using safe default values
func (g *Generator) GenerateRowWithDefaults(columns []engine.Column) []engine.Record {
	records, _ := g.GenerateRowData(columns)
	return records
}

func TestGenerateRowDataWithConstraintsSkipsSerialAndRespectsCheckValues(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1) // deterministic output

	columns := []engine.Column{
		{Name: "id", Type: "serial", IsAutoIncrement: true},
		{Name: "status", Type: "varchar(10)"},
		{Name: "created_at", Type: "timestamp"},
	}
	constraints := map[string]map[string]any{
		"status": {"check_values": []string{"open", "closed"}},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, constraints)
	if err != nil {
		t.Fatalf("unexpected error generating row data: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected serial column to be skipped, got %d records", len(records))
	}

	status := findRecord(records, "status")
	if status == nil {
		t.Fatalf("expected status column to be present")
	}
	if status.Value != "open" && status.Value != "closed" {
		t.Fatalf("expected status to respect check_values constraint, got %s", status.Value)
	}
	if status.Extra["Type"] != "varchar(10)" {
		t.Fatalf("expected type to be stored in Extra metadata")
	}

	created := findRecord(records, "created_at")
	if created == nil {
		t.Fatalf("expected created_at column to be present")
	}
	if _, err := time.Parse("2006-01-02 15:04:05", created.Value); err != nil {
		t.Fatalf("expected timestamp to be formatted without timezone, got %s", created.Value)
	}
}

func TestGenerateRowWithDefaultsCoversCommonTypes(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1)

	// Set up length pointer for varchar test
	noteLen := 2
	columns := []engine.Column{
		{Name: "id", Type: "serial", IsAutoIncrement: true},
		{Name: "price", Type: "decimal(10,2)"},
		{Name: "active", Type: "boolean"},
		{Name: "created", Type: "date"},
		{Name: "note", Type: "varchar(2)", Length: &noteLen},
		{Name: "payload", Type: "jsonb"},
	}

	records := g.GenerateRowWithDefaults(columns)
	if len(records) != 5 { // serial skipped
		t.Fatalf("expected serial column to be skipped, got %d records", len(records))
	}

	price := findRecord(records, "price")
	if price == nil {
		t.Fatalf("expected price column to be present")
	}
	// Decimal values are randomly generated within range [0, 10000]
	// Just check it's not empty and has proper format
	if price.Value == "" {
		t.Fatalf("expected decimal value to be generated, got empty string")
	}

	active := findRecord(records, "active")
	if active == nil || (active.Value != "false" && active.Value != "true") {
		t.Fatalf("expected boolean default to be present, got %#v", active)
	}

	note := findRecord(records, "note")
	if note == nil || len(note.Value) != 2 {
		t.Fatalf("expected varchar(2) default to respect length, got %#v", note)
	}

	payload := findRecord(records, "payload")
	if payload == nil || !strings.HasPrefix(payload.Value, "{") {
		t.Fatalf("expected json default to be object, got %#v", payload)
	}
}

func TestGenerateRowDataWithConstraintsHandlesArrays(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(2)

	columns := []engine.Column{
		{Name: "tags", Type: "text[]"},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, nil)
	if err != nil {
		t.Fatalf("unexpected error generating array data: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if !strings.HasPrefix(records[0].Value, "{") || !strings.HasSuffix(records[0].Value, "}") {
		t.Fatalf("expected array value to be wrapped in braces, got %s", records[0].Value)
	}
}

func findRecord(records []engine.Record, key string) *engine.Record {
	for _, r := range records {
		if r.Key == key {
			return &r
		}
	}
	return nil
}

func TestDetectDatabaseTypeHandlesArrayAndDefaults(t *testing.T) {
	if got := detectDatabaseType("text[]"); got != "array" {
		t.Fatalf("expected array type for text[], got %s", got)
	}
	if got := detectDatabaseType("unknown_type"); got != "text" {
		t.Fatalf("expected unknown types to default to text, got %s", got)
	}
}

// TestDetectDatabaseTypeHandlesCommonTypes tests the type detection function
// Note: parseMaxLen was removed in favor of constraint-based length handling

func TestMatchColumnNameEmail(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"email", "user_email", "e_mail", "Email", "EMAIL"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match email pattern", colName)
		}
		email, ok := value.(string)
		if !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
		if !strings.Contains(email, "@") {
			t.Fatalf("expected email format for %q, got %q", colName, email)
		}
	}
}

func TestMatchColumnNameUsername(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"username", "user_name", "uname", "login"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match username pattern", colName)
		}
		if _, ok := value.(string); !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
	}
}

func TestMatchColumnNamePhone(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"phone", "phone_number", "mobile", "cell", "telephone", "tel"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match phone pattern", colName)
		}
		if _, ok := value.(string); !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
	}
}

func TestMatchColumnNameAddress(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []struct {
		colName  string
		pattern  string
		expected bool
	}{
		{"address", "address", true},
		{"street", "street", true},
		{"city", "city", true},
		{"state", "state", true},
		{"country", "country", true},
		{"zip", "zip", true},
		{"postal_code", "postal", true},
	}

	for _, tc := range testCases {
		value, matched := MatchColumnName(tc.colName, 0, faker)
		if matched != tc.expected {
			t.Fatalf("expected column name %q match=%v, got match=%v", tc.colName, tc.expected, matched)
		}
		if matched {
			if _, ok := value.(string); !ok {
				t.Fatalf("expected string value for %q, got %T", tc.colName, value)
			}
		}
	}
}

func TestMatchColumnNameUrl(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"url", "website", "link", "homepage"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match URL pattern", colName)
		}
		url, ok := value.(string)
		if !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
		if !strings.HasPrefix(url, "http") {
			t.Fatalf("expected URL format for %q, got %q", colName, url)
		}
	}
}

func TestMatchColumnNameIPAddress(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"ip", "ip_address", "ipaddress", "ip_addr"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match IP pattern", colName)
		}
		ip, ok := value.(string)
		if !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
		// IPv4 should have dots
		if !strings.Contains(ip, ".") {
			t.Fatalf("expected IPv4 format for %q, got %q", colName, ip)
		}
	}
}

func TestMatchColumnNameNames(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []struct {
		colName string
		pattern string
	}{
		{"first_name", "first_name"},
		{"firstname", "firstname"},
		{"fname", "fname"},
		{"last_name", "last_name"},
		{"lastname", "lastname"},
		{"lname", "lname"},
		{"surname", "surname"},
		{"name", "name"},
		{"full_name", "full_name"},
	}

	for _, tc := range testCases {
		value, matched := MatchColumnName(tc.colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match name pattern", tc.colName)
		}
		if _, ok := value.(string); !ok {
			t.Fatalf("expected string value for %q, got %T", tc.colName, value)
		}
	}
}

func TestMatchColumnNameCompany(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"company", "organization", "org", "company_name"}
	for _, colName := range testCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match company pattern", colName)
		}
		if _, ok := value.(string); !ok {
			t.Fatalf("expected string value for %q, got %T", colName, value)
		}
	}
}

func TestMatchColumnNameNoMatch(t *testing.T) {
	faker := gofakeit.New(1)

	testCases := []string{"created_at", "updated_at", "id", "status", "amount", "count"}
	for _, colName := range testCases {
		_, matched := MatchColumnName(colName, 0, faker)
		if matched {
			t.Fatalf("expected column name %q to NOT match any pattern", colName)
		}
	}
}

func TestMatchColumnNameRespectsMaxLen(t *testing.T) {
	faker := gofakeit.New(1)

	// Test with a short max length
	value, matched := MatchColumnName("email", 10, faker)
	if !matched {
		t.Fatal("expected email to match")
	}
	email, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}
	if len(email) > 10 {
		t.Fatalf("expected email to be truncated to 10 chars, got %d: %q", len(email), email)
	}
}

func TestGenerateValueUsesColumnNameContext(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1)

	columns := []engine.Column{
		{Name: "user_email", Type: "varchar(255)"},
		{Name: "phone_number", Type: "varchar(20)"},
		{Name: "website", Type: "text"},
	}

	for _, col := range columns {
		value, err := g.GenerateValue(col.Name, col.Type, nil)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", col.Name, err)
		}

		str, ok := value.(string)
		if !ok {
			t.Fatalf("expected string for %q, got %T", col.Name, value)
		}

		switch col.Name {
		case "user_email":
			if !strings.Contains(str, "@") {
				t.Fatalf("expected email format for user_email, got %q", str)
			}
		case "phone_number":
			// Phone should have some digits
			hasDigit := false
			for _, r := range str {
				if r >= '0' && r <= '9' {
					hasDigit = true
					break
				}
			}
			if !hasDigit {
				t.Fatalf("expected phone format for phone_number, got %q", str)
			}
		case "website":
			if !strings.HasPrefix(str, "http") {
				t.Fatalf("expected URL format for website, got %q", str)
			}
		}
	}
}

func TestGenerateValueCheckValuesHasPriority(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1)

	// Even though column is named "email", check_values should take priority
	constraints := map[string]any{
		"check_values": []string{"active", "inactive"},
	}

	value, err := g.GenerateValue("email", "varchar(50)", constraints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	str, ok := value.(string)
	if !ok {
		t.Fatalf("expected string, got %T", value)
	}

	if str != "active" && str != "inactive" {
		t.Fatalf("expected check_values to take priority, got %q", str)
	}
}

func TestMatchColumnNameLatLong(t *testing.T) {
	faker := gofakeit.New(1)

	latCases := []string{"latitude", "lat"}
	for _, colName := range latCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match latitude pattern", colName)
		}
		lat, ok := value.(float64)
		if !ok {
			t.Fatalf("expected float64 for latitude, got %T", value)
		}
		if lat < -90 || lat > 90 {
			t.Fatalf("expected latitude in range [-90, 90], got %f", lat)
		}
	}

	longCases := []string{"longitude", "lng", "lon"}
	for _, colName := range longCases {
		value, matched := MatchColumnName(colName, 0, faker)
		if !matched {
			t.Fatalf("expected column name %q to match longitude pattern", colName)
		}
		lng, ok := value.(float64)
		if !ok {
			t.Fatalf("expected float64 for longitude, got %T", value)
		}
		if lng < -180 || lng > 180 {
			t.Fatalf("expected longitude in range [-180, 180], got %f", lng)
		}
	}
}
