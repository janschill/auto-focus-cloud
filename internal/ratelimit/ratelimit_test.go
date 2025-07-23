package ratelimit

import (
	"fmt"
	"testing"
	"time"
)

func TestFixedWindowLimiter_Allow_BasicFunctionality(t *testing.T) {
	limiter := New(3, time.Minute) // 3 requests per minute

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow("192.168.1.1") {
			t.Errorf("Request %d should be allowed, but was denied", i+1)
		}
	}

	// 4th request should be denied
	if limiter.Allow("192.168.1.1") {
		t.Error("4th request should be denied, but was allowed")
	}
}

func TestFixedWindowLimiter_Allow_DifferentIPs(t *testing.T) {
	limiter := New(2, time.Minute) // 2 requests per minute

	// Different IPs should have independent limits
	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Use up ip1's limit
	if !limiter.Allow(ip1) {
		t.Error("First request for ip1 should be allowed")
	}
	if !limiter.Allow(ip1) {
		t.Error("Second request for ip1 should be allowed")
	}
	if limiter.Allow(ip1) {
		t.Error("Third request for ip1 should be denied")
	}

	// ip2 should still have full limit available
	if !limiter.Allow(ip2) {
		t.Error("First request for ip2 should be allowed")
	}
	if !limiter.Allow(ip2) {
		t.Error("Second request for ip2 should be allowed")
	}
	if limiter.Allow(ip2) {
		t.Error("Third request for ip2 should be denied")
	}
}

func TestFixedWindowLimiter_Allow_WindowReset(t *testing.T) {
	limiter := New(2, 100*time.Millisecond) // 2 requests per 100ms

	ip := "192.168.1.1"

	// Use up the limit
	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("Second request should be allowed")
	}
	if limiter.Allow(ip) {
		t.Error("Third request should be denied")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be able to make requests again
	if !limiter.Allow(ip) {
		t.Error("First request after window reset should be allowed")
	}
	if !limiter.Allow(ip) {
		t.Error("Second request after window reset should be allowed")
	}
	if limiter.Allow(ip) {
		t.Error("Third request after window reset should be denied")
	}
}

func TestFixedWindowLimiter_Allow_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		maxRequests int
		window      time.Duration
		identifier  string
		requests    int
		expectPass  bool
	}{
		{
			name:        "zero limit should deny all",
			maxRequests: 0,
			window:      time.Minute,
			identifier:  "192.168.1.1",
			requests:    1,
			expectPass:  false,
		},
		{
			name:        "single request limit",
			maxRequests: 1,
			window:      time.Minute,
			identifier:  "192.168.1.1",
			requests:    1,
			expectPass:  true,
		},
		{
			name:        "empty identifier",
			maxRequests: 5,
			window:      time.Minute,
			identifier:  "",
			requests:    3,
			expectPass:  true,
		},
		{
			name:        "very long identifier",
			maxRequests: 5,
			window:      time.Minute,
			identifier:  "very.long.identifier.with.many.dots.and.characters.192.168.1.100",
			requests:    3,
			expectPass:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := New(tt.maxRequests, tt.window)

			var lastResult bool
			for i := 0; i < tt.requests; i++ {
				lastResult = limiter.Allow(tt.identifier)
			}

			if lastResult != tt.expectPass {
				t.Errorf("Expected %v, got %v for %d requests with limit %d",
					tt.expectPass, lastResult, tt.requests, tt.maxRequests)
			}
		})
	}
}

func TestFixedWindowLimiter_Allow_ConcurrentAccess(t *testing.T) {
	limiter := New(100, time.Minute) // High limit for concurrent test
	ip := "192.168.1.1"

	done := make(chan bool, 10)
	allowed := make(chan bool, 50)

	// Start 10 goroutines, each making 5 requests
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				allowed <- limiter.Allow(ip)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to finish
	for i := 0; i < 10; i++ {
		<-done
	}

	// Count allowed requests
	allowedCount := 0
	for i := 0; i < 50; i++ {
		if <-allowed {
			allowedCount++
		}
	}

	// All 50 requests should be allowed (under the limit of 100)
	if allowedCount != 50 {
		t.Errorf("Expected 50 allowed requests, got %d", allowedCount)
	}
}

func TestFixedWindowLimiter_Allow_TimeBoundaryConditions(t *testing.T) {
	limiter := New(1, 50*time.Millisecond) // 1 request per 50ms
	ip := "192.168.1.1"

	// First request should be allowed
	if !limiter.Allow(ip) {
		t.Error("First request should be allowed")
	}

	// Second request immediately should be denied
	if limiter.Allow(ip) {
		t.Error("Second request should be denied")
	}

	// Wait just under the window duration
	time.Sleep(40 * time.Millisecond)
	if limiter.Allow(ip) {
		t.Error("Request before window expiry should be denied")
	}

	// Wait for window to fully expire
	time.Sleep(20 * time.Millisecond) // Total wait: 60ms > 50ms window
	if !limiter.Allow(ip) {
		t.Error("Request after window expiry should be allowed")
	}
}

func TestFixedWindowLimiter_Allow_MultipleWindowResets(t *testing.T) {
	limiter := New(2, 30*time.Millisecond) // 2 requests per 30ms
	ip := "192.168.1.1"

	for window := 0; window < 3; window++ {
		// Use up the limit in this window
		if !limiter.Allow(ip) {
			t.Errorf("First request in window %d should be allowed", window)
		}
		if !limiter.Allow(ip) {
			t.Errorf("Second request in window %d should be allowed", window)
		}
		if limiter.Allow(ip) {
			t.Errorf("Third request in window %d should be denied", window)
		}

		// Wait for next window
		time.Sleep(40 * time.Millisecond)
	}
}

// Benchmark test to ensure performance is reasonable
func BenchmarkFixedWindowLimiter_Allow(b *testing.B) {
	limiter := New(1000000, time.Minute) // High limit to avoid denials
	ip := "192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		limiter.Allow(ip)
	}
}

func BenchmarkFixedWindowLimiter_Allow_DifferentIPs(b *testing.B) {
	limiter := New(1000000, time.Minute) // High limit to avoid denials

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i%256)
		limiter.Allow(ip)
	}
}
