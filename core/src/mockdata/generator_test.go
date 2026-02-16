/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mockdata

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
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
	constraints := map[string]map[string]any{
		"tags": {"nullable": false},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, constraints)
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

func TestDetectDatabaseTypeHandlesTimestampVariants(t *testing.T) {
	// PostgreSQL timestamp types with timezone
	timestampTypes := []string{
		"TIMESTAMP",
		"TIMESTAMP WITH TIME ZONE",
		"TIMESTAMP WITHOUT TIME ZONE",
		"timestamp with time zone",
		"timestamptz",
	}
	for _, typeName := range timestampTypes {
		if got := detectDatabaseType(typeName); got != "datetime" {
			t.Errorf("expected datetime for %q, got %s", typeName, got)
		}
	}

	// Time types
	timeTypes := []string{
		"TIME",
		"TIME WITH TIME ZONE",
		"TIME WITHOUT TIME ZONE",
		"time with time zone",
		"timetz",
	}
	for _, typeName := range timeTypes {
		if got := detectDatabaseType(typeName); got != "datetime" {
			t.Errorf("expected datetime for %q, got %s", typeName, got)
		}
	}

	// TINYINT should NOT be detected as time
	if got := detectDatabaseType("TINYINT"); got == "datetime" {
		t.Errorf("TINYINT should not be detected as datetime, got %s", got)
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

func TestGenDecimalRespectsPrecisionAndScale(t *testing.T) {
	faker := gofakeit.New(1)

	tests := []struct {
		name      string
		precision int
		scale     int
		maxVal    float64
	}{
		{"numeric(5,2)", 5, 2, 999.99},
		{"numeric(3,2)", 3, 2, 9.99},
		{"numeric(10,2)", 10, 2, 99999999.99},
		{"numeric(2,2)", 2, 2, 0.99},
		{"numeric(4,0)", 4, 0, 9999},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := map[string]any{
				"precision": tt.precision,
				"scale":     tt.scale,
			}

			for i := 0; i < 100; i++ {
				val := genDecimal(constraints, faker)
				f, ok := val.(float64)
				if !ok {
					t.Fatalf("expected float64, got %T", val)
				}
				if f < 0 || f > tt.maxVal {
					t.Fatalf("iteration %d: value %f exceeds max %f for %s", i, f, tt.maxVal, tt.name)
				}
			}
		})
	}
}

func TestGenDecimalAcceptsPrecisionAsInt64(t *testing.T) {
	faker := gofakeit.New(1)

	// precision as int64 (the original code path)
	constraints := map[string]any{
		"precision": int64(5),
		"scale":     2,
	}

	for i := 0; i < 50; i++ {
		val := genDecimal(constraints, faker)
		f, ok := val.(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", val)
		}
		if f > 999.99 {
			t.Fatalf("value %f exceeds max 999.99 for numeric(5,2) with int64 precision", f)
		}
	}
}

func TestGenDecimalDefaultsWithoutPrecision(t *testing.T) {
	faker := gofakeit.New(1)

	// No precision → default max 1000.0, scale 2
	for i := 0; i < 50; i++ {
		val := genDecimal(nil, faker)
		f, ok := val.(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", val)
		}
		if f < 0 || f > 1000.0 {
			t.Fatalf("value %f outside default range [0, 1000]", f)
		}
		// Verify scale=2: at most 2 decimal places
		str := strconv.FormatFloat(f, 'f', -1, 64)
		if idx := strings.Index(str, "."); idx >= 0 {
			decimals := len(str) - idx - 1
			if decimals > 2 {
				t.Fatalf("expected at most 2 decimal places, got %d in %s", decimals, str)
			}
		}
	}
}

func TestGenDecimalRespectsCheckMinMax(t *testing.T) {
	faker := gofakeit.New(1)

	constraints := map[string]any{
		"check_min": 10.0,
		"check_max": 20.0,
	}

	for i := 0; i < 100; i++ {
		val := genDecimal(constraints, faker)
		f, ok := val.(float64)
		if !ok {
			t.Fatalf("expected float64, got %T", val)
		}
		if f < 10.0 || f > 20.0 {
			t.Fatalf("value %f outside constrained range [10, 20]", f)
		}
	}
}

func TestColumnPrecisionScaleFlowsToGenDecimal(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1)

	precision := 5
	scale := 2
	columns := []engine.Column{
		{Name: "amount", Type: "numeric", Precision: &precision, Scale: &scale},
	}
	constraints := map[string]map[string]any{
		"amount": {"nullable": false},
	}

	maxAllowed := math.Pow(10, float64(precision-scale)) - math.Pow(10, -float64(scale))

	for i := 0; i < 100; i++ {
		records, err := g.GenerateRowDataWithConstraints(columns, constraints)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		rec := findRecord(records, "amount")
		if rec == nil {
			t.Fatalf("iteration %d: expected amount record", i)
		}
		f, err := strconv.ParseFloat(rec.Value, 64)
		if err != nil {
			t.Fatalf("iteration %d: failed to parse %q as float: %v", i, rec.Value, err)
		}
		if f > maxAllowed {
			t.Fatalf("iteration %d: value %f exceeds max %f for numeric(%d,%d)", i, f, maxAllowed, precision, scale)
		}
	}
}

func TestGenIntRespectsConstraints(t *testing.T) {
	faker := gofakeit.New(1)

	tests := []struct {
		name    string
		typ     string
		min     float64
		max     float64
		wantMin int
		wantMax int
	}{
		{"check_min and check_max", "integer", 5, 10, 5, 10},
		{"smallint default range", "smallint", 0, 0, 1, 32767},
		{"tinyint default range", "tinyint", 0, 0, 1, 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var constraints map[string]any
			if tt.min != 0 || tt.max != 0 {
				constraints = map[string]any{}
				if tt.min != 0 {
					constraints["check_min"] = tt.min
				}
				if tt.max != 0 {
					constraints["check_max"] = tt.max
				}
			}

			wantMin := tt.wantMin
			wantMax := tt.wantMax
			if tt.min != 0 {
				wantMin = int(tt.min)
			}
			if tt.max != 0 {
				wantMax = int(tt.max)
			}

			for i := 0; i < 100; i++ {
				val := genInt(tt.typ, constraints, faker)
				n, ok := val.(int)
				if !ok {
					t.Fatalf("expected int, got %T", val)
				}
				if n < wantMin || n > wantMax {
					t.Fatalf("iteration %d: value %d outside range [%d, %d]", i, n, wantMin, wantMax)
				}
			}
		})
	}
}

func TestComputedColumnsAreSkipped(t *testing.T) {
	g := NewGenerator(0)
	g.faker = gofakeit.New(1)

	columns := []engine.Column{
		{Name: "id", Type: "integer", IsPrimary: true, IsAutoIncrement: true},
		{Name: "price", Type: "numeric"},
		{Name: "tax", Type: "numeric", IsComputed: true},
		{Name: "total", Type: "numeric", IsComputed: true},
	}
	constraints := map[string]map[string]any{
		"price": {"nullable": false},
	}

	records, err := g.GenerateRowDataWithConstraints(columns, constraints)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, r := range records {
		if r.Key == "id" || r.Key == "tax" || r.Key == "total" {
			t.Fatalf("expected column %q to be skipped (auto-increment or computed)", r.Key)
		}
	}

	if findRecord(records, "price") == nil {
		t.Fatal("expected price column to be present")
	}
}

func TestGenTextRespectsLength(t *testing.T) {
	faker := gofakeit.New(1)

	lengths := []int{3, 10, 50, 255}
	for _, maxLen := range lengths {
		constraints := map[string]any{"length": maxLen}
		for i := 0; i < 20; i++ {
			val := genText(constraints, faker)
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if len(str) > maxLen {
				t.Fatalf("length %d: text %q exceeds max length %d", len(str), str, maxLen)
			}
		}
	}
}

func TestGenUintRespectsConstraints(t *testing.T) {
	faker := gofakeit.New(1)

	constraints := map[string]any{
		"check_min": float64(100),
		"check_max": float64(200),
	}

	for i := 0; i < 100; i++ {
		val := genUint("uint32", constraints, faker)
		n, ok := val.(uint)
		if !ok {
			t.Fatalf("expected uint, got %T", val)
		}
		if n < 100 || n > 200 {
			t.Fatalf("iteration %d: value %d outside range [100, 200]", i, n)
		}
	}
}

func TestGenDateFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genDate(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		parsed, err := time.Parse("2006-01-02", str)
		if err != nil {
			t.Fatalf("iteration %d: failed to parse date %q: %v", i, str, err)
		}
		if parsed.After(time.Now()) || parsed.Before(time.Now().AddDate(-10, 0, 0)) {
			t.Fatalf("iteration %d: date %s outside expected 10-year range", i, str)
		}
	}
}

func TestGenDateTimeFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genDateTime(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		parsed, err := time.Parse("2006-01-02 15:04:05", str)
		if err != nil {
			t.Fatalf("iteration %d: failed to parse datetime %q: %v", i, str, err)
		}
		if parsed.After(time.Now()) || parsed.Before(time.Now().AddDate(-10, 0, 0)) {
			t.Fatalf("iteration %d: datetime %s outside expected 10-year range", i, str)
		}
	}
}

func TestGenTimeFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genTime(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if _, err := time.Parse("15:04:05", str); err != nil {
			t.Fatalf("iteration %d: failed to parse time %q: %v", i, str, err)
		}
	}
}

func TestGenYearRange(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genYear(faker)
		n, ok := val.(int)
		if !ok {
			t.Fatalf("expected int, got %T", val)
		}
		if n < 1970 || n > time.Now().Year() {
			t.Fatalf("iteration %d: year %d outside range [1970, %d]", i, n, time.Now().Year())
		}
	}
}

func TestGenIntervalFormat(t *testing.T) {
	faker := gofakeit.New(1)
	validUnits := map[string]bool{
		"seconds": true, "minutes": true, "hours": true,
		"days": true, "weeks": true, "months": true, "years": true,
	}

	pattern := regexp.MustCompile(`^(\d+)\s+(\w+)$`)
	for i := 0; i < 20; i++ {
		val := genInterval(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		matches := pattern.FindStringSubmatch(str)
		if matches == nil {
			t.Fatalf("iteration %d: interval %q doesn't match expected format", i, str)
		}
		num, _ := strconv.Atoi(matches[1])
		if num < 1 || num > 30 {
			t.Fatalf("iteration %d: interval value %d outside range [1, 30]", i, num)
		}
		if !validUnits[matches[2]] {
			t.Fatalf("iteration %d: invalid interval unit %q", i, matches[2])
		}
	}
}

func TestGenIPv4Format(t *testing.T) {
	faker := gofakeit.New(1)
	ipv4Pattern := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

	for i := 0; i < 20; i++ {
		val := genIPv4(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !ipv4Pattern.MatchString(str) {
			t.Fatalf("iteration %d: %q is not a valid IPv4 format", i, str)
		}
	}
}

func TestGenIPv6Format(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genIPv6(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !strings.Contains(str, ":") {
			t.Fatalf("iteration %d: %q doesn't look like an IPv6 address", i, str)
		}
	}
}

func TestGenMACAddrFormat(t *testing.T) {
	faker := gofakeit.New(1)
	macPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`)

	for i := 0; i < 20; i++ {
		val := genMACAddr(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !macPattern.MatchString(str) {
			t.Fatalf("iteration %d: %q is not a valid MAC address format", i, str)
		}
	}
}

func TestGenPointFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genPoint(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !strings.HasPrefix(str, "(") || !strings.HasSuffix(str, ")") {
			t.Fatalf("iteration %d: point %q not wrapped in parens", i, str)
		}
		if !strings.Contains(str, ",") {
			t.Fatalf("iteration %d: point %q missing comma separator", i, str)
		}
	}
}

func TestGenGeometryFormats(t *testing.T) {
	faker := gofakeit.New(1)

	tests := []struct {
		typeName string
		contains string
	}{
		{"line", "{"},
		{"lseg", "[("},
		{"box", "("},
		{"path", "[("},
		{"polygon", "(("},
		{"circle", "<("},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			val := genGeometry(tt.typeName, faker)
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.Contains(str, tt.contains) {
				t.Fatalf("%s: expected output to contain %q, got %q", tt.typeName, tt.contains, str)
			}
		})
	}
}

func TestGenSpatialFormats(t *testing.T) {
	faker := gofakeit.New(1)

	tests := []struct {
		typeName string
		prefix   string
	}{
		{"geometry", "POINT("},
		{"geography", "POINT("},
		{"linestring", "LINESTRING("},
		{"multipoint", "MULTIPOINT("},
		{"multilinestring", "MULTILINESTRING("},
		{"multipolygon", "MULTIPOLYGON("},
		{"geometrycollection", "GEOMETRYCOLLECTION("},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			val := genSpatial(tt.typeName, faker)
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, tt.prefix) {
				t.Fatalf("%s: expected prefix %q, got %q", tt.typeName, tt.prefix, str)
			}
		})
	}
}

func TestGenClickHouseDecimalFormats(t *testing.T) {
	faker := gofakeit.New(1)

	types := []string{"decimal32", "decimal64", "decimal128", "decimal256"}
	for _, typeName := range types {
		t.Run(typeName, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				val := genClickHouseDecimal(typeName, faker)
				str, ok := val.(string)
				if !ok {
					t.Fatalf("expected string, got %T", val)
				}
				f, err := strconv.ParseFloat(str, 64)
				if err != nil {
					t.Fatalf("iteration %d: failed to parse %q as float: %v", i, str, err)
				}
				if f < 0 {
					t.Fatalf("iteration %d: value %f should be non-negative", i, f)
				}
				if !strings.Contains(str, ".") {
					t.Fatalf("iteration %d: expected decimal point in %q", i, str)
				}
			}
		})
	}
}

func TestGenXMLFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genXML(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !strings.HasPrefix(str, "<data>") || !strings.HasSuffix(str, "</data>") {
			t.Fatalf("iteration %d: XML %q not wrapped in <data> tags", i, str)
		}
		if !strings.Contains(str, "<id>") || !strings.Contains(str, "<value>") {
			t.Fatalf("iteration %d: XML %q missing expected child elements", i, str)
		}
	}
}

func TestGenJSONFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genJSON(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !strings.HasPrefix(str, "{") || !strings.HasSuffix(str, "}") {
			t.Fatalf("iteration %d: JSON %q not a valid object", i, str)
		}
		for _, key := range []string{"id", "name", "email", "active"} {
			if !strings.Contains(str, fmt.Sprintf("%q", key)) {
				t.Fatalf("iteration %d: JSON missing key %q: %s", i, key, str)
			}
		}
	}
}

func TestGenHstoreFormat(t *testing.T) {
	faker := gofakeit.New(1)

	for i := 0; i < 20; i++ {
		val := genHstore(faker)
		str, ok := val.(string)
		if !ok {
			t.Fatalf("expected string, got %T", val)
		}
		if !strings.Contains(str, "=>") {
			t.Fatalf("iteration %d: hstore %q missing => separator", i, str)
		}
	}
}

func TestGenBinaryFormat(t *testing.T) {
	faker := gofakeit.New(1)

	t.Run("default length", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			val := genBinary(nil, faker)
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "0x") {
				t.Fatalf("iteration %d: binary %q missing 0x prefix", i, str)
			}
		}
	})

	t.Run("custom length", func(t *testing.T) {
		constraints := map[string]any{"length": 4}
		for i := 0; i < 20; i++ {
			val := genBinary(constraints, faker)
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "0x") {
				t.Fatalf("iteration %d: binary %q missing 0x prefix", i, str)
			}
			// 0x prefix + 2 hex chars per byte = 2 + 4*2 = 10
			hexPart := str[2:]
			if len(hexPart) != 8 {
				t.Fatalf("iteration %d: expected 8 hex chars for 4 bytes, got %d in %q", i, len(hexPart), str)
			}
		}
	})
}

func TestGenArrayFormats(t *testing.T) {
	faker := gofakeit.New(1)

	types := []string{"text[]", "int[]", "integer[]", "varchar[]"}
	for _, dbType := range types {
		t.Run(dbType, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				val := genArray(dbType, faker)
				str, ok := val.(string)
				if !ok {
					t.Fatalf("expected string, got %T", val)
				}
				if !strings.HasPrefix(str, "{") || !strings.HasSuffix(str, "}") {
					t.Fatalf("iteration %d: array %q not wrapped in braces", i, str)
				}
			}
		})
	}
}

func TestGenerateByTypeRoutesToCorrectGenerator(t *testing.T) {
	faker := gofakeit.New(1)

	tests := []struct {
		dbType   string
		validate func(t *testing.T, val any)
	}{
		{"integer", func(t *testing.T, val any) {
			if _, ok := val.(int); !ok {
				t.Fatalf("expected int, got %T", val)
			}
		}},
		{"boolean", func(t *testing.T, val any) {
			if _, ok := val.(bool); !ok {
				t.Fatalf("expected bool, got %T", val)
			}
		}},
		{"uuid", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if len(strings.Split(str, "-")) != 5 {
				t.Fatalf("expected UUID format, got %q", str)
			}
		}},
		{"date", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if _, err := time.Parse("2006-01-02", str); err != nil {
				t.Fatalf("expected date format, got %q", str)
			}
		}},
		{"json", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "{") {
				t.Fatalf("expected JSON object, got %q", str)
			}
		}},
		{"inet", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.Contains(str, ".") {
				t.Fatalf("expected IPv4 format, got %q", str)
			}
		}},
		{"macaddr", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.Contains(str, ":") {
				t.Fatalf("expected MAC address format, got %q", str)
			}
		}},
		{"numeric", func(t *testing.T, val any) {
			f, ok := val.(float64)
			if !ok {
				t.Fatalf("expected float64, got %T", val)
			}
			if f < 0 || f > 1000 {
				t.Fatalf("expected value in default range, got %f", f)
			}
		}},
		{"money", func(t *testing.T, val any) {
			if _, ok := val.(float64); !ok {
				t.Fatalf("expected float64 for money, got %T", val)
			}
		}},
		{"bytea", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "0x") {
				t.Fatalf("expected hex prefix, got %q", str)
			}
		}},
		{"xml", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "<") {
				t.Fatalf("expected XML, got %q", str)
			}
		}},
		{"hstore", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.Contains(str, "=>") {
				t.Fatalf("expected hstore format, got %q", str)
			}
		}},
		{"point", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "(") {
				t.Fatalf("expected point format, got %q", str)
			}
		}},
		{"text[]", func(t *testing.T, val any) {
			str, ok := val.(string)
			if !ok {
				t.Fatalf("expected string, got %T", val)
			}
			if !strings.HasPrefix(str, "{") {
				t.Fatalf("expected array format, got %q", str)
			}
		}},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			val := GenerateByType(tt.dbType, "", nil, faker)
			tt.validate(t, val)
		})
	}
}

func TestGenDecimalScaleRounding(t *testing.T) {
	faker := gofakeit.New(1)

	scales := []int{0, 1, 3, 4}
	for _, scale := range scales {
		t.Run(fmt.Sprintf("scale_%d", scale), func(t *testing.T) {
			constraints := map[string]any{"scale": scale}
			for i := 0; i < 50; i++ {
				val := genDecimal(constraints, faker)
				f, ok := val.(float64)
				if !ok {
					t.Fatalf("expected float64, got %T", val)
				}
				// Verify rounding: multiply by 10^scale, result should be integer
				multiplier := math.Pow(10, float64(scale))
				rounded := math.Round(f * multiplier)
				if math.Abs(f*multiplier-rounded) > 1e-9 {
					t.Fatalf("iteration %d: value %f has more than %d decimal places", i, f, scale)
				}
			}
		})
	}
}
