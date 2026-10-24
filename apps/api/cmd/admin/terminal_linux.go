//go:build linux

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// Desligar echo antes de iniciar a leitura evita uma corrida entre SIGINT e
// uma goroutine que ainda não alterou o terminal. Somente esta função restaura estado.
func readPassword(ctx context.Context, prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return "", errors.New("terminal indisponível")
	}
	hidden := *state
	hidden.Lflag &^= unix.ECHO
	hidden.Lflag |= unix.ICANON | unix.ISIG
	hidden.Iflag |= unix.ICRNL
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, &hidden); err != nil {
		return "", errors.New("não foi possível ocultar a senha")
	}
	defer func() { _ = unix.IoctlSetTermios(fd, unix.TCSETS, state); fmt.Fprintln(os.Stderr) }()
	fmt.Fprint(os.Stderr, prompt)
	type result struct {
		value string
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := bufio.NewReaderSize(os.Stdin, 512).ReadSlice('\n')
		ch <- result{string(value), err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return "", errors.New("senha não lida ou excede o limite permitido")
		}
		return strings.TrimSuffix(strings.TrimSuffix(r.value, "\n"), "\r"), nil
	}
}
