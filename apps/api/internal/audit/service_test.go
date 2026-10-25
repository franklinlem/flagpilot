package audit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProjectionRejectsUnknownNestedAndSensitiveFields(t *testing.T) {
	for _, data := range []string{`{"key":"checkout","enabled":true,"secret":"forbidden","salt":"forbidden","token_hash":"forbidden","context":{"bucketingKey":"forbidden"},"description":{"password":"forbidden"}}`, `{"description":"Texto <script> seguro como texto","key":"checkout"}`} {
		projected := redact("flag", []byte(data))
		b, _ := json.Marshal(projected)
		if strings.Contains(string(b), "forbidden") || string(projected["key"]) != `"checkout"` {
			t.Fatal("projeção expôs campo indevido")
		}
	}
	for _, data := range []string{`[]`, `null`, `{"invalid"`, strings.Repeat("x", 32769)} {
		if redact("flag", []byte(data)) != nil {
			t.Fatal("documento inválido aceito")
		}
	}
	if redact("future_kind", []byte(`{"name":"unknown"}`)) != nil {
		t.Fatal("tipo desconhecido exposto")
	}
}
