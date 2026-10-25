package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestErrorsDoNotExposeConnectionCredentials(t *testing.T) {
	for _, url := range []string{"", "unknown://operator:sentinel-password@127.0.0.1/private-db"} {
		var output bytes.Buffer
		if run(url, "/nonexistent-migrations", &output) != 1 {
			t.Fatal("invalid configuration must fail")
		}
		for _, secret := range []string{"sentinel-password", "operator", "private-db", "unknown://"} {
			if strings.Contains(output.String(), secret) {
				t.Fatal("migration error exposed connection details")
			}
		}
	}
}
