package middleware

import (
	"net/http"
	"sync"
	"time"
)

// tokenBucket implements a simple per-IP token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(maxTokens, refillRatePerSecond float64) *tokenBucket {
	return &tokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRatePerSecond,
		lastRefill: time.Now(),
	}
}

func (b *tokenBucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// ipRateLimiter manages per-IP token buckets with periodic cleanup.
type ipRateLimiter struct {
	buckets        map[string]*tokenBucket
	mu             sync.Mutex
	maxTokens      float64
	refillRate     float64
	cleanupTicker  *time.Ticker
	cleanupAfter   time.Duration
	bucketLastSeen map[string]time.Time
}

func newIPRateLimiter(maxTokens, refillRatePerSecond float64) *ipRateLimiter {
	l := &ipRateLimiter{
		buckets:        make(map[string]*tokenBucket),
		maxTokens:      maxTokens,
		refillRate:     refillRatePerSecond,
		cleanupAfter:   10 * time.Minute,
		bucketLastSeen: make(map[string]time.Time),
	}
	go l.cleanup()
	return l
}

func (l *ipRateLimiter) getBucket(ip string) *tokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.buckets[ip]
	if !ok {
		b = newTokenBucket(l.maxTokens, l.refillRate)
		l.buckets[ip] = b
	}
	l.bucketLastSeen[ip] = time.Now()
	return b
}

func (l *ipRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-l.cleanupAfter)
		for ip, last := range l.bucketLastSeen {
			if last.Before(cutoff) {
				delete(l.buckets, ip)
				delete(l.bucketLastSeen, ip)
			}
		}
		l.mu.Unlock()
	}
}

// authLimiter: 10 requests burst, 1 request per 10 seconds (6/min) refill.
var authLimiter = newIPRateLimiter(10, 0.1)

// RateLimitAuth is a middleware that enforces rate limiting on auth endpoints.
// It uses the real IP set by chi's RealIP middleware (RemoteAddr after that).
func RateLimitAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Use X-Forwarded-For / X-Real-IP if already resolved by RealIP middleware
		if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
			ip = realIP
		}
		bucket := authLimiter.getBucket(ip)
		if !bucket.allow() {
			w.Header().Set("Retry-After", "10")
			http.Error(w, `{"error":"rate_limited","message":"too many requests, please try again later"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
