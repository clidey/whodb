//go:build ee

package src

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/ee/core/src/plugins/dynamodb"
	"github.com/clidey/whodb/ee/core/src/plugins/mssql"
	"github.com/clidey/whodb/ee/core/src/plugins/oracle"
)

func init() {
	// Register EE plugins when building with -tags ee
	SetEEInitializer(func(e *engine.Engine) {
		e.RegistryPlugin(oracle.NewOraclePlugin())
		e.RegistryPlugin(mssql.NewMSSQLPlugin())
		e.RegistryPlugin(dynamodb.NewDynamoDBPlugin())
	})
}