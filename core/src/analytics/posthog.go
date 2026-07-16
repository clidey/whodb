/*
 * Copyright 2026 Clidey, Inc.
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
	"maps"
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
	Deployment  string
	// Edition is the build edition ("ce" or "ee"). Callers must set it so
	// every event carries a non-null build_edition property.
	Edition string
	// Source identifies the emitting process ("backend", "cli", or "desktop")
	// and is stamped on every event as the source property.
	Source string
	// SuppressClientLogs disables PostHog background logger output. Useful for
	// short-lived CLIs where network failures should stay silent.
	SuppressClientLogs bool
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

// GroupIdentity describes a PostHog group and its safe group properties.
type GroupIdentity struct {
	Type       string
	Key        string
	Properties map[string]any
}

var (
	clientMu sync.RWMutex
	client   posthog.Client
	cfg      Config

	initOnce sync.Once
	enabled  atomic.Bool

	// distinctIDResolver, when set, is consulted first to derive the analytics
	// distinct id from a request context. Set once during process startup,
	// before the server begins handling requests.
	distinctIDResolver func(ctx context.Context) string
)

// SetDistinctIDResolver registers a resolver consulted before request metadata
// when deriving the analytics distinct id. Call during startup, before the
// server begins handling requests.
func SetDistinctIDResolver(resolver func(ctx context.Context) string) {
	distinctIDResolver = resolver
}

// Initialize prepares the global PostHog client. It is safe to invoke multiple times.
func Initialize(config Config) error {
	var initErr error
	initOnce.Do(func() {
		cfg = config
		enabled.Store(false)

		posthogConfig := posthog.Config{
			Endpoint: config.Host,
		}
		if config.SuppressClientLogs {
			posthogConfig.Logger = silentPosthogLogger{}
		}

		c, err := posthog.NewWithConfig(config.APIKey, posthogConfig)
		if err != nil {
			initErr = err
			return
		}
		storeClient(c)
	})

	if initErr != nil {
		enabled.Store(false)
	}

	return initErr
}

type silentPosthogLogger struct{}

func (silentPosthogLogger) Debugf(string, ...any) {}

func (silentPosthogLogger) Logf(string, ...any) {}

func (silentPosthogLogger) Warnf(string, ...any) {}

func (silentPosthogLogger) Errorf(string, ...any) {}

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
		Groups:     buildGroups(properties),
	}

	enqueue(message)
}

// CaptureError emits a structured error event. The raw error message is never
// sent; it is mapped to a low-cardinality code via ErrorCode to avoid leaking
// hostnames, table names, or SQL fragments.
func CaptureError(ctx context.Context, operation string, err error, properties map[string]any) {
	if err == nil {
		return
	}

	props := make(map[string]any, len(properties)+2)
	maps.Copy(props, properties)
	props["operation"] = operation
	props["error_code"] = ErrorCode(err)

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

// IdentifyGroup creates or updates a PostHog group with safe group properties.
func IdentifyGroup(ctx context.Context, group GroupIdentity) {
	group.Type = strings.TrimSpace(group.Type)
	group.Key = strings.TrimSpace(group.Key)
	if !Enabled() || group.Type == "" || group.Key == "" {
		return
	}

	message := posthog.GroupIdentify{
		Type:       group.Type,
		Key:        group.Key,
		Timestamp:  time.Now().UTC(),
		Properties: buildIdentifyTraits(ctx, group.Properties),
	}

	enqueue(message)
}

// TrackMutation is a helper for recording GraphQL mutations executed by the user.
func TrackMutation(ctx context.Context, name string, props map[string]any) {
	Capture(ctx, "graphql.mutation."+name, props)
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
		// Mirror posthog-js, which auto-captures $host from window.location:
		// stamping it here keeps server-side events filterable by host too.
		if _, exists := props["$host"]; !exists {
			props.Set("$host", metadata.Domain)
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
		if _, exists := props["build_environment"]; !exists {
			props.Set("build_environment", cfg.Environment)
		}
	}
	if cfg.AppVersion != "" {
		if _, exists := props["app_version"]; !exists {
			props.Set("app_version", cfg.AppVersion)
		}
	}
	if cfg.Deployment != "" {
		if _, exists := props["deployment"]; !exists {
			props.Set("deployment", cfg.Deployment)
		}
	}
	if cfg.Edition != "" {
		if _, exists := props["build_edition"]; !exists {
			props.Set("build_edition", cfg.Edition)
		}
	}
	if cfg.Source != "" {
		if _, exists := props["source"]; !exists {
			props.Set("source", cfg.Source)
		}
	}

	props.Set("$lib", libraryName)
	if cfg.AppVersion != "" {
		props.Set("$lib_version", cfg.AppVersion)
	}

	return props
}

func buildGroups(properties map[string]any) posthog.Groups {
	raw, ok := properties["$groups"]
	if !ok {
		return nil
	}

	groups := posthog.NewGroups()
	switch typed := raw.(type) {
	case map[string]any:
		for key, value := range typed {
			if groupType := strings.TrimSpace(key); groupType != "" {
				groups.Set(groupType, value)
			}
		}
	case map[string]string:
		for key, value := range typed {
			groupType := strings.TrimSpace(key)
			groupKey := strings.TrimSpace(value)
			if groupType != "" && groupKey != "" {
				groups.Set(groupType, groupKey)
			}
		}
	}
	if len(groups) == 0 {
		return nil
	}
	return groups
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
		if _, exists := properties["build_environment"]; !exists {
			properties.Set("build_environment", cfg.Environment)
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
	if distinctIDResolver != nil {
		if id := strings.TrimSpace(distinctIDResolver(ctx)); id != "" {
			return id
		}
	}

	metadata := MetadataFromContext(ctx)
	if metadata.DistinctID != "" {
		return metadata.DistinctID
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
