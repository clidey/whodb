package sqlite3

import (
	"errors"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
)

func (p *Sqlite3Plugin) AddStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	if len(fields) == 0 {
		return false, errors.New("no fields provided to create the table")
	}

	var columns []string
	for field, fieldType := range fields {
		columns = append(columns, fmt.Sprintf("%s %s", field, fieldType))
	}

	createTableSQL := fmt.Sprintf("CREATE TABLE %s (%s);", storageUnit, strings.Join(columns, ", "))

	if err := db.Exec(createTableSQL).Error; err != nil {
		return false, err
	}

	return true, nil
}

func (p *Sqlite3Plugin) AddRow(config *engine.PluginConfig, schema string, storageUnit string, values []engine.Record) (bool, error) {
	db, err := DB(config)
	if err != nil {
		return false, err
	}

	sqlDb, err := db.DB()
	if err != nil {
		return false, err
	}
	defer sqlDb.Close()

	if len(values) == 0 {
		return false, errors.New("no values provided to insert into the table")
	}

	columns := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	args := make([]interface{}, 0, len(values))

	for _, value := range values {
		columns = append(columns, value.Key)
		if value.Extra["Config"] == "sql" {
			placeholders = append(placeholders, value.Value)
		} else {
			placeholders = append(placeholders, fmt.Sprintf("CAST(? AS %v)", value.Extra["Type"]))
			args = append(args, value.Value)
		}
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s);",
		storageUnit,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	if err := db.Exec(insertSQL, args...).Error; err != nil {
		return false, err
	}

	return true, nil
}
