package mysql

import (
	"net"
	"net/url"
	"strconv"
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
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
)

// todo: https://github.com/go-playground/validator
// todo: convert below to their respective types before passing into the configuration. check if it can be done before coming here

func DB(config *engine.PluginConfig) (*gorm.DB, error) {
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306"))
	if err != nil {
		return nil, err
	}
	parseTime := common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True")
	loc, err := time.LoadLocation(common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "UTC"))
	if err != nil {
		return nil, err
	}
	allowClearTextPasswords := common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0")

	mysqlConfig := mysqldriver.NewConfig()
	mysqlConfig.User = config.Credentials.Username
	mysqlConfig.Passwd = config.Credentials.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = net.JoinHostPort(config.Credentials.Hostname, strconv.Itoa(port))
	mysqlConfig.DBName = config.Credentials.Database
	mysqlConfig.AllowCleartextPasswords = allowClearTextPasswords == "1"
	mysqlConfig.ParseTime = strings.ToLower(parseTime) == "true"
	mysqlConfig.Loc = loc

	// if this config is a pre-configured profile, then allow reading of additional params
	if config.Credentials.IsProfile {
		params := make(map[string]string)
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey, parseTimeKey, locKey, allowClearTextPasswordsKey:
				continue
			default:
				params[record.Key] = url.QueryEscape(record.Value)
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
