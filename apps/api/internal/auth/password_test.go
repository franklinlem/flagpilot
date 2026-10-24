package auth

import (
	"strings"
	"testing"
)

func TestPasswordAndCSRF(t *testing.T) {
	password := "test-only-password-123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword(password, hash) || VerifyPassword("wrong-password", hash) {
		t.Fatal("verificação inválida")
	}
	other, err := HashPassword(password)
	if err != nil || hash == other {
		t.Fatal("salt não aleatório")
	}
	for _, value := range []string{"", "short", strings.Repeat(" ", 12), strings.Repeat("a", 257)} {
		if _, err := HashPassword(value); err == nil {
			t.Fatal("senha inválida aceita")
		}
	}
	for _, value := range []string{"broken", strings.Replace(hash, "m=19456", "m=999999999", 1), strings.Replace(hash, "v=19", "v=18", 1), strings.Repeat("a", 300)} {
		if VerifyPassword(password, value) {
			t.Fatal("hash inválido aceito")
		}
	}
	a, _ := randomToken()
	b, _ := randomToken()
	if !ValidCSRF(a, csrf(a)) || ValidCSRF(b, csrf(a)) || ValidCSRF(a, "") || ValidCSRF("invalid", csrf(a)) {
		t.Fatal("CSRF não vinculado à sessão")
	}
	for _, email := range []string{"a", "Name <one@example.invalid>", "one@example.invalid\nInjected: value"} {
		if _, ok := NormalizeEmail(email); ok {
			t.Fatal("email inválido aceito")
		}
	}
	if value, ok := NormalizeEmail(" ONE@example.invalid "); !ok || value != "one@example.invalid" {
		t.Fatal("normalização inválida")
	}
}
