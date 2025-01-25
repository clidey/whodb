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
	//hostPathKey                = "Host path"
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

	// collation cannot contain & characters by default, so we remove them
	collation = strings.ReplaceAll(collation, "&", "")

	mysqlConfig := mysqldriver.Config{
		User:                    config.Credentials.Username,
		Passwd:                  config.Credentials.Password,
		Net:                     "tcp",
		Addr:                    net.JoinHostPort(config.Credentials.Hostname, port),
		DBName:                  config.Credentials.Database,
		AllowCleartextPasswords: allowClearTextPasswords == "1",
		ParseTime:               strings.ToLower(parseTime) == "true",
		Loc:                     loc,
		Collation:               collation,
	}

	// if this config is a pre-configured profile, then allow reading of additional params
	if config.Credentials.IsProfile {
		params := make(map[string]string)
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey, collationKey, parseTimeKey, locKey, allowClearTextPasswordsKey:
				continue
			default:
				params[record.Key] = record.Value
			}
		}
		mysqlConfig.Params = params
	}

	db, err := gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
