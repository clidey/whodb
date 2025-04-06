package gorm_plugin

import (
	"errors"
	"fmt"
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) DeleteRow(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	return plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
		pkColumns, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			pkColumns = []string{}
		}

		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			return false, err
		}

		conditions := make(map[string]interface{})
		convertedValues := make(map[string]interface{})
		for column, strValue := range values {
			columnType, exists := columnTypes[column]
			if !exists {
				return false, fmt.Errorf("column '%s' does not exist in table %s", column, storageUnit)
			}

			convertedValue, err := p.ConvertStringValue(strValue, columnType)
			if err != nil {
				return false, fmt.Errorf("failed to convert value for column '%s': %v", column, err)
			}

			if common.ContainsString(pkColumns, column) {
				conditions[column] = convertedValue
			} else {
				convertedValues[column] = convertedValue
			}
		}

		schema = p.EscapeIdentifier(schema)
		storageUnit = p.EscapeIdentifier(storageUnit)
		tableName := p.FormTableName(schema, storageUnit)

		var result *gorm.DB
		if len(conditions) == 0 {
			result = db.Table(tableName).Where(convertedValues).Delete(convertedValues)
		} else {
			result = db.Table(tableName).Where(conditions).Delete(conditions)
		}

		if result.Error != nil {
			return false, result.Error
		}

		// todo: investigate why the clickhouse driver doesnt show any updated rows after a delete
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were deleted")
		}

		return true, nil
	})
}
