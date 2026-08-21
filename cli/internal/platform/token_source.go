/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package platform

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const accessTokenSkew = time.Minute

// RefreshTokenStore persists the OIDC refresh token for one hosted account.
type RefreshTokenStore interface {
	GetPlatformRefreshToken(hostURL, accountID string) (string, error)
	SavePlatformRefreshToken(hostURL, accountID, refreshToken string) error
}

// AccessTokenSource supplies access tokens for authenticated platform requests.
type AccessTokenSource interface {
	Token(context.Context) (string, error)
	Invalidate()
}

type accessTokenEntry struct {
	accessToken string
	expiresAt   time.Time
}

var (
	accessTokenCacheMu sync.Mutex
	accessTokenCache   = map[string]accessTokenEntry{}
)

// OIDCTokenSource obtains tokens from the hosted platform's discovered OIDC
// token endpoint and stores only refresh tokens through the supplied store.
type OIDCTokenSource struct {
	hostURL   string
	accountID string
	store     RefreshTokenStore
}

// NewOIDCTokenSource creates a token source for one hosted account.
func NewOIDCTokenSource(hostURL, accountID string, store RefreshTokenStore) *OIDCTokenSource {
	return &OIDCTokenSource{hostURL: hostURL, accountID: accountID, store: store}
}

// Token returns a cached access token or refreshes it through standard OIDC.
func (s *OIDCTokenSource) Token(ctx context.Context) (string, error) {
	key := s.cacheKey()
	accessTokenCacheMu.Lock()
	defer accessTokenCacheMu.Unlock()

	if entry, ok := accessTokenCache[key]; ok && time.Until(entry.expiresAt) > accessTokenSkew {
		return entry.accessToken, nil
	}
	return s.refreshLocked(ctx, key)
}

// Invalidate discards the cached access token for this hosted account.
func (s *OIDCTokenSource) Invalidate() {
	accessTokenCacheMu.Lock()
	defer accessTokenCacheMu.Unlock()
	delete(accessTokenCache, s.cacheKey())
}

func (s *OIDCTokenSource) cacheKey() string {
	return s.hostURL + "\x00" + s.accountID
}

func (s *OIDCTokenSource) refreshLocked(ctx context.Context, key string) (string, error) {
	refreshToken, err := s.store.GetPlatformRefreshToken(s.hostURL, s.accountID)
	if err != nil {
		return "", fmt.Errorf("cannot load hosted WhoDB refresh token. Run: whodb login --host %s", s.hostURL)
	}

	tokens, err := RefreshToken(ctx, s.hostURL, refreshToken)
	if err != nil && IsInvalidGrant(err) {
		// Another CLI process may have rotated the keyring token between the
		// first read and this refresh attempt. Retry only when it changed.
		latest, latestErr := s.store.GetPlatformRefreshToken(s.hostURL, s.accountID)
		if latestErr == nil && latest != refreshToken {
			refreshToken = latest
			tokens, err = RefreshToken(ctx, s.hostURL, refreshToken)
		}
	}
	if err != nil {
		delete(accessTokenCache, key)
		return "", fmt.Errorf("cannot refresh hosted WhoDB login. Run: whodb login --host %s", s.hostURL)
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
		if err := s.store.SavePlatformRefreshToken(s.hostURL, s.accountID, tokens.RefreshToken); err != nil {
			return "", fmt.Errorf("cannot update hosted WhoDB refresh token: %w", err)
		}
	}
	if tokens.ExpiresIn <= 0 {
		delete(accessTokenCache, key)
		return tokens.AccessToken, nil
	}
	accessTokenCache[key] = accessTokenEntry{
		accessToken: tokens.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
	}
	return tokens.AccessToken, nil
}
