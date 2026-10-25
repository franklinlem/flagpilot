package flags

import (
	"strings"
	"testing"
)

func TestCatalogFilter(t *testing.T) {
	for _, filter := range []Filter{{Query: "%"}, {Query: "quote'"}, {Query: "bad key"}, {Query: strings.Repeat("a", 81)}, {Status: "enabled"}} {
		if _, err := filter.Normalize(); err == nil {
			t.Fatal("filtro inválido aceito")
		}
	}
	value, err := (Filter{Query: " NEW_CHECKOUT "}).Normalize()
	if err != nil || value.Query != "new_checkout" || value.Status != "all" {
		t.Fatal("normalização incorreta")
	}
}
