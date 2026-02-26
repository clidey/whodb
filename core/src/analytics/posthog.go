/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/posthog/posthog-go"

	"github.com/clidey/whodb/core/src/log"
)

type contextKey string

const (
	contextKeyMetadata contextKey = "analytics.metadata"

	libraryName = "whodb-backend"
	anonymousID = "anonymous"
	headerKeyID = "X-Whodb-Analytics-Id"
)

// Config contains runtime configuration for the PostHog client.
type Config struct {
	APIKey      string
	Host        string
	Environment string
	AppVersion  string
}

// Metadata stores request-scoped attributes that are useful for analytics enrichment.
type Metadata struct {
	DistinctID string
	Domain     string
	Path       string
	Method     string
	UserAgent  string
	Referer    string
	RequestID  string
}

var (
	clientMu sync.RWMutex
	client   posthog.Client
	cfg      Config

	initOnce   sync.Once
	enabled    atomic.Bool
	fallbackID atomic.Value
)

// Initialize prepares the global PostHog client. It is safe to invoke multiple times.
func Initialize(config Config) error {
	var initErr error
	initOnce.Do(func() {
		cfg = config
		fallbackID.Store(hashIdentifier(fmt.Sprintf("fallback:%d", time.Now().UnixNano())))
		enabled.Store(false)

		c, err := posthog.NewWithConfig(config.APIKey, posthog.Config{
			Endpoint: config.Host,
		})
		if err != nil {
			initErr = err
			//log.WithError(err).Error("Analytics: failed to initialize PostHog client")
			return
		}
		storeClient(c)
		//log.Info("Analytics: PostHog backend client ready")
	})

	if initErr != nil {
		enabled.Store(false)
	}

	return initErr
}

// Shutdown flushes events and disposes the PostHog client.
func Shutdown() {
	old := storeClient(nil)
	if old == nil {
		return
	}

	if err := old.Close(); err != nil {
		log.WithError(err).Warn("Analytics: unable to flush PostHog queue during shutdown")
	}

	enabled.Store(false)
}

// SetEnabled toggles whether analytics events should be emitted.
func SetEnabled(state bool) {
	if state && !Configured() {
		enabled.Store(false)
		return
	}
	enabled.Store(state)
}

// Enabled reports whether analytics are currently active.
func Enabled() bool {
	return enabled.Load() && Configured()
}

// Configured reports whether a PostHog client has been configured successfully.
func Configured() bool {
	return loadClient() != nil
}

// WithMetadata stores metadata inside the supplied context.
func WithMetadata(ctx context.Context, metadata Metadata) context.Context {
	return context.WithValue(ctx, contextKeyMetadata, metadata)
}

// MetadataFromContext extracts analytics metadata from the context.
func MetadataFromContext(ctx context.Context) Metadata {
	value := ctx.Value(contextKeyMetadata)
	if value == nil {
		return Metadata{}
	}

	metadata, ok := value.(Metadata)
	if !ok {
		return Metadata{}
	}
	return metadata
}

// Capture emits a generic analytics event using the distinct id derived from the context.
func Capture(ctx context.Context, event string, properties map[string]any) {
	CaptureWithDistinctID(ctx, distinctIDFromContext(ctx), event, properties)
}

// CaptureWithDistinctID emits an analytics event using an explicit distinct id.
func CaptureWithDistinctID(ctx context.Context, distinctID string, event string, properties map[string]any) {
	distinctID = strings.TrimSpace(distinctID)
	if !Enabled() || distinctID == "" {
		return
	}

	message := posthog.Capture{
		Event:      event,
		DistinctId: distinctID,
		Timestamp:  time.Now().UTC(),
		Properties: buildProperties(ctx, properties),
	}

	enqueue(message)
}

// CaptureError emits a structured error event.
func CaptureError(ctx context.Context, operation string, err error, properties map[string]any) {
	if err == nil {
		return
	}

	props := make(map[string]any, len(properties)+2)
	for k, v := range properties {
		props[k] = v
	}
	props["operation"] = operation
	props["error_message"] = err.Error()

	Capture(ctx, "$exception", props)
}

// IdentifyWithDistinctID associates user traits with a specific distinct id.
func IdentifyWithDistinctID(ctx context.Context, distinctID string, traits map[string]any) {
	distinctID = strings.TrimSpace(distinctID)
	if !Enabled() || distinctID == "" {
		return
	}

	message := posthog.Identify{
		DistinctId: distinctID,
		Timestamp:  time.Now().UTC(),
		Properties: buildIdentifyTraits(ctx, traits),
	}

	enqueue(message)
}

// TrackMutation is a helper for recording GraphQL mutations executed by the user.
func TrackMutation(ctx context.Context, name string, props map[string]any) {
	Capture(ctx, fmt.Sprintf("graphql.mutation.%s", name), props)
}

// HashIdentifier hashes a potentially sensitive identifier before emission.
func HashIdentifier(value string) string {
	return hashIdentifier(value)
}

// BuildMetadata constructs request metadata from an HTTP request.
func BuildMetadata(r *http.Request) Metadata {
	domain := deriveDomain(r)

	metadata := Metadata{
		DistinctID: strings.TrimSpace(r.Header.Get(headerKeyID)),
		Domain:     domain,
		Path:       "",
		Method:     r.Method,
		UserAgent:  r.UserAgent(),
		Referer:    r.Referer(),
		RequestID:  strings.TrimSpace(r.Header.Get("X-Request-Id")),
	}

	if r.URL != nil {
		metadata.Path = r.URL.Path
	}

	return metadata
}

func deriveDomain(r *http.Request) string {
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if parsed, err := url.Parse(origin); err == nil && parsed.Hostname() != "" {
			return strings.ToLower(parsed.Hostname())
		}
	}

	host := strings.TrimSpace(r.Host)
	if host == "" && r.URL != nil {
		host = strings.TrimSpace(r.URL.Host)
	}
	if host == "" {
		return ""
	}

	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(parsedHost)
	}

	return strings.ToLower(host)
}

func buildProperties(ctx context.Context, properties map[string]any) posthog.Properties {
	props := posthog.NewProperties()
	for key, value := range properties {
		props.Set(key, value)
	}

	metadata := MetadataFromContext(ctx)

	if metadata.Domain != "" {
		if _, exists := props["domain"]; !exists {
			props.Set("domain", metadata.Domain)
		}
	}
	if metadata.Path != "" {
		if _, exists := props["path"]; !exists {
			props.Set("path", metadata.Path)
		}
	}
	if metadata.Method != "" {
		if _, exists := props["method"]; !exists {
			props.Set("method", metadata.Method)
		}
	}
	if metadata.UserAgent != "" {
		if _, exists := props["user_agent"]; !exists {
			props.Set("user_agent", metadata.UserAgent)
		}
	}
	if metadata.Referer != "" {
		if _, exists := props["referer"]; !exists {
			props.Set("referer", metadata.Referer)
		}
	}
	if metadata.RequestID != "" {
		if _, exists := props["request_id"]; !exists {
			props.Set("request_id", metadata.RequestID)
		}
	}

	if cfg.Environment != "" {
		if _, exists := props["environment"]; !exists {
			props.Set("environment", cfg.Environment)
		}
	}
	if cfg.AppVersion != "" {
		if _, exists := props["app_version"]; !exists {
			props.Set("app_version", cfg.AppVersion)
		}
	}

	props.Set("$lib", libraryName)
	if cfg.AppVersion != "" {
		props.Set("$lib_version", cfg.AppVersion)
	}

	return props
}

func buildIdentifyTraits(ctx context.Context, traits map[string]any) posthog.Properties {
	properties := posthog.NewProperties()
	for key, value := range traits {
		properties.Set(key, value)
	}

	metadata := MetadataFromContext(ctx)
	if metadata.Domain != "" {
		if _, exists := properties["last_seen_domain"]; !exists {
			properties.Set("last_seen_domain", metadata.Domain)
		}
	}
	if cfg.Environment != "" {
		if _, exists := properties["environment"]; !exists {
			properties.Set("environment", cfg.Environment)
		}
	}
	if cfg.AppVersion != "" {
		if _, exists := properties["app_version"]; !exists {
			properties.Set("app_version", cfg.AppVersion)
		}
	}

	return properties
}

func enqueue(message posthog.Message) {
	if !Enabled() {
		return
	}

	c := loadClient()
	if c == nil {
		return
	}

	if err := c.Enqueue(message); err != nil && !errors.Is(err, posthog.ErrTooManyRequests) {
		log.WithError(err).Warn("Analytics: failed to enqueue message")
	}
}

func loadClient() posthog.Client {
	clientMu.RLock()
	defer clientMu.RUnlock()
	return client
}

func storeClient(c posthog.Client) posthog.Client {
	clientMu.Lock()
	old := client
	client = c
	clientMu.Unlock()
	return old
}

func distinctIDFromContext(ctx context.Context) string {
	metadata := MetadataFromContext(ctx)
	if metadata.DistinctID != "" {
		return metadata.DistinctID
	}

	if value, ok := fallbackID.Load().(string); ok && value != "" {
		return value
	}

	return anonymousID
}

func hashIdentifier(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	hasher := sha256.New()
	if _, err := hasher.Write([]byte(value)); err != nil {
		return ""
	}
	return hex.EncodeToString(hasher.Sum(nil))
}
