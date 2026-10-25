package evaluation

import (
	"crypto/sha256"
	"sync"
	"time"
)

type bucket struct {
	count int
	until time.Time
}

// Limiter tem memória limitada e usa somente o peer real, sem confiar em X-Forwarded-For.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]bucket
	now     func() time.Time
	sweep   time.Time
}

func NewLimiter() *Limiter { return &Limiter{entries: map[string]bucket{}, now: time.Now} }
func (l *Limiter) Allow(peer, secret string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if !now.Before(l.sweep) || len(l.entries) >= 4096 {
		for key, value := range l.entries {
			if !now.Before(value.until) {
				delete(l.entries, key)
			}
		}
		l.sweep = now.Add(time.Second)
	}
	hash := sha256.Sum256([]byte(secret))
	keys := []string{"global", "peer:" + peer, "credential:" + string(hash[:])}
	limits := []int{600, 120, 100}
	missing := 0
	for i, key := range keys {
		value, ok := l.entries[key]
		if ok && !now.Before(value.until) {
			delete(l.entries, key)
			ok = false
		}
		if ok && value.count >= limits[i] {
			return false
		}
		if !ok {
			missing++
		}
	}
	if len(l.entries)+missing > 4096 {
		return false
	}
	for _, key := range keys {
		value, ok := l.entries[key]
		if !ok {
			value.until = now.Add(time.Second)
		}
		value.count++
		l.entries[key] = value
	}
	return true
}
