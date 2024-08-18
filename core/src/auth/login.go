package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src/common"
)

func Login(ctx context.Context, input *model.LoginCredentials) (*model.StatusResponse, error) {
	loginInfoJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	tokenValue := base64.StdEncoding.EncodeToString(loginInfoJSON)

	tokenCookie := &http.Cookie{
		Name:     string(AuthKey_Token),
		Value:    tokenValue,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), tokenCookie)

	var profiles []model.LoginCredentials
	profilesCookie, err := ctx.Value(common.RouterKey_Request).(*http.Request).Cookie(string(AuthKey_Profiles))
	if err == nil {
		decodedProfiles, err := base64.StdEncoding.DecodeString(profilesCookie.Value)
		if err == nil {
			json.Unmarshal(decodedProfiles, &profiles)
		}
	}

	profiles = append(profiles, *input)

	profiles = removeDuplicateProfiles(profiles)

	profilesJSON, err := json.Marshal(profiles)
	if err != nil {
		return nil, err
	}

	profilesValue := base64.StdEncoding.EncodeToString(profilesJSON)

	profilesCookie = &http.Cookie{
		Name:     string(AuthKey_Profiles),
		Value:    profilesValue,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	http.SetCookie(ctx.Value(common.RouterKey_ResponseWriter).(http.ResponseWriter), profilesCookie)

	return &model.StatusResponse{
		Status: true,
	}, nil
}

func removeDuplicateProfiles(profiles []model.LoginCredentials) []model.LoginCredentials {
	uniqueProfiles := make([]model.LoginCredentials, 0)
	profileMap := make(map[string]bool)

	for _, profile := range profiles {
		key := generateProfileKey(profile)
		if !profileMap[key] {
			uniqueProfiles = append(uniqueProfiles, profile)
			profileMap[key] = true
		}
	}

	return uniqueProfiles
}

func generateProfileKey(profile model.LoginCredentials) string {
	return profile.Type + profile.Hostname + profile.Username + profile.Database
}
