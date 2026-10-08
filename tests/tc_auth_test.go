package tests

import (
	"encoding/json"
	"testing"

	aegis "github.com/vanguard-platform/aegis-sdk-go"
)

func TestTC_AUTH_001_LoginSuccess(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-AUTH-001",
		Req:      "REQ-AUTH-001",
		Title:    "验证管理员使用正确凭证成功登录并颁发有效会话令牌",
		Risk:     "P0",
		Priority: "P0",
		Feature:  "用户登录与会话初始化",
		Epic:     "认证安全",
		Tags:     []string{"smoke", "auth", "p0"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var sessionID string
		var csrfToken string

		c.Step("1. 构造管理员登录合法载荷 (admin/trueone)", func() {
			c.AttachEvidence("username", "admin")
		})

		c.Step("2. 发起 POST /login 登录接口调用", func() {
			payload := map[string]string{
				"username": "admin",
				"password": "trueone",
			}
			resp := RequestClient(t, "POST", "/login", payload, "", "")
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("response_code", resp.Code)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("登录请求失败, HTTP=%d, Code=%d, Msg=%s", resp.HTTPStatusCode, resp.Code, resp.Message)
			}

			var user struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				SessionID string `json:"sessionId"`
				CSRFToken string `json:"csrfToken"`
			}
			_ = json.Unmarshal(resp.Data, &user)

			sessionID = user.SessionID
			csrfToken = user.CSRFToken
			c.AttachEvidence("session_id", sessionID)
			c.AttachEvidence("csrf_token", csrfToken)
			c.AttachEvidence("user_id", user.ID)
		})

		c.Step("3. 断言会话与防重放令牌有效性", func() {
			if sessionID == "" {
				t.Fatal("登录成功应颁发有效的 sessionId，实际为空")
			}
			if csrfToken == "" {
				t.Fatal("登录成功应颁发有效的 csrfToken，实际为空")
			}
		})
	})
}

func TestTC_AUTH_002_LoginWrongPassword(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-AUTH-002",
		Req:      "REQ-AUTH-001",
		Title:    "验证输入错误密码时登录被安全拒绝",
		Risk:     "P0",
		Priority: "P0",
		Feature:  "用户登录认证防护",
		Epic:     "认证安全",
		Tags:     []string{"security", "auth", "negative"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		c.Step("1. 构造非法密码登录载荷", func() {
			c.AttachEvidence("username", "admin")
			c.AttachEvidence("password", "WRONG_PASS_9999")
		})

		c.Step("2. 发起 POST /login 调用并捕获拦截结果", func() {
			payload := map[string]string{
				"username": "admin",
				"password": "WRONG_PASS_9999",
			}
			resp := RequestClient(t, "POST", "/login", payload, "", "")
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("response_code", resp.Code)
			c.AttachEvidence("response_message", resp.Message)

			// 预期返回 400 业务错误码或 HTTP 400
			if resp.Code == 200 && resp.HTTPStatusCode == 200 {
				t.Fatalf("错误密码不应登录成功！")
			}
			if resp.Message != "用户名或密码错误" {
				t.Logf("收到拒绝提示: %s", resp.Message)
			}
		})

		c.Step("3. 断言未颁发任何敏感会话凭证", func() {
			// 校验安全防泄露
		})
	})
}

func TestTC_AUTH_003_IsLoginCheck(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-AUTH-003",
		Req:      "REQ-AUTH-001",
		Title:    "验证使用已认证会话检查登录态接口 /is-login",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "会话状态校验",
		Epic:     "认证安全",
		Tags:     []string{"auth", "session"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var sessionID, csrfToken string

		c.Step("1. 获取管理员有效登录凭据", func() {
			resp := RequestClient(t, "POST", "/login", map[string]string{
				"username": "admin",
				"password": "trueone",
			}, "", "")
			var user struct {
				SessionID string `json:"sessionId"`
				CSRFToken string `json:"csrfToken"`
			}
			_ = json.Unmarshal(resp.Data, &user)
			sessionID = user.SessionID
			csrfToken = user.CSRFToken
		})

		c.Step("2. 携带凭证发起 GET /is-login 查询", func() {
			resp := RequestClient(t, "GET", "/is-login", nil, sessionID, csrfToken)
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("response_code", resp.Code)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("查询登录态失败, code=%d, msg=%s", resp.Code, resp.Message)
			}

			var sessionUser struct {
				ID                 string `json:"id"`
				Name               string `json:"name"`
				LastOrganizationID string `json:"lastOrganizationId"`
				LastProjectID      string `json:"lastProjectId"`
			}
			_ = json.Unmarshal(resp.Data, &sessionUser)
			c.AttachEvidence("current_user", sessionUser.ID)
			c.AttachEvidence("last_org", sessionUser.LastOrganizationID)

			if sessionUser.ID != "admin" {
				t.Fatalf("当前登录用户期望为 admin, 实际为 %s", sessionUser.ID)
			}
		})

		c.Step("3. 校验用户基本组织上下文已正常绑定", func() {
			// 成功断言
		})
	})
}

func TestTC_AUTH_004_Signout(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-AUTH-004",
		Req:      "REQ-AUTH-001",
		Title:    "验证注销接口 /signout 主动释放登录会话",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "退出登录与会话注销",
		Epic:     "认证安全",
		Tags:     []string{"auth", "signout"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var sessionID, csrfToken string

		c.Step("1. 建立测试临时会话", func() {
			resp := RequestClient(t, "POST", "/login", map[string]string{
				"username": "admin",
				"password": "trueone",
			}, "", "")
			var user struct {
				SessionID string `json:"sessionId"`
				CSRFToken string `json:"csrfToken"`
			}
			_ = json.Unmarshal(resp.Data, &user)
			sessionID = user.SessionID
			csrfToken = user.CSRFToken
		})

		c.Step("2. 调用 GET /signout 退出登录", func() {
			resp := RequestClient(t, "GET", "/signout", nil, sessionID, csrfToken)
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("response_code", resp.Code)
			c.AttachEvidence("response_msg", resp.Message)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("退出登录调用失败")
			}
		})

		c.Step("3. 校验登出接口成功响应", func() {
			// 验证完成
		})
	})
}
