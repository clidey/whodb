package mysql

import (
	"net"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	portKey                    = "Port"
	collationKey               = "Collation"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
	hostPathKey                = "Host path"
)

// todo: https://github.com/go-playground/validator
// todo: convert below to their respective types before passing into the configuration. check if it can be done before coming here

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port := common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306")
	collation := common.GetRecordValueOrDefault(config.Credentials.Advanced, collationKey, "utf8mb4_general_ci")
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True")
	loc, err := time.LoadLocation(common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "Local"))
	if err != nil {
		return nil, err
	}
	allowClearTextPasswords := common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0")
	hostPath := common.GetRecordValueOrDefault(config.Credentials.Advanced, hostPathKey, "")

	mysqlConfig := mysqldriver.Config{
		User:                    config.Credentials.Username,
		Passwd:                  config.Credentials.Password,
		Net:                     "tcp",
		Addr:                    net.JoinHostPort(config.Credentials.Hostname, port),
		DBName:                  config.Credentials.Database,
		Params:                  make(map[string]string),
		AllowCleartextPasswords: allowClearTextPasswords == "1",
		ParseTime:               parseTime == "True",
		Loc:                     loc,
		Collation:               collation,
	}

	// if there is a hostPath presented, it takes priority over the Hostname
	// todo: reflect this in the ui with a popup or something
	if strings.HasPrefix(hostPath, "/") {
		mysqlConfig.Net = "unix"
		mysqlConfig.Addr = hostPath
	}

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
