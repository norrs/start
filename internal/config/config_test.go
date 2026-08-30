package config

import "testing"

func TestDevelopmentDefaultsValidate(t *testing.T) {
	t.Setenv("STARTPAGE_DEV_MODE", "true")
	t.Setenv("STARTPAGE_OIDC_ISSUER", "")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.DevMode {
		t.Fatal("expected development mode")
	}
	if c.ListenAddr != ":8080" {
		t.Fatalf("unexpected address %q", c.ListenAddr)
	}
}

func TestProductionNeedsOIDC(t *testing.T) {
	t.Setenv("STARTPAGE_DEV_MODE", "false")
	t.Setenv("STARTPAGE_OIDC_ISSUER", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProductionRejectsInsecureOIDCLogoutURL(t *testing.T) {
	c := Config{
		ListenAddr:       ":8080",
		BaseURL:          "https://start.example.com",
		DatabasePath:     "start.db",
		OIDCIssuer:       "https://auth.example.com",
		OIDCClientID:     "client",
		OIDCClientSecret: "secret",
		OIDCLogoutURL:    "http://auth.example.com/logout",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected insecure logout URL to be rejected")
	}
}
