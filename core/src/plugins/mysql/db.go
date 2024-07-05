package mysql

import (
	"fmt"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Port", "3306")
	charset := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Charset", "utf8mb4")
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Parse Time", "%v")
	loc := common.GetRecordValueOrDefault(config.Credentials.Advanced, "Loc", "%v")
	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=%v&parseTime=%v&loc=%v", config.Credentials.Username, config.Credentials.Password, config.Credentials.Hostname, config.Credentials.Database, port, charset, parseTime, loc)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
