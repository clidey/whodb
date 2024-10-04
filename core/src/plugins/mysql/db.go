package mysql

import (
	"fmt"
	"net/url"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	portKey                    = "Port"
	charsetKey                 = "Charset"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306")
	charset := common.GetRecordValueOrDefault(config.Credentials.Advanced, charsetKey, "utf8mb4")
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True")
	loc := common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "Local")
	allowClearTextPasswords := common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0")

	params := url.Values{}

	for _, record := range config.Credentials.Advanced {
		switch record.Key {
		case portKey, charsetKey, parseTimeKey, locKey, allowClearTextPasswordsKey:
			continue
		default:
			params.Add(record.Key, fmt.Sprintf("%v", record.Value))
		}
	}

	dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=%v&parseTime=%v&loc=%v&allowCleartextPasswords=%v&%v", config.Credentials.Username, config.Credentials.Password, config.Credentials.Hostname, port, config.Credentials.Database, charset, parseTime, loc, allowClearTextPasswords, params.Encode())
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
