package keeper_integration

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/clidey/whodb/core/src/env"
	"github.com/keeper-security/secrets-manager-go/core"
)

func GetLoginProfiles() ([]env.DatabaseCredentials, error) {
	keeperToken := env.KeeperToken
	if keeperToken == "" {
		return nil, errors.New("keeper token is missing")
	}

	clientOptions := core.ClientOptions{
		Token: keeperToken,
	}
	secretsManager := core.NewSecretsManager(&clientOptions)

	records, err := secretsManager.GetSecrets(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secrets from Keeper: %v", err)
	}

	var profiles []env.DatabaseCredentials

	for _, record := range records {
		var credentials env.DatabaseCredentials
		for key, value := range record.RecordDict {
			switch key {
			case "host":
				credentials.Hostname = value.(string)
			case "user":
				credentials.Username = value.(string)
			case "password":
				credentials.Password = value.(string)
			case "database":
				credentials.Database = value.(string)
			case "port":
				credentials.Port = value.(string)
			case "type":
				credentials.Type = value.(string)
			case "config":
				configMap := make(map[string]string)
				err := json.Unmarshal([]byte(value.(string)), &configMap)
				if err == nil {
					credentials.Config = configMap
				}
			}
		}

		profiles = append(profiles, env.DatabaseCredentials{
			Hostname: credentials.Hostname,
			Username: credentials.Username,
			Password: credentials.Password,
			Database: credentials.Database,
			Port:     credentials.Port,
			Config:   credentials.Config,
			Type:     credentials.Type,
		})
	}

	return profiles, nil
}
