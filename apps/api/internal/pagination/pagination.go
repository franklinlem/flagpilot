package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrValidation = errors.New("dados ou paginação inválidos")

type PageRequest struct {
	Limit  int
	Cursor string
}
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
type cursor struct {
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
	After string `json:"after"`
}

func (p PageRequest) After(kind, scope string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if p.Limit < 1 || p.Limit > 100 || len(p.Cursor) > 512 {
		return id, ErrValidation
	}
	if p.Cursor == "" {
		return id, nil
	}
	data, err := base64.RawURLEncoding.DecodeString(p.Cursor)
	if err != nil {
		return id, ErrValidation
	}
	var c cursor
	if json.Unmarshal(data, &c) != nil || c.Kind != kind || c.Scope != scope {
		return id, ErrValidation
	}
	id, err = identifiers.Parse(c.After)
	if err != nil {
		return id, ErrValidation
	}
	return id, nil
}
func NextCursor(kind, scope string, id pgtype.UUID) *string {
	data, _ := json.Marshal(cursor{Kind: kind, Scope: scope, After: identifiers.String(id)})
	encoded := base64.RawURLEncoding.EncodeToString(data)
	return &encoded
}
