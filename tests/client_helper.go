package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"trueone-anubis/internal/router"
)

var (
	localBaseURL   = "http://127.0.0.1:8081"
	isLiveServer   bool
	detectLiveOnce sync.Once
	testEngine     *gin.Engine
	engineOnce     sync.Once
)

func checkLiveServer() bool {
	detectLiveOnce.Do(func() {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Get(localBaseURL + "/is-login")
		if err == nil && resp != nil {
			_ = resp.Body.Close()
			isLiveServer = true
		}
	})
	return isLiveServer
}

func getTestEngine() *gin.Engine {
	engineOnce.Do(func() {
		gin.SetMode(gin.TestMode)
		testEngine = router.SetupRouter()
	})
	return testEngine
}

type APIResponse struct {
	HTTPStatusCode int
	Code           int             `json:"code"`
	Message        string          `json:"message"`
	Data           json.RawMessage `json:"data"`
}

// RequestClient 发送测试请求
func RequestClient(t *testing.T, method, path string, body any, sessionID, csrfToken string) *APIResponse {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		reqBody = bytes.NewReader(raw)
	}

	if checkLiveServer() {
		// 发往真实运行的 8081 后端
		url := localBaseURL + path
		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			t.Fatalf("创建 HTTP 请求失败: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if sessionID != "" {
			req.Header.Set("Cookie", fmt.Sprintf("SESSION=%s; CSRF-TOKEN=%s", sessionID, csrfToken))
			req.Header.Set("CSRF-TOKEN", csrfToken)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("执行 HTTP 请求失败: %v", err)
		}
		defer resp.Body.Close()

		respBytes, _ := io.ReadAll(resp.Body)
		var apiResp APIResponse
		apiResp.HTTPStatusCode = resp.StatusCode
		_ = json.Unmarshal(respBytes, &apiResp)
		return &apiResp
	}

	// 回退到内存 Gin Engine
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set("Cookie", fmt.Sprintf("SESSION=%s; CSRF-TOKEN=%s", sessionID, csrfToken))
		req.Header.Set("CSRF-TOKEN", csrfToken)
	}

	w := httptest.NewRecorder()
	getTestEngine().ServeHTTP(w, req)

	var apiResp APIResponse
	apiResp.HTTPStatusCode = w.Code
	_ = json.Unmarshal(w.Body.Bytes(), &apiResp)
	return &apiResp
}
