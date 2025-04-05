package gorm_plugin

import (
	"github.com/clidey/whodb/core/src/common"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins"
	"gorm.io/gorm"
	"net/url"
	"strconv"
	"time"
)

const (
	portKey                    = "Port"
	parseTimeKey               = "Parse Time"
	locKey                     = "Loc"
	allowClearTextPasswordsKey = "Allow clear text passwords"
	sslModeKey                 = "SSL Mode"
	httpProtocolKey            = "HTTP Protocol"
	readOnlyKey                = "Readonly"
	debugKey                   = "Debug"
	connectionTimeoutKey       = "Connection Timeout"
)

type ConnectionInput struct {
	//common
	Username string `validate:"required"`
	Password string `validate:"required"`
	Database string `validate:"required"`
	Hostname string `validate:"required"`
	Port     int    `validate:"required"`

	//mysql/mariadb
	ParseTime               bool           `validate:"boolean"`
	Loc                     *time.Location `validate:"required"`
	AllowClearTextPasswords bool           `validate:"boolean"`

	//clickhouse
	SSLMode      string
	HTTPProtocol string
	ReadOnly     string
	Debug        string

	ConnectionTimeout int

	ExtraOptions map[string]string `validate:"omitnil"`
}

func (p *GormPlugin) ParseConnectionConfig(config *engine.PluginConfig) (*ConnectionInput, error) {
	//common
	port, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, portKey, "3306"))
	if err != nil {
		return nil, err
	}

	//mysql/mariadb specific
	parseTime, err := strconv.ParseBool(common.GetRecordValueOrDefault(config.Credentials.Advanced, parseTimeKey, "True"))
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(common.GetRecordValueOrDefault(config.Credentials.Advanced, locKey, "Local"))
	if err != nil {
		return nil, err
	}
	allowClearTextPasswords, err := strconv.ParseBool(common.GetRecordValueOrDefault(config.Credentials.Advanced, allowClearTextPasswordsKey, "0"))
	if err != nil {
		return nil, err
	}

	//clickhouse specific
	sslMode := common.GetRecordValueOrDefault(config.Credentials.Advanced, sslModeKey, "disable")
	httpProtocol := common.GetRecordValueOrDefault(config.Credentials.Advanced, httpProtocolKey, "disable")
	readOnly := common.GetRecordValueOrDefault(config.Credentials.Advanced, readOnlyKey, "disable")
	debug := common.GetRecordValueOrDefault(config.Credentials.Advanced, debugKey, "disable")

	connectionTimeout, err := strconv.Atoi(common.GetRecordValueOrDefault(config.Credentials.Advanced, connectionTimeoutKey, "90"))
	if err != nil {
		return nil, err
	}

	input := &ConnectionInput{
		Username:                url.PathEscape(config.Credentials.Username),
		Password:                url.PathEscape(config.Credentials.Password),
		Database:                url.PathEscape(config.Credentials.Database),
		Hostname:                url.PathEscape(config.Credentials.Hostname),
		Port:                    port,
		ParseTime:               parseTime,
		Loc:                     loc,
		AllowClearTextPasswords: allowClearTextPasswords,
		SSLMode:                 sslMode,
		HTTPProtocol:            httpProtocol,
		ReadOnly:                readOnly,
		Debug:                   debug,
		ConnectionTimeout:       connectionTimeout,
	}

	// if this config is a pre-configured profile, then allow reading of additional params
	if config.Credentials.IsProfile {
		params := make(map[string]string)
		for _, record := range config.Credentials.Advanced {
			switch record.Key {
			case portKey, parseTimeKey, locKey, allowClearTextPasswordsKey, sslModeKey, httpProtocolKey, readOnlyKey, debugKey, connectionTimeoutKey:
				continue
			default:
				params[record.Key] = url.QueryEscape(record.Value) // todo: this may break for postgres
			}
		}
		input.ExtraOptions = params
	}

	return input, nil
}

func (p *GormPlugin) IsAvailable(config *engine.PluginConfig) bool {
	available, err := plugins.WithConnection[bool](config, p.DB, func(db *gorm.DB) (bool, error) {
		sqlDb, err := db.DB()
		if err != nil {
			return false, err
		}
		if err = sqlDb.Ping(); err != nil {
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return false
	}

	return available
}
