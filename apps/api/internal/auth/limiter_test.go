package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestLoginLimits(t *testing.T) {
	for _, account := range []bool{false, true} {
		l := NewLimiter()
		now := time.Now()
		l.now = func() time.Time { return now }
		for i := range 10 {
			ip, email := "ip", "email"
			if account {
				ip = fmt.Sprint(i)
			} else {
				email = fmt.Sprint(i)
			}
			if !l.Allow(ip, email) {
				t.Fatal("limite prematuro")
			}
		}
		if l.Allow("ip", "email") {
			t.Fatal("limite não aplicado")
		}
		now = now.Add(16 * time.Minute)
		if !l.Allow("ip", "email") {
			t.Fatal("janela não expirou")
		}
	}
	l := NewLimiter()
	for i := range 120 {
		if !l.Allow(fmt.Sprint(i), fmt.Sprint(i)) {
			t.Fatal("global prematuro")
		}
	}
	if l.Allow("new", "new") {
		t.Fatal("limite global ausente")
	}
	l = NewLimiter()
	for i := range 4096 {
		l.entries[fmt.Sprint(i)] = entry{count: 1, expires: time.Now().Add(time.Hour)}
	}
	if l.Allow("new", "new") || len(l.entries) != 4096 {
		t.Fatal("mapa sem limite")
	}
}
