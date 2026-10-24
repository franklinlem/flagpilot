package config

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment     string
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        slog.Level
	DBTimeout       time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) { return load(os.Getenv) }

func load(get func(string) string) (Config, error) {
	c := Config{Environment: get("APP_ENV"), HTTPAddr: get("HTTP_ADDR"), DatabaseURL: get("DATABASE_URL"), DBTimeout: 2 * time.Second, ShutdownTimeout: 10 * time.Second}
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Environment != "development" && c.Environment != "test" && c.Environment != "production" {
		return c, errors.New("APP_ENV inválido")
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(c.HTTPAddr)
	if err != nil {
		return c, errors.New("HTTP_ADDR inválido")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return c, errors.New("porta HTTP_ADDR inválida")
	}
	u, err := url.Parse(c.DatabaseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.User == nil || u.User.Username() == "" || u.Path == "" || u.Path == "/" {
		return c, errors.New("DATABASE_URL obrigatório e deve ser uma URL PostgreSQL válida")
	}
	if c.Environment == "production" && u.Query().Get("sslmode") != "verify-full" {
		return c, errors.New("produção requer DATABASE_URL com sslmode=verify-full")
	}
	if v := get("LOG_LEVEL"); v != "" {
		if err := c.LogLevel.UnmarshalText([]byte(v)); err != nil {
			return c, errors.New("LOG_LEVEL inválido")
		}
	}
	for _, v := range []struct {
		key    string
		target *time.Duration
	}{{"DB_TIMEOUT", &c.DBTimeout}, {"SHUTDOWN_TIMEOUT", &c.ShutdownTimeout}} {
		if s := get(v.key); s != "" {
			d, err := time.ParseDuration(s)
			if err != nil || d <= 0 || d > time.Minute {
				return c, errors.New(v.key + " deve estar entre 0 e 1 minuto")
			}
			*v.target = d
		}
	}
	return c, nil
}
