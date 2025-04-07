package postgres

import (
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func escape(x string) string {
	return strings.ReplaceAll(x, "'", "\\'")
}

func (p *PostgresPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}

	host := escape(connectionInput.Hostname)
	username := escape(connectionInput.Username)
	password := escape(connectionInput.Password)
	database := escape(connectionInput.Database)

	params := strings.Builder{}
	if connectionInput.ExtraOptions != nil {
		for _, record := range config.Credentials.Advanced {
			params.WriteString(fmt.Sprintf("%v='%v' ", record.Key, escape(record.Value)))
		}
	}

	dsn := fmt.Sprintf("host='%v' user='%v' password='%v' dbname='%v' port='%v'",
		host, username, password, database, connectionInput.Port)

	if params.Len() > 0 {
		dsn = fmt.Sprintf("%v %v", dsn, params.String())
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
