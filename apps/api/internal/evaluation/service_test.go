package evaluation

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBooleanDecision(t *testing.T) {
	for _, test := range []struct {
		found, enabled bool
		rollout        int32
		value          bool
		reason         string
		err            error
	}{{false, true, 10000, false, "NOT_FOUND", nil}, {true, false, 10000, false, "DISABLED", nil}, {true, true, 0, false, "DISABLED", nil}, {true, true, 10000, true, "ENABLED", nil}, {true, true, 5000, false, "MISSING_CONTEXT", nil}, {true, true, -1, false, "", ErrUnavailable}, {true, true, 10001, false, "", ErrUnavailable}} {
		value, reason, err := Decide(test.found, test.enabled, test.rollout, nil)
		if value != test.value || reason != test.reason || !errors.Is(err, test.err) {
			t.Fatal("decisão booleana incorreta")
		}
	}
}
func TestContextAndKeyBounds(t *testing.T) {
	for _, key := range []string{"", "Invalid", "bad key", strings.Repeat("a", 81)} {
		if valid(Request{FlagKey: key}) {
			t.Fatal("chave inválida aceita")
		}
	}
	for _, key := range []string{"", strings.Repeat("x", 257), strings.Repeat("á", 129), "nul\x00value"} {
		if valid(Request{FlagKey: "valid", Context: &Context{BucketingKey: &key}}) {
			t.Fatal("contexto fora dos limites")
		}
	}
	key := strings.Repeat("á", 128)
	if !valid(Request{FlagKey: "valid", Context: &Context{BucketingKey: &key}}) {
		t.Fatal("limite UTF-8 exato rejeitado")
	}
}
func TestLimiterBoundsAndExpiry(t *testing.T) {
	clock := time.Now()
	l := NewLimiter()
	l.now = func() time.Time { return clock }
	for range 100 {
		if !l.Allow("peer", "credential") {
			t.Fatal("limite antecipado")
		}
	}
	if l.Allow("peer", "credential") {
		t.Fatal("limite por token ignorado")
	}
	for i := range 20 {
		if !l.Allow("peer", fmt.Sprint(i)) {
			t.Fatal("limite por peer antecipado")
		}
	}
	if l.Allow("peer", "another") {
		t.Fatal("rotação de token contornou peer")
	}
	clock = clock.Add(time.Second)
	if !l.Allow("peer", "credential") {
		t.Fatal("janela não expirou")
	}
	for i := range 4096 {
		l.entries[fmt.Sprint(i)] = bucket{until: clock.Add(time.Second)}
	}
	if l.Allow("other", "new") {
		t.Fatal("mapa cheio aceitou novas chaves")
	}
}
func TestLimiterConcurrent(t *testing.T) {
	l := NewLimiter()
	instant := time.Now()
	l.now = func() time.Time { return instant }
	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l.Allow("peer", "credential") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 100 {
		t.Fatal("limite concorrente ignorado")
	}
}
