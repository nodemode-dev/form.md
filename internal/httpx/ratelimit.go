package httpx

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu      sync.Mutex
	perSec  float64
	burst   float64
	buckets map[string]*bucket
	swept   time.Time
}

func NewLimiter(perMinute, burst int) *Limiter {
	return &Limiter{perSec: float64(perMinute) / 60, burst: float64(burst), buckets: map[string]*bucket{}}
}

func (l *Limiter) refill(key string, now time.Time) *bucket {
	if now.Sub(l.swept) > time.Minute {
		l.swept = now
		for k, b := range l.buckets {
			if b.tokens+now.Sub(b.last).Seconds()*l.perSec >= l.burst {
				delete(l.buckets, k)
			}
		}
	}
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
		return b
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSec)
	b.last = now
	return b
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.refill(key, time.Now())
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (l *Limiter) Exhausted(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.refill(key, time.Now()).tokens < 1
}

func (l *Limiter) Spend(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	b := l.refill(key, time.Now())
	b.tokens = max(0, b.tokens-1)
}
