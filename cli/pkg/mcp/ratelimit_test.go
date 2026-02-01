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
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_QPSLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     3,
		Daily:   0, // Unlimited daily
	})

	ip := "192.168.1.1"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		result := rl.Allow(ip)
		if !result.Allowed {
			t.Errorf("Request %d should be allowed, got blocked", i+1)
		}
	}

	// 4th request should be blocked
	result := rl.Allow(ip)
	if result.Allowed {
		t.Error("4th request should be blocked due to QPS limit")
	}
	if result.LimitType != "qps" {
		t.Errorf("Expected limit type 'qps', got %q", result.LimitType)
	}
	if result.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
}

func TestRateLimiter_QPSSlidingWindow(t *testing.T) {
	now := time.Now()
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     2,
		Daily:   0,
	})
	rl.nowFunc = func() time.Time { return now }

	ip := "192.168.1.2"

	// Use 2 requests
	rl.Allow(ip)
	rl.Allow(ip)

	// Should be blocked
	result := rl.Allow(ip)
	if result.Allowed {
		t.Error("Should be blocked after 2 requests")
	}

	// Move time forward 1.1 seconds (outside sliding window)
	now = now.Add(1100 * time.Millisecond)

	// Should be allowed again
	result = rl.Allow(ip)
	if !result.Allowed {
		t.Error("Should be allowed after sliding window expires")
	}
}

func TestRateLimiter_DailyLimit(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     100, // High QPS to not interfere
		Daily:   5,
	})

	ip := "192.168.1.3"

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		result := rl.Allow(ip)
		if !result.Allowed {
			t.Errorf("Request %d should be allowed, got blocked", i+1)
		}
	}

	// 6th request should be blocked
	result := rl.Allow(ip)
	if result.Allowed {
		t.Error("6th request should be blocked due to daily limit")
	}
	if result.LimitType != "daily" {
		t.Errorf("Expected limit type 'daily', got %q", result.LimitType)
	}
}

func TestRateLimiter_DailyReset(t *testing.T) {
	now := time.Date(2024, 1, 15, 23, 59, 59, 0, time.UTC)
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     100,
		Daily:   2,
	})
	rl.nowFunc = func() time.Time { return now }

	ip := "192.168.1.4"

	// Use up daily limit
	rl.Allow(ip)
	rl.Allow(ip)

	result := rl.Allow(ip)
	if result.Allowed {
		t.Error("Should be blocked after daily limit")
	}

	// Move to next day (midnight UTC)
	now = time.Date(2024, 1, 16, 0, 0, 1, 0, time.UTC)

	// Should be allowed after daily reset
	result = rl.Allow(ip)
	if !result.Allowed {
		t.Error("Should be allowed after daily reset at midnight UTC")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     2,
		Daily:   0,
	})

	ip1 := "192.168.1.5"
	ip2 := "192.168.1.6"

	// Use up IP1's limit
	rl.Allow(ip1)
	rl.Allow(ip1)
	result := rl.Allow(ip1)
	if result.Allowed {
		t.Error("IP1 should be blocked")
	}

	// IP2 should still be allowed
	result = rl.Allow(ip2)
	if !result.Allowed {
		t.Error("IP2 should still be allowed")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: false,
		QPS:     1,
		Daily:   1,
	})

	ip := "192.168.1.7"

	// Should always be allowed when disabled
	for i := 0; i < 100; i++ {
		result := rl.Allow(ip)
		if !result.Allowed {
			t.Error("All requests should be allowed when rate limiting is disabled")
		}
	}
}

func TestRateLimiter_BypassToken(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled:     true,
		QPS:         1,
		Daily:       1,
		BypassToken: "secret-token",
	})

	// Request without bypass header
	req1 := httptest.NewRequest("POST", "/mcp", nil)
	if rl.IsBypassed(req1) {
		t.Error("Request without bypass header should not be bypassed")
	}

	// Request with wrong bypass header
	req2 := httptest.NewRequest("POST", "/mcp", nil)
	req2.Header.Set("X-RateLimit-Bypass", "wrong-token")
	if rl.IsBypassed(req2) {
		t.Error("Request with wrong token should not be bypassed")
	}

	// Request with correct bypass header
	req3 := httptest.NewRequest("POST", "/mcp", nil)
	req3.Header.Set("X-RateLimit-Bypass", "secret-token")
	if !rl.IsBypassed(req3) {
		t.Error("Request with correct token should be bypassed")
	}
}

func TestRateLimiter_BypassTokenEmpty(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled:     true,
		QPS:         1,
		BypassToken: "", // No bypass token configured
	})

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.Header.Set("X-RateLimit-Bypass", "any-token")
	if rl.IsBypassed(req) {
		t.Error("Should not be bypassed when no bypass token is configured")
	}
}

func TestRateLimiter_Middleware_AllowedRequest(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     10,
		Daily:   0,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = "192.168.1.8:12345"
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "success" {
		t.Errorf("Expected 'success', got %q", rr.Body.String())
	}

	// Check rate limit headers
	if rr.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header")
	}
	if rr.Header().Get("X-RateLimit-Remaining") == "" {
		t.Error("Expected X-RateLimit-Remaining header")
	}
}

func TestRateLimiter_Middleware_BlockedRequest(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     1,
		Daily:   0,
	})

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.9:12345"

	// First request allowed
	req1 := httptest.NewRequest("POST", "/mcp", nil)
	req1.RemoteAddr = ip
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Errorf("First request should be allowed, got %d", rr1.Code)
	}

	// Second request blocked
	req2 := httptest.NewRequest("POST", "/mcp", nil)
	req2.RemoteAddr = ip
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", rr2.Code)
	}

	// Check response body
	var errResp RateLimitError
	if err := json.Unmarshal(rr2.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}
	if errResp.RetryAfter <= 0 {
		t.Error("RetryAfter should be positive")
	}
	if errResp.LimitType != "qps" {
		t.Errorf("Expected limit type 'qps', got %q", errResp.LimitType)
	}

	// Check headers
	if rr2.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header")
	}
	if rr2.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", rr2.Header().Get("Content-Type"))
	}
}

func TestRateLimiter_Middleware_BypassedRequest(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled:     true,
		QPS:         1,
		Daily:       1,
		BypassToken: "bypass-me",
	})

	callCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.10:12345"

	// Make many requests with bypass token - all should succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		req.RemoteAddr = ip
		req.Header.Set("X-RateLimit-Bypass", "bypass-me")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should be allowed with bypass, got %d", i+1, rr.Code)
		}
	}

	if callCount != 10 {
		t.Errorf("Expected 10 handler calls, got %d", callCount)
	}
}

func TestRateLimiter_Middleware_Disabled(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: false,
		QPS:     1,
	})

	callCount := 0
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	ip := "192.168.1.11:12345"

	// Make many requests - all should succeed when disabled
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/mcp", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("Request %d should be allowed when disabled, got %d", i+1, rr.Code)
		}
	}

	if callCount != 10 {
		t.Errorf("Expected 10 handler calls, got %d", callCount)
	}
}

func TestRateLimiter_ExtractIP(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{Enabled: true, QPS: 10})

	tests := []struct {
		name     string
		setup    func(*http.Request)
		expected string
	}{
		{
			name:     "RemoteAddr only",
			setup:    func(r *http.Request) { r.RemoteAddr = "10.0.0.1:12345" },
			expected: "10.0.0.1",
		},
		{
			name: "X-Forwarded-For single IP",
			setup: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.2:12345"
				r.Header.Set("X-Forwarded-For", "203.0.113.50")
			},
			expected: "203.0.113.50",
		},
		{
			name: "X-Forwarded-For multiple IPs (use first)",
			setup: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.3:12345"
				r.Header.Set("X-Forwarded-For", "203.0.113.50, 70.41.3.18, 150.172.238.178")
			},
			expected: "203.0.113.50",
		},
		{
			name: "X-Real-IP takes precedence over X-Forwarded-For",
			setup: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.4:12345"
				r.Header.Set("X-Forwarded-For", "203.0.113.50")
				r.Header.Set("X-Real-IP", "198.51.100.25")
			},
			expected: "198.51.100.25",
		},
		{
			name: "CF-Connecting-IP takes highest precedence",
			setup: func(r *http.Request) {
				r.RemoteAddr = "10.0.0.5:12345"
				r.Header.Set("X-Forwarded-For", "203.0.113.50")
				r.Header.Set("X-Real-IP", "198.51.100.25")
				r.Header.Set("CF-Connecting-IP", "192.0.2.100")
			},
			expected: "192.0.2.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", nil)
			tt.setup(req)
			got := rl.extractIP(req)
			if got != tt.expected {
				t.Errorf("extractIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	now := time.Now()
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     10,
		Daily:   0,
	})
	rl.nowFunc = func() time.Time { return now }

	// Create some entries
	rl.Allow("192.168.1.20")
	rl.Allow("192.168.1.21")
	rl.Allow("192.168.1.22")

	stats := rl.Stats()
	if stats["tracked_ips"].(int) != 3 {
		t.Errorf("Expected 3 tracked IPs, got %d", stats["tracked_ips"].(int))
	}

	// Move time forward past cleanup threshold
	now = now.Add(15 * time.Minute)

	// Run cleanup with 10 minute max age
	rl.Cleanup(10 * time.Minute)

	stats = rl.Stats()
	if stats["tracked_ips"].(int) != 0 {
		t.Errorf("Expected 0 tracked IPs after cleanup, got %d", stats["tracked_ips"].(int))
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     100,
		Daily:   1000,
	})

	var wg sync.WaitGroup
	const numGoroutines = 50
	const requestsPerGoroutine = 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1.100" // Same IP to test concurrent access
			for j := 0; j < requestsPerGoroutine; j++ {
				rl.Allow(ip)
			}
		}(i)
	}

	wg.Wait()

	// Just verify no panic/deadlock occurred
	stats := rl.Stats()
	if stats["tracked_ips"].(int) != 1 {
		t.Errorf("Expected 1 tracked IP, got %d", stats["tracked_ips"].(int))
	}
}

func TestRateLimiter_Stats(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled:     true,
		QPS:         15,
		Daily:       500,
		BypassToken: "secret",
	})

	rl.Allow("192.168.1.30")
	rl.Allow("192.168.1.31")

	stats := rl.Stats()

	if stats["enabled"] != true {
		t.Error("Expected enabled=true")
	}
	if stats["qps_limit"] != 15 {
		t.Errorf("Expected qps_limit=15, got %v", stats["qps_limit"])
	}
	if stats["daily_limit"] != 500 {
		t.Errorf("Expected daily_limit=500, got %v", stats["daily_limit"])
	}
	if stats["tracked_ips"] != 2 {
		t.Errorf("Expected tracked_ips=2, got %v", stats["tracked_ips"])
	}
	if stats["bypass_active"] != true {
		t.Error("Expected bypass_active=true")
	}
}

func TestRateLimiter_DefaultValues(t *testing.T) {
	// Test defaults when enabled with zero values
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     0,     // Should default to 10
		Daily:   0,     // Should default to 1000 when enabled
	})

	if rl.opts.QPS != 10 {
		t.Errorf("Expected default QPS=10, got %d", rl.opts.QPS)
	}
	if rl.opts.Daily != 1000 {
		t.Errorf("Expected default Daily=1000, got %d", rl.opts.Daily)
	}
}

func TestRateLimiter_Remaining(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     5,
		Daily:   0,
	})

	ip := "192.168.1.40"

	result := rl.Allow(ip)
	if result.Remaining != 4 {
		t.Errorf("Expected remaining=4, got %d", result.Remaining)
	}

	result = rl.Allow(ip)
	if result.Remaining != 3 {
		t.Errorf("Expected remaining=3, got %d", result.Remaining)
	}

	result = rl.Allow(ip)
	if result.Remaining != 2 {
		t.Errorf("Expected remaining=2, got %d", result.Remaining)
	}
}

func TestRateLimiter_DailyErrorMessage(t *testing.T) {
	rl := NewRateLimiter(RateLimitOptions{
		Enabled: true,
		QPS:     100,
		Daily:   1,
	})

	ip := "192.168.1.50"

	// Use up daily limit
	rl.Allow(ip)

	// Next request blocked
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = ip + ":12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", rr.Code)
	}

	var errResp RateLimitError
	json.Unmarshal(rr.Body.Bytes(), &errResp)

	if errResp.LimitType != "daily" {
		t.Errorf("Expected limit type 'daily', got %q", errResp.LimitType)
	}
	if errResp.Message == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestIntToString(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{123, "123"},
		{1000, "1000"},
		{-1, "-1"},
		{-123, "-123"},
	}

	for _, tt := range tests {
		got := intToString(tt.input)
		if got != tt.expected {
			t.Errorf("intToString(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
