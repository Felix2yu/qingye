package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"qingye/server/config"

	"github.com/gin-gonic/gin"
)

func TestSetup_RegistersRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{CORSOrigins: []string{"*"}}
	Setup(r, cfg)

	routes := r.Routes()
	if len(routes) == 0 {
		t.Fatal("no routes registered")
	}
	registered := map[string]bool{}
	for _, rt := range routes {
		registered[rt.Method+" "+rt.Path] = true
	}
	for _, want := range []string{
		"GET /healthz",
		"GET /api/rooms",
		"POST /api/import/confirm",
		"GET /api/weather/current",
		"POST /api/care-logs",
	} {
		if !registered[want] {
			t.Errorf("route %q not registered", want)
		}
	}
}

func TestCorsMiddleware_Allowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"https://example.com"}))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("allow-origin header missing for allowed origin")
	}
}

func TestCorsMiddleware_Disallowed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"https://example.com"}))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://evil.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin should not receive allow-origin header")
	}
}

func TestCorsMiddleware_AllowAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"*"}))
	r.GET("/x", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("allow-all should return *")
	}
}

func TestCorsMiddleware_Options(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(corsMiddleware([]string{"*"}))
	r.GET("/x", func(c *gin.Context) {})
	req := httptest.NewRequest("OPTIONS", "/x", nil)
	req.Header.Set("Origin", "https://anything.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("OPTIONS should return 204, got %d", w.Code)
	}
}

func TestServeWeb_EmptyDir(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	serveWeb(r, "") // 应为 no-op，不 panic
}

func TestServeWeb_WithDir(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	serveWeb(r, dir)

	// 根路径回退到 index.html
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("GET / code = %d, want 200", w.Code)
	}

	// /api 前缀的未知路径返回 404 JSON
	req2 := httptest.NewRequest("GET", "/api/unknown", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusNotFound {
		t.Errorf("GET /api/unknown code = %d, want 404", w2.Code)
	}
}
