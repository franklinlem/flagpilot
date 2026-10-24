package auth

import (
	"sync"
	"time"
)

type entry struct {
	count   int
	expires time.Time
}

// Limiter é limitado em memória; rejeita novas chaves quando cheio. Uma réplica no MVP.
type Limiter struct {
	mu      sync.Mutex
	entries map[string]entry
	now     func() time.Time
}

func NewLimiter() *Limiter { return &Limiter{entries: make(map[string]entry), now: time.Now} }
func (l *Limiter) Allow(ip, email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for k, e := range l.entries {
		if !now.Before(e.expires) {
			delete(l.entries, k)
		}
	}
	keys := []string{"global", "ip:" + ip, "account:" + fmtHash(email)}
	limits := []int{120, 10, 10}
	windows := []time.Duration{time.Minute, 15 * time.Minute, 15 * time.Minute}
	for i, k := range keys {
		if e, ok := l.entries[k]; ok && e.count >= limits[i] {
			return false
		}
	}
	missing := 0
	for _, k := range keys {
		if _, ok := l.entries[k]; !ok {
			missing++
		}
	}
	if len(l.entries)+missing > 4096 {
		return false
	}
	for i, k := range keys {
		e, ok := l.entries[k]
		if !ok {
			e.expires = now.Add(windows[i])
		}
		e.count++
		l.entries[k] = e
	}
	return true
}
func fmtHash(value string) string { return string(tokenHash(value)) }
