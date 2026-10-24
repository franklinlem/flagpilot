package identifiers

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
)

func New() pgtype.UUID {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("random source unavailable")
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return pgtype.UUID{Bytes: b, Valid: true}
}
func String(id pgtype.UUID) string {
	b := id.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func Parse(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if len(value) != 36 {
		return id, errors.New("UUID inválido")
	}
	err := id.Scan(value)
	return id, err
}
