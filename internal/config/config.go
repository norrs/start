package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr           string
	BaseURL              string
	DatabasePath         string
	SeedPath             string
	StaticDir            string
	OIDCIssuer           string
	OIDCClientID         string
	OIDCClientSecret     string
	OIDCClientSecretFile string
	OIDCScopes           []string
	OIDCLogoutURL        string
	AdminGroups          []string
	TrustedProxies       []string
	MetricsEnabled       bool
	LogLevel             string
	DevMode              bool
	DevUserEmail         string
	DevUserName          string
	DevUserGroups        []string
	ClaimMaxAge          time.Duration
	AdminClaimMaxAge     time.Duration
	SessionIdleTimeout   time.Duration
	SessionLifetime      time.Duration
}

func Load() (Config, error) {
	c := Config{
		ListenAddr:           env("STARTPAGE_LISTEN_ADDR", ":8080"),
		BaseURL:              env("STARTPAGE_BASE_URL", "http://localhost:8080"),
		DatabasePath:         env("STARTPAGE_DATABASE_PATH", "./start.db"),
		SeedPath:             env("STARTPAGE_SEED_PATH", "./links.yaml"),
		StaticDir:            env("STARTPAGE_STATIC_DIR", "./web/dist"),
		OIDCIssuer:           strings.TrimSuffix(os.Getenv("STARTPAGE_OIDC_ISSUER"), "/"),
		OIDCClientID:         os.Getenv("STARTPAGE_OIDC_CLIENT_ID"),
		OIDCClientSecret:     os.Getenv("STARTPAGE_OIDC_CLIENT_SECRET"),
		OIDCClientSecretFile: os.Getenv("STARTPAGE_OIDC_CLIENT_SECRET_FILE"),
		OIDCScopes:           csv(env("STARTPAGE_OIDC_SCOPES", "openid,profile,email,groups")),
		OIDCLogoutURL:        strings.TrimSpace(os.Getenv("STARTPAGE_OIDC_LOGOUT_URL")),
		AdminGroups:          csv(env("STARTPAGE_ADMIN_GROUPS", "start-admins")),
		TrustedProxies:       csv(os.Getenv("STARTPAGE_TRUSTED_PROXIES")),
		LogLevel:             env("STARTPAGE_LOG_LEVEL", "info"),
		DevUserEmail:         env("STARTPAGE_DEV_USER_EMAIL", "developer@localhost"),
		DevUserName:          env("STARTPAGE_DEV_USER_NAME", "Local Developer"),
		DevUserGroups:        csv(env("STARTPAGE_DEV_USER_GROUPS", "start-admins")),
	}

	var err error
	if c.MetricsEnabled, err = boolEnv("STARTPAGE_METRICS_ENABLED", false); err != nil {
		return Config{}, err
	}
	if c.DevMode, err = boolEnv("STARTPAGE_DEV_MODE", c.OIDCIssuer == ""); err != nil {
		return Config{}, err
	}
	if c.ClaimMaxAge, err = durationEnv("STARTPAGE_CLAIM_MAX_AGE", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if c.AdminClaimMaxAge, err = durationEnv("STARTPAGE_ADMIN_CLAIM_MAX_AGE", 5*time.Minute); err != nil {
		return Config{}, err
	}
	if c.SessionIdleTimeout, err = durationEnv("STARTPAGE_SESSION_IDLE_TIMEOUT", 8*time.Hour); err != nil {
		return Config{}, err
	}
	if c.SessionLifetime, err = durationEnv("STARTPAGE_SESSION_LIFETIME", 24*time.Hour); err != nil {
		return Config{}, err
	}

	if c.OIDCClientSecret == "" && c.OIDCClientSecretFile != "" {
		b, readErr := os.ReadFile(c.OIDCClientSecretFile)
		if readErr != nil {
			return Config{}, fmt.Errorf("read OIDC client secret file: %w", readErr)
		}
		c.OIDCClientSecret = strings.TrimSpace(string(b))
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return errors.New("STARTPAGE_LISTEN_ADDR must not be empty")
	}
	if c.DatabasePath == "" {
		return errors.New("STARTPAGE_DATABASE_PATH must not be empty")
	}
	if !c.DevMode {
		if c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCClientSecret == "" {
			return errors.New("OIDC issuer, client ID, and client secret are required outside development mode")
		}
		if !strings.HasPrefix(c.BaseURL, "https://") {
			return errors.New("STARTPAGE_BASE_URL must use HTTPS outside development mode")
		}
		if c.OIDCLogoutURL != "" {
			logoutURL, err := url.Parse(c.OIDCLogoutURL)
			if err != nil || logoutURL.Scheme != "https" || logoutURL.Host == "" {
				return errors.New("STARTPAGE_OIDC_LOGOUT_URL must be an absolute HTTPS URL outside development mode")
			}
		}
	}
	if c.AdminClaimMaxAge > c.ClaimMaxAge {
		return errors.New("admin claim max age cannot exceed ordinary claim max age")
	}
	return nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
func csv(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
func boolEnv(k string, fallback bool) (bool, error) {
	v := os.Getenv(k)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: %w", k, err)
	}
	return b, nil
}
func durationEnv(k string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(k)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", k, err)
	}
	return d, nil
}
