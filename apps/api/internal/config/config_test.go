package config

import (
	"strings"
	"testing"
)

func TestConfigValidationAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		valid  bool
	}{
		{"defaults", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db?sslmode=disable"}, true},
		{"invalid port", map[string]string{"HTTP_ADDR": "localhost:99999", "DATABASE_URL": "postgres://user:secret@localhost/db"}, false},
		{"missing database", map[string]string{}, false},
		{"invalid secret url", map[string]string{"DATABASE_URL": "postgres://user:supersecret@%broken"}, false},
		{"unknown environment", map[string]string{"APP_ENV": "prod", "DATABASE_URL": "postgres://user:secret@localhost/db"}, false},
		{"production insecure", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://user:secret@localhost/db?sslmode=disable"}, false},
		{"production TLS", map[string]string{"APP_ENV": "production", "DATABASE_URL": "postgres://user:secret@localhost/db?sslmode=verify-full", "PUBLIC_BASE_URL": "https://flags.example.invalid"}, true},
		{"remote insecure origin", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db", "PUBLIC_BASE_URL": "http://example.invalid"}, false},
		{"wildcard origin", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db", "ALLOWED_ORIGINS": "*"}, false},
		{"invalid session ttl", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db", "SESSION_TTL": "0s"}, false},
		{"negative timeout", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db", "DB_TIMEOUT": "-1s"}, false},
		{"excessive timeout", map[string]string{"DATABASE_URL": "postgres://user:secret@localhost/db", "DB_TIMEOUT": "2m"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := load(func(k string) string { return tc.values[k] })
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if err != nil && strings.Contains(err.Error(), "secret") {
				t.Fatal("erro expôs credencial")
			}
		})
	}
}
func TestMetricsOptInAndAddress(t *testing.T) {
	for _, value := range []string{"", "127.0.0.1:9090", "0.0.0.0:9090", "[::1]:9090", "10.0.0.2:9090"} {
		cfg, err := load(func(key string) string {
			if key == "DATABASE_URL" {
				return "postgres://user:secret@localhost/db"
			}
			if key == "METRICS_ADDR" {
				return value
			}
			return ""
		})
		if err != nil || cfg.MetricsAddr != value {
			t.Fatal("configuração de métricas válida rejeitada")
		}
	}
	for _, value := range []string{":9090", "localhost:9090", "8.8.8.8:9090", "127.0.0.1:8080", "127.0.0.1:0", "bad"} {
		_, err := load(func(key string) string {
			if key == "DATABASE_URL" {
				return "postgres://user:secret@localhost/db"
			}
			if key == "METRICS_ADDR" {
				return value
			}
			return ""
		})
		if err == nil {
			t.Fatal("endereço de métricas inseguro/inválido aceito")
		}
	}
}
