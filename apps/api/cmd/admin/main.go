package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
	"github.com/franklinlem/flagpilot/apps/api/internal/config"
	"github.com/franklinlem/flagpilot/apps/api/internal/storage/postgres"
	"golang.org/x/term"
)

func main() { os.Exit(run()) }
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if len(os.Args) < 2 || (os.Args[1] != "bootstrap" && os.Args[1] != "reset-password") {
		fmt.Fprintln(os.Stderr, "Uso: admin bootstrap|reset-password [--email EMAIL] [--org NOME --slug SLUG] [--password-stdin]")
		return 1
	}
	fs := flag.NewFlagSet(os.Args[1], flag.ContinueOnError)
	email := fs.String("email", "", "Email do administrador")
	name := fs.String("org", "", "Nome da organização (bootstrap)")
	slug := fs.String("slug", "", "Slug da organização (bootstrap)")
	stdin := fs.Bool("password-stdin", false, "Ler uma senha por stdin para automação; nunca usar argumento de senha")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return 1
	}
	if fs.NArg() != 0 {
		return 1
	}
	reader := bufio.NewReader(os.Stdin)
	prompt := func(label string, target *string) error {
		if *target != "" {
			return nil
		}
		if *stdin {
			return errors.New("informe email/org/slug como flags ao usar --password-stdin")
		}
		fmt.Fprint(os.Stderr, label)
		type result struct {
			s   string
			err error
		}
		ch := make(chan result, 1)
		go func() { value, err := reader.ReadString('\n'); ch <- result{value, err} }()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-ch:
			if r.err != nil {
				return errors.New("entrada encerrada")
			}
			*target = strings.TrimSpace(r.s)
			return nil
		}
	}
	for _, v := range []struct {
		label  string
		target *string
	}{{"Email: ", email}} {
		if err := prompt(v.label, v.target); err != nil {
			return fail(err)
		}
	}
	if os.Args[1] == "bootstrap" {
		for _, v := range []struct {
			label  string
			target *string
		}{{"Nome da organização: ", name}, {"Slug (ex.: minha-equipe): ", slug}} {
			if err := prompt(v.label, v.target); err != nil {
				return fail(err)
			}
		}
	}
	var password string
	if *stdin {
		value, err := io.ReadAll(io.LimitReader(os.Stdin, 258))
		if err != nil {
			return fail(errors.New("falha ao ler senha"))
		}
		password = strings.TrimSuffix(strings.TrimSuffix(string(value), "\n"), "\r")
	} else {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fail(errors.New("senha exige terminal; automação deve usar --password-stdin"))
		}
		first, err := readPassword(ctx, "Senha (mínimo 12 caracteres): ")
		if err != nil {
			return fail(err)
		}
		second, err := readPassword(ctx, "Confirme a senha: ")
		if err != nil {
			return fail(err)
		}
		if first != second {
			return fail(errors.New("senhas não coincidem"))
		}
		password = first
	}
	if !auth.ValidatePassword(password) {
		return fail(errors.New("senha deve ter ao menos 12 caracteres, até 256 bytes e não ser vazia"))
	}
	cfg, err := config.Load()
	if err != nil {
		return fail(err)
	}
	store, err := postgres.New(ctx, cfg.DatabaseURL, cfg.DBTimeout)
	if err != nil {
		return fail(err)
	}
	defer store.Close()
	if err = store.Ready(ctx); err != nil {
		return fail(errors.New("banco não está pronto; execute migrations antes do comando"))
	}
	service, err := auth.New(store.Pool, cfg.SessionTTL, cfg.DBTimeout)
	if err != nil {
		return fail(err)
	}
	if os.Args[1] == "bootstrap" {
		err = service.Bootstrap(ctx, *email, password, *name, *slug)
	} else {
		err = service.ResetPassword(ctx, *email, password)
	}
	if err != nil {
		return fail(err)
	}
	fmt.Println("Operação concluída e auditada. Use a conta na tela de login.")
	return 0
}
func fail(err error) int {
	if errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Operação interrompida.")
		return 130
	}
	fmt.Fprintln(os.Stderr, err)
	return 1
}
