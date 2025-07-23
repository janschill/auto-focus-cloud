package ratelimit

import (
	"sync"
	"time"
)

type RateLimit interface {
	Allow(addr string) bool
}

type WindowData struct {
	count       int
	windowStart time.Time
}

type FixedWindowLimitter struct {
	maxRequests int
	window      time.Duration
	requests    map[string]*WindowData
	mutex       sync.Mutex
}

func New(maxRequests int, interval time.Duration) RateLimit {
	return &FixedWindowLimitter{
		maxRequests: maxRequests,
		window:      interval,
		requests:    make(map[string]*WindowData),
	}
}

func (rl *FixedWindowLimitter) Allow(addr string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	wd := rl.requests[addr]

	// no data or we have requests but 10 minutes have passed
	if wd == nil || now.Sub(wd.windowStart) > rl.window {
		if rl.maxRequests == 0 {
			return false
		}

		wd = &WindowData{
			count:       1,
			windowStart: now,
		}
		rl.requests[addr] = wd

		return true
	}

	if wd.count >= rl.maxRequests {
		return false
	}
	wd.count++

	return true
}
