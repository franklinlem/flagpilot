//go:build !linux

package main

import (
	"context"
	"errors"
)

func readPassword(context.Context, string) (string, error) {
	return "", errors.New("prompt de senha inicialmente suportado no Linux; use --password-stdin para automação")
}
