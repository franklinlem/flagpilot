package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Parâmetros mínimos Argon2id recomendados pelo OWASP, com custo limitado no login.
const memoryKiB uint32 = 19 * 1024
const iterations uint32 = 2
const parallelism uint8 = 1

func ValidatePassword(password string) bool {
	return utf8.ValidString(password) && utf8.RuneCountInString(password) >= 12 && len(password) <= 256 && strings.TrimSpace(password) != ""
}

func HashPassword(password string) (string, error) {
	if !ValidatePassword(password) {
		return "", ErrValidation
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", ErrUnavailable
	}
	key := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, 32)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memoryKiB, iterations, parallelism, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func VerifyPassword(password, encoded string) bool {
	// Não aceitar parâmetros arbitrários vindos do banco (custo/memória sem limite).
	if len(password) > 256 || len(encoded) > 256 {
		return false
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != fmt.Sprintf("m=%d,t=%d,p=%d", memoryKiB, iterations, parallelism) {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(want) != 32 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, iterations, memoryKiB, parallelism, 32)
	return subtle.ConstantTimeCompare(got, want) == 1
}
