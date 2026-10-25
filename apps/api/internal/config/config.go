package config

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment     string
	HTTPAddr        string
	MetricsAddr     string
	DatabaseURL     string
	LogLevel        slog.Level
	DBTimeout       time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
	SessionTTL      time.Duration
	SecureCookies   bool
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
	c.MetricsAddr = get("METRICS_ADDR")
	if c.MetricsAddr != "" {
		host, metricsPort, err := net.SplitHostPort(c.MetricsAddr)
		metricsNumber, numberErr := strconv.Atoi(metricsPort)
		ip := net.ParseIP(host)
		if err != nil || numberErr != nil || ip == nil || (!ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified()) || metricsNumber < 1 || metricsNumber > 65535 || metricsNumber == p {
			return c, errors.New("METRICS_ADDR deve usar IP local/privado explícito e porta distinta válida")
		}
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

	c.SessionTTL = 12 * time.Hour
	if value := get("SESSION_TTL"); value != "" {
		c.SessionTTL, err = time.ParseDuration(value)
		if err != nil || c.SessionTTL < time.Hour || c.SessionTTL > 7*24*time.Hour {
			return c, errors.New("SESSION_TTL deve estar entre 1h e 168h")
		}
	}
	base := get("PUBLIC_BASE_URL")
	if base == "" {
		if c.Environment == "production" {
			return c, errors.New("PUBLIC_BASE_URL obrigatório em produção")
		}
		base = "http://127.0.0.1:5173"
	}
	origins := get("ALLOWED_ORIGINS")
	if origins == "" {
		origins = base
	}
	for _, value := range append(strings.Split(origins, ","), base) {
		value = strings.TrimSpace(value)
		origin, err := url.Parse(value)
		if err != nil || origin.Host == "" || origin.User != nil || (origin.Scheme != "https" && origin.Scheme != "http") || origin.RawQuery != "" || origin.Fragment != "" || (origin.Path != "" && origin.Path != "/") {
			return c, errors.New("origem deve conter apenas scheme e host")
		}
		if origin.Scheme == "http" {
			ip := net.ParseIP(origin.Hostname())
			if c.Environment == "production" || (origin.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
				return c, errors.New("HTTP permitido somente em loopback de desenvolvimento")
			}
		}
		c.AllowedOrigins = append(c.AllowedOrigins, origin.Scheme+"://"+origin.Host)
	}
	c.SecureCookies = c.Environment == "production" || strings.HasPrefix(base, "https://")
	return c, nil
}
