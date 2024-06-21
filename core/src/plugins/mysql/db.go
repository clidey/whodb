package mysql

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%v:%v@tcp(%v:3306)/%v?charset=utf8mb4&parseTime=True&loc=Local", config.Credentials.Username, config.Credentials.Password, config.Credentials.Hostname, config.Credentials.Database)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
