package ratelimiter

import (
	"sync"
	"time"
)

// Limiter implements a token-bucket rate limiter.
// Tokens refill at `rate` per `interval`. Capacity is burst; refill occurs in whole intervals.
type Limiter struct {
	mu       sync.Mutex
	rate     int
	capacity int
	interval time.Duration
	tokens   int
	last     time.Time
}

// New creates a limiter that allows `rate` requests per `interval`
// with an initial burst of `burst` tokens.
func New(rate int, interval time.Duration, burst int) *Limiter {
	if rate <= 0 || burst <= 0 || interval <= 0 {
		panic("rate, interval and burst must be positive")
	}
	return &Limiter{rate: rate, capacity: burst, interval: interval, tokens: burst, last: time.Now()}
}

func (l *Limiter) allowN(n int) bool {
	if n <= 0 {
		return true
	}
	return l.allowAt(n, time.Now())
}

func (l *Limiter) allowAt(n int, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	elapsed := now.Sub(l.last)
	if elapsed >= l.interval {
		periods := elapsed / l.interval
		missing := l.capacity - l.tokens
		if periods >= time.Duration(1+(missing-1)/l.rate) {
			l.tokens = l.capacity
		} else {
			l.tokens += int(periods) * l.rate
		}
		l.last = now.Add(-(elapsed % l.interval))
	}

	if l.tokens >= n {
		l.tokens -= n
		return true
	}
	return false
}

// Allow consumes one token. Returns true if the request is allowed.
func (l *Limiter) Allow() bool { return l.allowN(1) }
