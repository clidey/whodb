package gorm_plugin

import (
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
)

func (p *GormPlugin) UpdateStorageUnit(config *engine.PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) (bool, error) {
		pkColumns, err := p.GetPrimaryKeyColumns(db, schema, storageUnit)
		if err != nil {
			return false, err
		}

		columnTypes, err := p.GetColumnTypes(db, schema, storageUnit)
		if err != nil {
			return false, err
		}

		conditions := make(map[string]any)
		convertedValues := make(map[string]any)
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

		tableName := p.FormTableName(schema, storageUnit)

		result := db.Table(tableName).Where(conditions).Updates(convertedValues)

		if result.Error != nil {
			return false, result.Error
		}

		// todo: investigate why the clickhouse driver doesnt show any updated rows after an update
		if p.Type != engine.DatabaseType_ClickHouse && result.RowsAffected == 0 {
			return false, errors.New("no rows were updated")
		}

		return true, nil
	})
}
