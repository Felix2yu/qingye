package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	if resp.Message != "ok" {
		t.Errorf("message = %q, want ok", resp.Message)
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	BadRequest(c, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 400 {
		t.Errorf("code = %d, want 400", resp.Code)
	}
}

func TestNotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	NotFound(c, "不存在")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestServerError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ServerError(c, "内部错误")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}

func TestParseID_有效(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	c.Params = gin.Params{{Key: "id", Value: "42"}}

	id, ok := parseID(c)
	if !ok || id != 42 {
		t.Errorf("parseID = (%d, %v), want (42, true)", id, ok)
	}
}

func TestParseID_无效(t *testing.T) {
	tests := []struct {
		name, param string
	}{
		{"非数字", "abc"},
		{"零", "0"},
		{"负数", "-1"},
		{"空", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/", nil)
			c.Params = gin.Params{{Key: "id", Value: tt.param}}

			_, ok := parseID(c)
			if ok {
				t.Errorf("parseID(%q) should fail", tt.param)
			}
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400", w.Code)
			}
		})
	}
}
