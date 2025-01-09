package keeper_integration

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/log"
	"github.com/keeper-security/secrets-manager-go/core"
)

func GetLoginProfiles() ([]env.DatabaseCredentials, error) {
	profiles := []env.DatabaseCredentials{}
	if len(env.KeeperToken) == 0 {
		return profiles, nil
	}

	configBytes, err := base64.StdEncoding.DecodeString(env.KeeperToken)
	if err != nil {
		log.Logger.Fatalf("Failed to decode Base64 configuration: %v", err)
	}

	var config map[string]interface{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Logger.Fatalf("Failed to parse JSON configuration: %v", err)
	}

	keeperConfig := make(map[string]string)
	for key, value := range config {
		if strValue, ok := value.(string); ok {
			keeperConfig[key] = strValue
		} else {
			log.Logger.Fatalf("Invalid configuration value for key '%s': expected string, got %T", key, value)
		}
	}

	clientConfig := &core.ClientOptions{
		Config: core.NewMemoryKeyValueStorage(keeperConfig),
	}

	client := core.NewSecretsManager(clientConfig)

	secrets, err := client.GetSecrets([]string{})
	if err != nil {
		log.Logger.Fatalf("Failed to fetch secret: %v", err)
	}

	type PamUser struct {
		User     string
		Password string
	}

	pamDatabaseUsers := map[string]PamUser{}
	for _, secret := range secrets {
		recordType := secret.RecordDict["type"]
		if recordType != "pamUser" {
			continue
		}

		fields, ok := secret.RecordDict["fields"].([]interface{})
		if !ok {
			log.Logger.Warn("Failed to parse 'fields'")
			continue
		}
		pamDatabaseUser := PamUser{}
		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			if fieldMap["type"] == "login" {
				pamDatabaseUser.User = fieldMap["value"].([]interface{})[0].(string)
			} else if fieldMap["type"] == "password" {
				passwords := fieldMap["value"].([]interface{})
				if len(passwords) > 0 {
					pamDatabaseUser.Password = passwords[0].(string)
				}
			}
		}
		pamDatabaseUsers[secret.Uid] = pamDatabaseUser
	}

	for _, secret := range secrets {
		recordUid := secret.Uid
		recordTitle := secret.RecordDict["title"].(string)
		recordType := secret.RecordDict["type"].(string)
		if recordType != "pamDatabase" {
			continue
		}

		fields, ok := secret.RecordDict["fields"].([]interface{})
		if !ok {
			log.Logger.Warn("Failed to parse 'fields'")
			continue
		}
		credentials := env.DatabaseCredentials{
			CustomId: recordUid,
			Alias:    recordTitle,
			Source:   env.Keeper_Source,
		}
		for _, field := range fields {
			fieldMap, ok := field.(map[string]interface{})
			if !ok {
				continue
			}

			if fieldMap["type"] == "pamSettings" {
				valueArray, ok := fieldMap["value"].([]interface{})
				if !ok || len(valueArray) == 0 {
					log.Logger.Warnf("Invalid 'value' field in pamSettings: %v", fieldMap["value"])
					continue
				}

				valueMap, ok := valueArray[0].(map[string]interface{})
				if !ok {
					log.Logger.Warnf("Invalid first element in 'value' array: %v", valueArray[0])
					continue
				}

				connection, ok := valueMap["connection"].(map[string]interface{})
				if !ok {
					log.Logger.Warnf("Invalid 'connection' field: %v", valueMap["connection"])
					continue
				}

				database, _ := connection["database"].(string)
				protocol, ok := connection["protocol"]
				if !ok {
					continue
				}

				credentials.Type = getDatabaseTypeFromProtocol(protocol.(string))
				userRecordUids, _ := connection["userRecords"].([]interface{})

				for _, uid := range userRecordUids {
					uidStr, ok := uid.(string)
					if !ok {
						log.Logger.Warnf("Invalid userRecord UID: %v", uid)
						continue
					}

					userRecord, exists := pamDatabaseUsers[uidStr]
					if !exists {
						log.Logger.Warnf("UserRecord UID not found: %s", uidStr)
						continue
					}

					credentials.Database = database
					credentials.Username = userRecord.User
					credentials.Password = userRecord.Password
				}
			} else if fieldMap["type"] == "pamHostname" {
				valueArray, ok := fieldMap["value"].([]interface{})
				if !ok || len(valueArray) == 0 {
					log.Logger.Warnf("Invalid 'value' field in pamSettings: %v", fieldMap["value"])
					continue
				}

				valueMap, ok := valueArray[0].(map[string]interface{})
				if !ok {
					log.Logger.Warnf("Invalid first element in 'value' array: %v", valueArray[0])
					continue
				}

				hostName := valueMap["hostName"].(string)
				port := valueMap["port"].(string)
				credentials.Hostname = hostName
				credentials.Port = port
			}
		}

		if len(credentials.Type) > 0 {
			profiles = append(profiles, credentials)
		}
	}

	return profiles, nil
}

func getDatabaseTypeFromProtocol(title string) string {
	title = strings.ToLower(title)

	databaseTypes := map[string]string{
		"postgres": "Postgres",
		"mysql":    "MySQL",
		"mariadb":  "MariaDB",
		"sqlite":   "Sqlite3",
		"mongo":    "MongoDB",
	}

	for keyword, dbType := range databaseTypes {
		if strings.Contains(title, keyword) {
			return dbType
		}
	}

	return "Unknown"
}
