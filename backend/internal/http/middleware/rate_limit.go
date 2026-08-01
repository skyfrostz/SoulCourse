package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimitRule struct {
	Name    string
	Limit   int
	Window  time.Duration
	KeyFunc func(*gin.Context) string
}

type rateLimitBucket struct {
	windowStart time.Time
	count       int
}

type RateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[string]rateLimitBucket
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		now:     time.Now,
		buckets: map[string]rateLimitBucket{},
	}
}

func (l *RateLimiter) Limit(rule RateLimitRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if rule.Limit <= 0 || rule.Window <= 0 {
			c.Next()
			return
		}
		keyFunc := rule.KeyFunc
		if keyFunc == nil {
			keyFunc = ClientIPKey
		}
		allowed, retryAfter := l.allow(rule, keyFunc(c))
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds()+0.5)))
			AbortWithError(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) allow(rule RateLimitRule, key string) (bool, time.Duration) {
	now := l.now()
	bucketKey := rule.Name + ":" + key

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[bucketKey]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= rule.Window {
		l.buckets[bucketKey] = rateLimitBucket{windowStart: now, count: 1}
		l.prune(now)
		return true, 0
	}
	if bucket.count >= rule.Limit {
		return false, rule.Window - now.Sub(bucket.windowStart)
	}
	bucket.count++
	l.buckets[bucketKey] = bucket
	return true, 0
}

func (l *RateLimiter) prune(now time.Time) {
	for key, bucket := range l.buckets {
		if now.Sub(bucket.windowStart) > 10*time.Minute {
			delete(l.buckets, key)
		}
	}
}

func ClientIPKey(c *gin.Context) string {
	return c.ClientIP()
}

func ClientIPAndUserKey(c *gin.Context) string {
	if userID := CurrentUserID(c); userID != nil {
		return fmt.Sprintf("%s:user:%d", c.ClientIP(), *userID)
	}
	return c.ClientIP()
}
