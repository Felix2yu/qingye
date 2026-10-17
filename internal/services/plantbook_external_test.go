package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// withPlantbookMock 启动一个本地 mock 服务替换 plantbookBaseURL，并返回
// 一个已启用（client_id+secret）的客户端；测试结束自动还原。
func withPlantbookMock(t *testing.T, h http.HandlerFunc) *PlantbookClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	old := plantbookBaseURL
	plantbookBaseURL = srv.URL
	t.Cleanup(func() { plantbookBaseURL = old })
	return NewPlantbookClient("cid", "secret", "")
}

func TestPlantbookClient_Enabled(t *testing.T) {
	if NewPlantbookClient("", "", "").Enabled() {
		t.Error("empty client should be disabled")
	}
	if !NewPlantbookClient("cid", "secret", "").Enabled() {
		t.Error("client with id/secret should be enabled")
	}
	if !NewPlantbookClient("", "", "tk").Enabled() {
		t.Error("client with static token should be enabled")
	}
}

func TestPlantbookClient_StaticToken(t *testing.T) {
	c := NewPlantbookClient("", "", "STATIC-TK")
	tk, err := c.getToken()
	if err != nil {
		t.Fatalf("static token getToken err: %v", err)
	}
	if tk != "STATIC-TK" {
		t.Errorf("static token = %q, want STATIC-TK", tk)
	}
}

func TestPlantbookClient_GetToken_OAuth(t *testing.T) {
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"access_token":"OAUTH-TK","expires_in":3600}`))
			return
		}
		w.WriteHeader(404)
	})
	tk, err := c.getToken()
	if err != nil {
		t.Fatalf("oauth getToken err: %v", err)
	}
	if tk != "OAUTH-TK" {
		t.Errorf("oauth token = %q, want OAUTH-TK", tk)
	}
	// 第二次应命中缓存，不再请求（仍返回同值）
	tk2, _ := c.getToken()
	if tk2 != "OAUTH-TK" {
		t.Errorf("cached token = %q, want OAUTH-TK", tk2)
	}
}

func TestPlantbookClient_Search(t *testing.T) {
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.Write([]byte(`{"access_token":"TK","expires_in":3600}`))
			return
		}
		if r.URL.Path == "/plant/search" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"count": 1,
				"results": []map[string]any{{
					"pid":           "monstera_deliciosa",
					"alias":         "Monstera",
					"common_names":  []map[string]any{{"name": "龟背竹", "language_code": "zh"}},
					"image_url":     "http://x/i.png",
					"link":          "http://x",
				}},
			})
			return
		}
		w.WriteHeader(404)
	})
	cands, err := c.Search("monstera")
	if err != nil {
		t.Fatalf("search err: %v", err)
	}
	if len(cands) != 1 || cands[0].PID != "monstera_deliciosa" || cands[0].Name != "龟背竹" {
		t.Errorf("unexpected candidates: %+v", cands)
	}
}

func TestPlantbookClient_Search_Disabled(t *testing.T) {
	// 未启用客户端直接返回空，不发请求
	c := NewPlantbookClient("", "", "")
	cands, err := c.Search("monstera")
	if err != nil || cands != nil {
		t.Errorf("disabled search should return nil,nil; got %v,%v", cands, err)
	}
}

func TestPlantbookClient_Search_Throttled(t *testing.T) {
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.Write([]byte(`{"access_token":"TK","expires_in":3600}`))
			return
		}
		if r.URL.Path == "/plant/search" {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(429)
			_, _ = w.Write([]byte("rate limited"))
			return
		}
		w.WriteHeader(404)
	})
	_, err := c.Search("monstera")
	if err == nil {
		t.Fatal("expected throttled error")
	}
	if !IsThrottled(err) {
		t.Errorf("error %v should be throttled", err)
	}
}

func TestPlantbookClient_Detail(t *testing.T) {
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.Write([]byte(`{"access_token":"TK","expires_in":3600}`))
			return
		}
		if r.URL.Path == "/plant/detail/monstera_deliciosa/" {
			detail := map[string]any{
				"pid":          "monstera_deliciosa",
				"display_pid":  "Monstera deliciosa",
				"alias":        "Monstera",
				"image_url":    "http://x/i.png",
				"max_temp":     30,
				"min_temp":     15,
				"light":        "明亮散射光",
				"watering":     "保持湿润",
				"soil":         "腐殖土",
				"fertilization": "每月一次",
				"pruning":      "修剪枯叶",
				"care": map[string]any{
					"light":         "明亮",
					"watering":      "多浇水",
					"soil":          "泥炭土",
					"fertilization":  "春夏季",
					"pruning":       "随时",
				},
			}
			_ = json.NewEncoder(w).Encode(detail)
			return
		}
		w.WriteHeader(404)
	})
	lib, err := c.Detail("monstera_deliciosa")
	if err != nil {
		t.Fatalf("detail err: %v", err)
	}
	if lib == nil || lib.PID != "monstera_deliciosa" {
		t.Fatalf("unexpected library: %+v", lib)
	}
	if lib.Name == "" {
		t.Errorf("name should not be empty, got %q", lib.Name)
	}
	if lib.Metrics == "" {
		t.Errorf("metrics should be populated from detail thresholds")
	}
}

func TestPlantbookClient_Detail_InvalidJSON(t *testing.T) {
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.Write([]byte(`{"access_token":"TK","expires_in":3600}`))
			return
		}
		w.Write([]byte("not-json"))
	})
	if _, err := c.Detail("x"); err == nil {
		t.Error("expected json parse error")
	}
}

func TestPlantbookClient_Detail_UnauthorizedRetry(t *testing.T) {
	var getCount int32
	c := withPlantbookMock(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token/" {
			w.Write([]byte(`{"access_token":"TK","expires_in":3600}`))
			return
		}
		// 第一次 GET 返回 401 触发凭据失效并重试；第二次 200
		if atomic.AddInt32(&getCount, 1) == 1 {
			w.WriteHeader(401)
			_, _ = w.Write([]byte("unauthorized"))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pid":       "x",
			"alias":     "X",
			"max_temp":  25,
			"min_temp":  10,
		})
	})
	lib, err := c.Detail("x")
	if err != nil {
		t.Fatalf("retry should succeed: %v", err)
	}
	if lib == nil || lib.PID != "x" {
		t.Errorf("unexpected library after retry: %+v", lib)
	}
	if atomic.LoadInt32(&getCount) != 2 {
		t.Errorf("expected 2 GET attempts, got %d", getCount)
	}
}
