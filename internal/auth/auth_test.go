package auth

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/norrs/start/internal/config"
	dbgen "github.com/norrs/start/internal/db"
)

func TestLogoutRedirectRoutesThroughConfiguredProvider(t *testing.T) {
	m := Manager{config: config.Config{
		BaseURL:       "https://start.example.com",
		OIDCLogoutURL: "https://auth.example.com/logout",
	}}

	got, err := url.Parse(m.logoutRedirectURL())
	if err != nil {
		t.Fatal(err)
	}
	if got.Scheme != "https" || got.Host != "auth.example.com" || got.Path != "/logout" {
		t.Fatalf("unexpected logout endpoint %q", got.String())
	}
	if got.Query().Get("login_for") != "app" {
		t.Fatalf("unexpected login_for %q", got.Query().Get("login_for"))
	}
	if got.Query().Get("redirect_uri") != "https://start.example.com/" {
		t.Fatalf("unexpected redirect URI %q", got.Query().Get("redirect_uri"))
	}
}

func TestAdministrativeMutationsRequireFreshClaims(t *testing.T) {
	tests := []struct {
		name, method, path string
		want               bool
	}{
		{name: "catalog import", method: http.MethodPost, path: "/api/v1/catalog/import", want: true},
		{name: "history restore", method: http.MethodPost, path: "/api/v1/link-history/restore", want: true},
		{name: "link update", method: http.MethodPut, path: "/api/v1/links/1", want: true},
		{name: "catalog read", method: http.MethodGet, path: "/api/v1/catalog/import", want: false},
		{name: "preferences", method: http.MethodPut, path: "/api/v1/me/preferences", want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(test.method, test.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := requiresFreshAdminClaims(request); got != test.want {
				t.Fatalf("requiresFreshAdminClaims()=%v, want %v", got, test.want)
			}
		})
	}
}

func TestLogoutRedirectFallsBackWithoutProviderLogout(t *testing.T) {
	m := Manager{config: config.Config{BaseURL: "https://start.example.com"}}
	if got := m.logoutRedirectURL(); got != "/auth/login" {
		t.Fatalf("unexpected fallback %q", got)
	}
}

func TestClaimsFreshnessUsesShorterWindowForAdministrators(t *testing.T) {
	now := time.Date(2026, 8, 29, 18, 0, 0, 0, time.UTC)
	user := dbgen.User{ClaimsRefreshedAt: now.Add(-10 * time.Minute).Format(time.RFC3339Nano)}
	cfg := config.Config{ClaimMaxAge: 15 * time.Minute, AdminClaimMaxAge: 5 * time.Minute}
	if !claimsAreFresh(user, cfg, false, now) {
		t.Fatal("ordinary identity should remain fresh")
	}
	if claimsAreFresh(user, cfg, true, now) {
		t.Fatal("administrative mutation should require a fresh login")
	}
}

func TestClaimsFreshnessRejectsMalformedTimestamp(t *testing.T) {
	if claimsAreFresh(dbgen.User{ClaimsRefreshedAt: "invalid"}, config.Config{ClaimMaxAge: time.Hour}, false, time.Now()) {
		t.Fatal("malformed refresh time must fail closed")
	}
}
