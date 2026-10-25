package pagination

import (
	"github.com/franklinlem/flagpilot/apps/api/internal/identifiers"
	"testing"
)

func TestCursorBindingAndLimits(t *testing.T) {
	id := identifiers.New()
	encoded := NextCursor("members", "org:user", id)
	got, err := (PageRequest{Limit: 1, Cursor: *encoded}).After("members", "org:user")
	if err != nil || got != id {
		t.Fatal("cursor não round-trip")
	}
	for _, p := range []PageRequest{{Limit: 0}, {Limit: 101}, {Limit: 1, Cursor: "invalid"}, {Limit: 1, Cursor: *encoded}} {
		if _, err := p.After("organizations", "other-user"); err == nil {
			t.Fatal("cursor ou limite inválido aceito")
		}
	}
}
