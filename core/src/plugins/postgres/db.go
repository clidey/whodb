package postgres

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	portKey = "Port"
)

func escape(x string) string {
	return strings.ReplaceAll(x, "'", "\\'")
}

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "5432"))
	if err != nil {
		return nil, err
	}
	host := escape(config.Credentials.Hostname)
	username := escape(config.Credentials.Username)
	password := escape(config.Credentials.Password)
	database := escape(config.Credentials.Database)

	params := strings.Builder{}
	if config.Credentials.IsProfile {
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey:
				continue
			default:
				params.WriteString(fmt.Sprintf("%v='%v' ", record.Key, escape(record.Value)))
			}
		}
	}

	dsn := fmt.Sprintf("host='%v' user='%v' password='%v' dbname='%v' port='%v'",
		host, username, password, database, port)

	if params.Len() > 0 {
		dsn = fmt.Sprintf("%v %v", dsn, params.String())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
