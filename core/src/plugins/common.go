package plugins

import (
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/gorm"
)

type DBOperation[T any] func(*gorm.DB) (T, error)
type DBCreationFunc func(pluginConfig *engine.PluginConfig) (*gorm.DB, error)

// WithConnection handles database connection lifecycle and executes the operation
func WithConnection[T any](config *engine.PluginConfig, DB DBCreationFunc, operation DBOperation[T]) (T, error) {
	db, err := DB(config)
	if err != nil {
		var zero T
		return zero, err
	}
	sqlDb, err := db.DB()
	if err != nil {
		var zero T
		return zero, err
	}
	defer sqlDb.Close()
	return operation(db)
}
