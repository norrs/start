package server

import (
	"database/sql"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/norrs/start/internal/auth"
	"github.com/norrs/start/internal/catalog"
	"github.com/norrs/start/internal/config"
	dbgen "github.com/norrs/start/internal/db"
	"github.com/norrs/start/internal/policy"
	"github.com/norrs/start/internal/seed"
)

type Server struct {
	cfg      config.Config
	db       *sql.DB
	catalog  *catalog.Store
	auth     *auth.Manager
	sessions *scs.SessionManager
	engine   *gin.Engine
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	registry *prometheus.Registry
}

func New(cfg config.Config, db *sql.DB, cat *catalog.Store, authn *auth.Manager, sessions *scs.SessionManager) (*Server, error) {
	if cfg.LogLevel != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}
	s := &Server{cfg: cfg, db: db, catalog: cat, auth: authn, sessions: sessions, engine: r, requests: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "start_http_requests_total", Help: "HTTP requests handled by Start."}, []string{"method", "route", "status"}), duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "start_http_request_duration_seconds", Help: "HTTP request duration.", Buckets: prometheus.DefBuckets}, []string{"method", "route"}), registry: prometheus.NewRegistry()}
	s.registry.MustRegister(s.requests, s.duration, prometheus.NewGoCollector(), prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	r.Use(s.requestContext(), gin.CustomRecovery(s.recover), s.sameOrigin())
	authn.Register(r)
	r.GET("/healthz", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", s.ready)
	if cfg.MetricsEnabled {
		r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})))
	}
	api := r.Group("/api/v1", authn.Require())
	api.GET("/bootstrap", s.bootstrap)
	api.PUT("/me/preferences", s.savePreferences)
	api.PUT("/me/hidden-links/:id", s.hideLink(true))
	api.DELETE("/me/hidden-links/:id", s.hideLink(false))
	api.POST("/links", s.createLink)
	api.PUT("/links/:id", s.updateLink)
	api.DELETE("/links/:id", s.deleteLink)
	api.GET("/links/:id/revisions", s.revisions)
	api.POST("/links/:id/restore/:revision", s.restore)
	api.GET("/link-history", s.linkHistory)
	api.POST("/link-history/restore", s.restoreHistory)
	api.POST("/catalog/import", s.importCatalog)
	r.NoRoute(s.spa)
	return s, nil
}

func (s *Server) Handler() http.Handler { return s.sessions.LoadAndSave(s.engine) }

func (s *Server) recover(c *gin.Context, recovered any) {
	slog.Error("request panic", "request_id", requestID(c), "panic_type", fmt.Sprintf("%T", recovered))
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		fail(c, http.StatusInternalServerError, "internal_error", "Start could not complete the request")
		return
	}
	c.Data(http.StatusInternalServerError, "text/html; charset=utf-8", browserError("Something went wrong", "Start could not complete that request.", requestID(c)))
}

func browserError(title, message, requestID string) []byte {
	requestLine := ""
	if requestID != "" {
		requestLine = `<p><small>Request ID: <code>` + html.EscapeString(requestID) + `</code></small></p>`
	}
	return []byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>` + html.EscapeString(title) + ` · Start</title><style>body{font:16px system-ui;display:grid;place-items:center;min-height:100vh;margin:0;background:#09101d;color:#edf4ff}main{max-width:34rem;padding:2rem}a{color:#38bdf8}code{color:#cbd5e1}</style><main><p>Start</p><h1>` + html.EscapeString(title) + `</h1><p>` + html.EscapeString(message) + `</p>` + requestLine + `<p><a href="/">Return home</a></p></main></html>`)
}

func (s *Server) requestContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		s.requests.WithLabelValues(c.Request.Method, route, status).Inc()
		s.duration.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
		slog.Info("request", "request_id", requestID, "method", c.Request.Method, "route", route, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
	}
}

func (s *Server) sameOrigin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		origin := c.GetHeader("Origin")
		if origin != "" && !s.originAllowed(origin) {
			fail(c, http.StatusForbidden, "bad_origin", "Request origin is not allowed")
			return
		}
		c.Next()
	}
}

func (s *Server) originAllowed(origin string) bool {
	if strings.EqualFold(strings.TrimSuffix(origin, "/"), strings.TrimSuffix(s.cfg.BaseURL, "/")) {
		return true
	}
	if !s.cfg.DevMode {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return (u.Scheme == "http" || u.Scheme == "https") && (host == "localhost" || host == "127.0.0.1" || host == "::1")
}

func (s *Server) ready(c *gin.Context) {
	ctx, cancel := timeContext(c, 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (s *Server) bootstrap(c *gin.Context) {
	id, user, ok := identity(c)
	if !ok {
		return
	}
	out, err := s.catalog.Bootstrap(c.Request.Context(), id, user)
	if err != nil {
		fail(c, 500, "bootstrap_failed", err.Error())
		return
	}
	c.JSON(200, out)
}
func (s *Server) savePreferences(c *gin.Context) {
	_, user, ok := identity(c)
	if !ok {
		return
	}
	var p catalog.Preferences
	if err := c.ShouldBindJSON(&p); err != nil {
		fail(c, 400, "invalid_preferences", err.Error())
		return
	}
	saved, err := s.catalog.SavePreferences(c.Request.Context(), user.ID, p)
	if err != nil {
		fail(c, 400, "invalid_preferences", err.Error())
		return
	}
	c.JSON(200, saved)
}
func (s *Server) hideLink(hide bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, user, ok := identity(c)
		if !ok {
			return
		}
		id, err := parseID(c.Param("id"))
		if err != nil {
			fail(c, 400, "invalid_id", err.Error())
			return
		}
		if err := s.catalog.Hide(c.Request.Context(), user.ID, id, hide); err != nil {
			fail(c, 500, "preference_failed", err.Error())
			return
		}
		c.Status(204)
	}
}
func (s *Server) createLink(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	var input catalog.LinkInput
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, 400, "invalid_link", err.Error())
		return
	}
	linkID, err := s.catalog.CreateLink(c.Request.Context(), id, input, requestID(c))
	if err != nil {
		storeError(c, err)
		return
	}
	c.JSON(201, gin.H{"id": linkID})
}
func (s *Server) updateLink(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	linkID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, "invalid_id", err.Error())
		return
	}
	var input catalog.LinkInput
	if err = c.ShouldBindJSON(&input); err != nil {
		fail(c, 400, "invalid_link", err.Error())
		return
	}
	if err = s.catalog.UpdateLink(c.Request.Context(), id, linkID, input, requestID(c)); err != nil {
		storeError(c, err)
		return
	}
	c.Status(204)
}
func (s *Server) deleteLink(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	linkID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, "invalid_id", err.Error())
		return
	}
	revision, err := strconv.ParseInt(c.Query("revision"), 10, 64)
	if err != nil {
		fail(c, 400, "invalid_revision", "revision query parameter is required")
		return
	}
	if err = s.catalog.DeleteLink(c.Request.Context(), id, linkID, revision, requestID(c)); err != nil {
		storeError(c, err)
		return
	}
	c.Status(204)
}
func (s *Server) revisions(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	linkID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, "invalid_id", err.Error())
		return
	}
	rows, err := s.catalog.Revisions(c.Request.Context(), id, linkID)
	if err != nil {
		storeError(c, err)
		return
	}
	c.JSON(200, rows)
}
func (s *Server) restore(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	linkID, err := parseID(c.Param("id"))
	if err != nil {
		fail(c, 400, "invalid_id", err.Error())
		return
	}
	revision, err := parseID(c.Param("revision"))
	if err != nil {
		fail(c, 400, "invalid_revision", err.Error())
		return
	}
	if err = s.catalog.Restore(c.Request.Context(), id, linkID, revision, requestID(c)); err != nil {
		storeError(c, err)
		return
	}
	c.Status(204)
}

func (s *Server) linkHistory(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	rows, err := s.catalog.LinkHistory(c.Request.Context(), id)
	if err != nil {
		storeError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (s *Server) restoreHistory(c *gin.Context) {
	id, _, ok := identity(c)
	if !ok {
		return
	}
	var input struct {
		Items []catalog.RestoreTarget `json:"items"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		fail(c, http.StatusBadRequest, "invalid_restore", err.Error())
		return
	}
	if err := s.catalog.RestoreMany(c.Request.Context(), id, input.Items, requestID(c)); err != nil {
		storeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

const maximumCatalogImportBytes = 1 << 20

func (s *Server) importCatalog(c *gin.Context) {
	id, user, ok := identity(c)
	if !ok {
		return
	}
	if !id.SystemAdmin {
		fail(c, http.StatusForbidden, "forbidden", "Only system administrators may import catalogs")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maximumCatalogImportBytes)
	contents, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(c, http.StatusRequestEntityTooLarge, "catalog_too_large", "Catalog YAML must be 1 MiB or smaller")
			return
		}
		fail(c, http.StatusBadRequest, "invalid_catalog", "Could not read the catalog YAML")
		return
	}
	if len(contents) == 0 {
		fail(c, http.StatusBadRequest, "invalid_catalog", "Catalog YAML is empty")
		return
	}
	document, err := seed.Parse(contents)
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid_catalog", err.Error())
		return
	}
	result, err := s.catalog.ImportSeedAs(c.Request.Context(), document, requestID(c), user.ID)
	if err != nil {
		storeError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) spa(c *gin.Context) {
	if strings.HasPrefix(c.Request.URL.Path, "/api/") {
		fail(c, 404, "not_found", "The requested API route does not exist")
		return
	}
	requested := strings.TrimPrefix(filepath.Clean("/"+c.Request.URL.Path), "/")
	path := filepath.Join(s.cfg.StaticDir, requested)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		c.File(path)
		return
	}
	if requested != "" {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Page not found · Start</title><style>body{font:16px system-ui;display:grid;place-items:center;min-height:100vh;margin:0;background:#09101d;color:#edf4ff}main{max-width:34rem;padding:2rem}a{color:#38bdf8}code{color:#cbd5e1}</style><main><p>Start</p><h1>Page not found</h1><p><code>`+html.EscapeString(c.Request.URL.Path)+`</code> does not exist.</p><p><a href="/">Return home and search the launchpad</a></p></main></html>`))
		return
	}
	index := filepath.Join(s.cfg.StaticDir, "index.html")
	if _, err := os.Stat(index); err != nil {
		c.Data(404, "text/html; charset=utf-8", []byte(`<!doctype html><title>Start</title><h1>Start</h1><p>Frontend assets are not built. Run <code>bazel build //web:dist</code>.</p>`))
		return
	}
	c.File(index)
}

func identity(c *gin.Context) (policy.Identity, dbgen.User, bool) {
	id, ok := auth.Identity(c)
	if !ok {
		fail(c, 401, "unauthenticated", "Sign in is required")
		return policy.Identity{}, dbgen.User{}, false
	}
	u, ok := auth.User(c)
	if !ok {
		fail(c, 401, "unauthenticated", "Sign in is required")
		return policy.Identity{}, dbgen.User{}, false
	}
	return id, u, true
}
func parseID(v string) (int64, error) {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid identifier")
	}
	return n, nil
}
func requestID(c *gin.Context) string { v, _ := c.Get("request_id"); s, _ := v.(string); return s }
func fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message, "requestId": requestID(c)}})
}
func storeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog.ErrForbidden):
		fail(c, 403, "forbidden", "You are not allowed to perform this action")
	case errors.Is(err, catalog.ErrConflict):
		fail(c, 409, "revision_conflict", "The link changed; reload it and try again")
	case errors.Is(err, sql.ErrNoRows):
		fail(c, 404, "not_found", "The requested link was not found")
	default:
		fail(c, 400, "request_failed", err.Error())
	}
}
