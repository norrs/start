package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/norrs/start/internal/catalog"
	"github.com/norrs/start/internal/config"
	dbgen "github.com/norrs/start/internal/db"
	"github.com/norrs/start/internal/policy"
)

const identityKey = "start.identity"
const userKey = "start.user"

type Manager struct {
	config   config.Config
	store    *catalog.Store
	sessions *scs.SessionManager
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    *oauth2.Config
}

type Claims struct {
	Issuer            string   `json:"iss"`
	Subject           string   `json:"sub"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
	Nonce             string   `json:"nonce"`
}

func New(ctx context.Context, cfg config.Config, store *catalog.Store, sessions *scs.SessionManager) (*Manager, error) {
	m := &Manager{config: cfg, store: store, sessions: sessions}
	if cfg.DevMode {
		return m, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.OIDCIssuer)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}
	m.provider = provider
	m.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.OIDCClientID})
	m.oauth = &oauth2.Config{ClientID: cfg.OIDCClientID, ClientSecret: cfg.OIDCClientSecret, Endpoint: provider.Endpoint(), RedirectURL: strings.TrimSuffix(cfg.BaseURL, "/") + "/auth/callback", Scopes: cfg.OIDCScopes}
	return m, nil
}

func (m *Manager) Register(r *gin.Engine) {
	r.GET("/auth/login", m.login)
	r.GET("/auth/callback", m.callback)
	r.POST("/auth/logout", m.logout)
}

func (m *Manager) Require() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.config.DevMode {
			u, err := m.store.EnsureUser(c.Request.Context(), "urn:start:development", m.config.DevUserEmail, m.config.DevUserEmail, m.config.DevUserName, m.config.DevUserGroups)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, errorBody("identity_failed", err.Error()))
				return
			}
			m.sessions.Put(c.Request.Context(), "user_id", u.ID)
		}
		userID := m.sessions.GetInt64(c.Request.Context(), "user_id")
		if userID == 0 {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("unauthenticated", "Sign in is required"))
				return
			}
			target := c.Request.URL.RequestURI()
			c.Redirect(http.StatusFound, "/auth/login?return="+url.QueryEscape(target))
			c.Abort()
			return
		}
		id, user, err := m.store.Identity(c.Request.Context(), userID, m.config.AdminGroups)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("identity_expired", "The session identity is no longer available"))
			return
		}
		if !claimsAreFresh(user, m.config, requiresFreshAdminClaims(c.Request), time.Now()) {
			_ = m.sessions.Destroy(c.Request.Context())
			c.AbortWithStatusJSON(http.StatusUnauthorized, errorBody("identity_stale", "Identity claims must be refreshed by signing in again"))
			return
		}
		c.Set(identityKey, id)
		c.Set(userKey, user)
		c.Next()
	}
}

func claimsAreFresh(user dbgen.User, cfg config.Config, administrativeMutation bool, now time.Time) bool {
	refreshed, err := time.Parse(time.RFC3339Nano, user.ClaimsRefreshedAt)
	if err != nil {
		return false
	}
	maxAge := cfg.ClaimMaxAge
	if administrativeMutation {
		maxAge = cfg.AdminClaimMaxAge
	}
	return maxAge > 0 && now.Sub(refreshed) <= maxAge
}

func requiresFreshAdminClaims(request *http.Request) bool {
	if request.Method == http.MethodGet || request.Method == http.MethodHead || request.Method == http.MethodOptions {
		return false
	}
	return strings.HasPrefix(request.URL.Path, "/api/v1/links") ||
		request.URL.Path == "/api/v1/link-history/restore" ||
		request.URL.Path == "/api/v1/catalog/import"
}

func Identity(c *gin.Context) (policy.Identity, bool) {
	v, ok := c.Get(identityKey)
	if !ok {
		return policy.Identity{}, false
	}
	id, ok := v.(policy.Identity)
	return id, ok
}
func User(c *gin.Context) (dbgen.User, bool) {
	v, ok := c.Get(userKey)
	if !ok {
		return dbgen.User{}, false
	}
	u, ok := v.(dbgen.User)
	return u, ok
}

func (m *Manager) login(c *gin.Context) {
	if m.config.DevMode {
		c.Redirect(http.StatusFound, safeReturn(c.Query("return")))
		return
	}
	state := randomToken(32)
	nonce := randomToken(32)
	verifier := randomToken(48)
	ret := safeReturn(c.Query("return"))
	m.sessions.Put(c.Request.Context(), "oidc_state", state)
	m.sessions.Put(c.Request.Context(), "oidc_nonce", nonce)
	m.sessions.Put(c.Request.Context(), "oidc_verifier", verifier)
	m.sessions.Put(c.Request.Context(), "oidc_return", ret)
	challenge := sha256.Sum256([]byte(verifier))
	location := m.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.SetAuthURLParam("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:])), oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	c.Redirect(http.StatusFound, location)
}

func (m *Manager) callback(c *gin.Context) {
	ctx := c.Request.Context()
	state := m.sessions.PopString(ctx, "oidc_state")
	nonce := m.sessions.PopString(ctx, "oidc_nonce")
	verifier := m.sessions.PopString(ctx, "oidc_verifier")
	ret := safeReturn(m.sessions.PopString(ctx, "oidc_return"))
	if state == "" || state != c.Query("state") {
		htmlError(c, http.StatusBadRequest, "Login failed", "The login state was invalid or expired.")
		return
	}
	token, err := m.oauth.Exchange(ctx, c.Query("code"), oauth2.SetAuthURLParam("code_verifier", verifier))
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("oidc_exchange_failed", "Unable to complete sign in"))
		return
	}
	raw, ok := token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorBody("missing_id_token", "Identity provider did not return an ID token"))
		return
	}
	idToken, err := m.verifier.Verify(ctx, raw)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("invalid_id_token", "Identity token validation failed"))
		return
	}
	var claims Claims
	if err := idToken.Claims(&claims); err != nil {
		c.JSON(http.StatusUnauthorized, errorBody("invalid_claims", "Identity claims could not be read"))
		return
	}
	if claims.Nonce != nonce {
		c.JSON(http.StatusUnauthorized, errorBody("invalid_nonce", "Identity token replay check failed"))
		return
	}
	if claims.Email == "" || claims.Subject == "" {
		c.JSON(http.StatusUnauthorized, errorBody("missing_claims", "Identity must include subject and email"))
		return
	}
	name := claims.PreferredUsername
	if name == "" {
		name = claims.Name
	}
	if name == "" {
		name = claims.Email
	}
	issuer := idToken.Issuer
	u, err := m.store.EnsureUser(ctx, issuer, claims.Subject, claims.Email, name, claims.Groups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("identity_store_failed", "Unable to store identity"))
		return
	}
	if err := m.sessions.RenewToken(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("session_failed", "Unable to create session"))
		return
	}
	m.sessions.Put(ctx, "user_id", u.ID)
	c.Redirect(http.StatusFound, ret)
}

func htmlError(c *gin.Context, status int, title, message string) {
	c.Data(status, "text/html; charset=utf-8", []byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>`+html.EscapeString(title)+` · Start</title><style>body{font:16px system-ui;display:grid;place-items:center;min-height:100vh;margin:0;background:#09101d;color:#edf4ff}main{max-width:34rem;padding:2rem}a{color:#38bdf8}</style><main><p>Start</p><h1>`+html.EscapeString(title)+`</h1><p>`+html.EscapeString(message)+`</p><a href="/auth/login">Try signing in again</a></main></html>`))
}

func (m *Manager) logout(c *gin.Context) {
	if err := m.sessions.Destroy(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, errorBody("logout_failed", "Unable to end the local session"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"redirectUrl": m.logoutRedirectURL()})
}

func (m *Manager) logoutRedirectURL() string {
	if m.config.DevMode || m.config.OIDCLogoutURL == "" {
		return "/auth/login"
	}

	logoutURL, err := url.Parse(m.config.OIDCLogoutURL)
	if err != nil || logoutURL.Scheme != "https" || logoutURL.Host == "" {
		return "/auth/login"
	}

	query := logoutURL.Query()
	query.Set("login_for", "app")
	query.Set("redirect_uri", strings.TrimSuffix(m.config.BaseURL, "/")+"/")
	logoutURL.RawQuery = query.Encode()
	return logoutURL.String()
}

func safeReturn(v string) string {
	if v == "" || !strings.HasPrefix(v, "/") || strings.HasPrefix(v, "//") {
		return "/"
	}
	return v
}
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func errorBody(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}
