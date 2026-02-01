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

package mcp

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RateLimitOptions configures the rate limiter.
type RateLimitOptions struct {
	// Enabled turns rate limiting on/off.
	Enabled bool

	// QPS is the maximum queries per second per IP (default: 10).
	QPS int

	// Daily is the maximum requests per day per IP (default: 1000, 0 = unlimited).
	Daily int

	// BypassToken allows trusted clients to bypass rate limiting.
	// Clients include this in the X-RateLimit-Bypass header.
	BypassToken string
}

// rateLimitEntry tracks rate limit state for a single IP.
type rateLimitEntry struct {
	// Sliding window for QPS tracking
	timestamps []time.Time
	tsMu       sync.Mutex

	// Daily counter with reset time
	dailyCount int64
	dailyReset time.Time
	dailyMu    sync.Mutex
}

// RateLimiter implements IP-based rate limiting.
type RateLimiter struct {
	opts    RateLimitOptions
	entries sync.Map // map[string]*rateLimitEntry

	// For testing: allows overriding time.Now()
	nowFunc func() time.Time
}

// RateLimitResult contains rate limit check results.
type RateLimitResult struct {
	Allowed    bool
	RetryAfter time.Duration
	Remaining  int
	Limit      int
	LimitType  string // "qps" or "daily"
}

// RateLimitError is returned when rate limit is exceeded.
type RateLimitError struct {
	Message    string `json:"error"`
	RetryAfter int    `json:"retry_after_seconds"`
	LimitType  string `json:"limit_type"`
}

// NewRateLimiter creates a new rate limiter with the given options.
func NewRateLimiter(opts RateLimitOptions) *RateLimiter {
	// Apply defaults
	if opts.QPS <= 0 {
		opts.QPS = 10
	}
	if opts.Daily < 0 {
		opts.Daily = 0 // 0 means unlimited
	}
	if opts.Daily == 0 && opts.Enabled {
		opts.Daily = 1000 // Default daily limit when enabled
	}

	return &RateLimiter{
		opts:    opts,
		nowFunc: time.Now,
	}
}

// Allow checks if a request from the given IP is allowed.
func (rl *RateLimiter) Allow(ip string) RateLimitResult {
	if !rl.opts.Enabled {
		return RateLimitResult{Allowed: true}
	}

	now := rl.now()
	entry := rl.getOrCreateEntry(ip)

	// Check QPS limit (sliding window)
	qpsResult := rl.checkQPS(entry, now)
	if !qpsResult.Allowed {
		return qpsResult
	}

	// Check daily limit (fixed window, resets at midnight UTC)
	if rl.opts.Daily > 0 {
		dailyResult := rl.checkDaily(entry, now)
		if !dailyResult.Allowed {
			return dailyResult
		}
	}

	return RateLimitResult{
		Allowed:   true,
		Remaining: qpsResult.Remaining,
		Limit:     rl.opts.QPS,
		LimitType: "qps",
	}
}

// IsBypassed checks if the request has a valid bypass token.
func (rl *RateLimiter) IsBypassed(r *http.Request) bool {
	if rl.opts.BypassToken == "" {
		return false
	}
	token := r.Header.Get("X-RateLimit-Bypass")
	return token != "" && token == rl.opts.BypassToken
}

// Middleware returns an HTTP middleware that applies rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip if not enabled
		if !rl.opts.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Check bypass token
		if rl.IsBypassed(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract client IP
		ip := rl.extractIP(r)

		// Check rate limit
		result := rl.Allow(ip)
		if !result.Allowed {
			rl.writeRateLimitResponse(w, result)
			return
		}

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", intToString(result.Limit))
		w.Header().Set("X-RateLimit-Remaining", intToString(result.Remaining))

		next.ServeHTTP(w, r)
	})
}

// Cleanup removes stale entries older than the given duration.
// Call this periodically to prevent memory growth.
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	now := rl.now()
	cutoff := now.Add(-maxAge)

	rl.entries.Range(func(key, value any) bool {
		entry := value.(*rateLimitEntry)

		// Check if entry is stale (no recent activity)
		entry.tsMu.Lock()
		lastActive := time.Time{}
		if len(entry.timestamps) > 0 {
			lastActive = entry.timestamps[len(entry.timestamps)-1]
		}
		entry.tsMu.Unlock()

		if lastActive.Before(cutoff) {
			rl.entries.Delete(key)
		}
		return true
	})
}

// Stats returns current rate limiter statistics.
func (rl *RateLimiter) Stats() map[string]any {
	count := 0
	rl.entries.Range(func(_, _ any) bool {
		count++
		return true
	})
	return map[string]any{
		"enabled":       rl.opts.Enabled,
		"qps_limit":     rl.opts.QPS,
		"daily_limit":   rl.opts.Daily,
		"tracked_ips":   count,
		"bypass_active": rl.opts.BypassToken != "",
	}
}

// Internal methods

func (rl *RateLimiter) now() time.Time {
	if rl.nowFunc != nil {
		return rl.nowFunc()
	}
	return time.Now()
}

func (rl *RateLimiter) getOrCreateEntry(ip string) *rateLimitEntry {
	entry, ok := rl.entries.Load(ip)
	if ok {
		return entry.(*rateLimitEntry)
	}

	newEntry := &rateLimitEntry{
		timestamps: make([]time.Time, 0, rl.opts.QPS),
		dailyReset: rl.nextMidnightUTC(),
	}

	actual, loaded := rl.entries.LoadOrStore(ip, newEntry)
	if loaded {
		return actual.(*rateLimitEntry)
	}
	return newEntry
}

func (rl *RateLimiter) checkQPS(entry *rateLimitEntry, now time.Time) RateLimitResult {
	entry.tsMu.Lock()
	defer entry.tsMu.Unlock()

	windowStart := now.Add(-time.Second)

	// Remove timestamps outside the sliding window
	validIdx := 0
	for i, ts := range entry.timestamps {
		if ts.After(windowStart) {
			validIdx = i
			break
		}
		if i == len(entry.timestamps)-1 {
			validIdx = len(entry.timestamps) // All expired
		}
	}
	if validIdx > 0 && validIdx <= len(entry.timestamps) {
		entry.timestamps = entry.timestamps[validIdx:]
	}

	// Check if under limit
	if len(entry.timestamps) >= rl.opts.QPS {
		// Calculate retry after based on oldest timestamp in window
		oldestInWindow := entry.timestamps[0]
		retryAfter := oldestInWindow.Add(time.Second).Sub(now)
		if retryAfter < 0 {
			retryAfter = 100 * time.Millisecond // Minimum retry
		}

		return RateLimitResult{
			Allowed:    false,
			RetryAfter: retryAfter,
			Remaining:  0,
			Limit:      rl.opts.QPS,
			LimitType:  "qps",
		}
	}

	// Add current timestamp
	entry.timestamps = append(entry.timestamps, now)

	return RateLimitResult{
		Allowed:   true,
		Remaining: rl.opts.QPS - len(entry.timestamps),
		Limit:     rl.opts.QPS,
		LimitType: "qps",
	}
}

func (rl *RateLimiter) checkDaily(entry *rateLimitEntry, now time.Time) RateLimitResult {
	entry.dailyMu.Lock()
	defer entry.dailyMu.Unlock()

	// Reset counter if past midnight
	if now.After(entry.dailyReset) {
		entry.dailyCount = 0
		entry.dailyReset = rl.nextMidnightUTC()
	}

	// Check if under limit
	if int(atomic.LoadInt64(&entry.dailyCount)) >= rl.opts.Daily {
		retryAfter := entry.dailyReset.Sub(now)
		if retryAfter < 0 {
			retryAfter = time.Hour // Fallback
		}

		return RateLimitResult{
			Allowed:    false,
			RetryAfter: retryAfter,
			Remaining:  0,
			Limit:      rl.opts.Daily,
			LimitType:  "daily",
		}
	}

	// Increment counter
	atomic.AddInt64(&entry.dailyCount, 1)

	return RateLimitResult{
		Allowed:   true,
		Remaining: rl.opts.Daily - int(atomic.LoadInt64(&entry.dailyCount)),
		Limit:     rl.opts.Daily,
		LimitType: "daily",
	}
}

func (rl *RateLimiter) nextMidnightUTC() time.Time {
	now := rl.now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

func (rl *RateLimiter) extractIP(r *http.Request) string {
	// Check common proxy headers (in order of preference)
	if ip := r.Header.Get("CF-Connecting-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain (original client)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Fall back to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (rl *RateLimiter) writeRateLimitResponse(w http.ResponseWriter, result RateLimitResult) {
	retryAfterSecs := int(result.RetryAfter.Seconds())
	if retryAfterSecs < 1 {
		retryAfterSecs = 1
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", intToString(retryAfterSecs))
	w.Header().Set("X-RateLimit-Limit", intToString(result.Limit))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.WriteHeader(http.StatusTooManyRequests)

	errResp := RateLimitError{
		Message:    "Rate limit exceeded",
		RetryAfter: retryAfterSecs,
		LimitType:  result.LimitType,
	}

	if result.LimitType == "daily" {
		errResp.Message = "Daily rate limit exceeded. Limit resets at midnight UTC."
	} else {
		errResp.Message = "Too many requests. Please slow down."
	}

	_ = json.NewEncoder(w).Encode(errResp)
}

// intToString converts int to string without importing strconv.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
