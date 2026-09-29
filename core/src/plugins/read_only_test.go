package plugins

import (
	"testing"

	"gorm.io/gorm"

	"github.com/clidey/whodb/core/src/engine"
)

func TestProtectedQueryCannotJoinTransaction(t *testing.T) {
	config := engine.NewPluginConfig(&engine.Credentials{Type: "Postgres"})
	config.ReadOnly = true
	config.Transaction = &gorm.DB{}
	_, err := WithConnection(config, func(*engine.PluginConfig) (*gorm.DB, error) { t.Fatal("opened connection"); return nil, nil }, func(*gorm.DB) (bool, error) { t.Fatal("joined writable transaction"); return true, nil })
	if err == nil {
		t.Fatal("expected existing transaction to be rejected")
	}
}
