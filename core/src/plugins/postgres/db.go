package postgres

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	portKey    = "Port"
	sslModeKey = "SSL Mode"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "5432")
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslModeKey, "disable")

	params := strings.Builder{}

	for _, record := range config.Credentials.Advanced {
		switch record.Key {
		case portKey, sslModeKey:
			continue
		default:
			params.WriteString(fmt.Sprintf("%v=%v ", record.Key, record.Value))
		}
	}

	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=%v %v", config.Credentials.Hostname, config.Credentials.Username, config.Credentials.Password, config.Credentials.Database, port, sslMode, params.String())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
