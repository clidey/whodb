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
	accessTokenLocks   sync.Map
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
	lock, _ := accessTokenLocks.LoadOrStore(key, make(chan struct{}, 1))
	gate := lock.(chan struct{})
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	defer func() { <-gate }()
	accessTokenCacheMu.Lock()
	entry, ok := accessTokenCache[key]
	accessTokenCacheMu.Unlock()
	if ok && time.Until(entry.expiresAt) > accessTokenSkew {
		return entry.accessToken, nil
	}
	if coordinator, ok := s.store.(interface {
		WithPlatformRefreshLock(context.Context, string, string, func() error) error
	}); ok {
		var token string
		err := coordinator.WithPlatformRefreshLock(ctx, s.hostURL, s.accountID, func() error {
			var err error
			token, err = s.refreshLocked(ctx, key)
			return err
		})
		return token, err
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
		s.Invalidate()
		return "", fmt.Errorf("cannot refresh hosted WhoDB login for %s: %w", s.hostURL, err)
	}
	if tokens.RefreshToken != "" && tokens.RefreshToken != refreshToken {
		if err := s.store.SavePlatformRefreshToken(s.hostURL, s.accountID, tokens.RefreshToken); err != nil {
			return "", fmt.Errorf("cannot update hosted WhoDB refresh token: %w", err)
		}
	}
	if tokens.ExpiresIn <= 0 {
		s.Invalidate()
		return tokens.AccessToken, nil
	}
	accessTokenCacheMu.Lock()
	accessTokenCache[key] = accessTokenEntry{
		accessToken: tokens.AccessToken,
		expiresAt:   time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second),
	}
	accessTokenCacheMu.Unlock()
	return tokens.AccessToken, nil
}
