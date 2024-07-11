package clickhouse

import (
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "9000")
	readTimeout := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Read Timeout", "10s")
	dsn := fmt.Sprintf("clickhouse://%v:%v@%v:%v/%v?read_timeout=%v", config.Credentials.Username, config.Credentials.Password, config.Credentials.Hostname, port, config.Credentials.Database, readTimeout)
	db, err := gorm.Open(clickhouse.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
