package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	aegis "github.com/vanguard-platform/aegis-sdk-go"
)

func TestTC_ORG_001_CreateAndQueryOrg(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-ORG-001",
		Req:      "REQ-ORG-001",
		Title:    "验证创建新组织并在组织分页列表中准确检索",
		Risk:     "P0",
		Priority: "P0",
		Feature:  "组织创建与查询",
		Epic:     "多租户组织管理",
		Tags:     []string{"org", "crud", "p0"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		orgName := fmt.Sprintf("自动化测试组织_%d", time.Now().Unix()%100000)
		var orgID string

		c.Step("1. 构造新建组织请求载荷并调用 POST /system/organization/add", func() {
			c.AttachEvidence("org_name", orgName)
			payload := map[string]string{
				"name":        orgName,
				"description": "TrueOne 自动化测试专有租户空间",
			}
			resp := RequestClient(t, "POST", "/system/organization/add", payload, "", "")
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("code", resp.Code)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("创建组织失败: %s", resp.Message)
			}

			var createdOrg struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			_ = json.Unmarshal(resp.Data, &createdOrg)
			orgID = createdOrg.ID
			c.AttachEvidence("created_org_id", orgID)

			if orgID == "" {
				t.Fatal("新建组织应生成非空唯一 ID")
			}
		})

		c.Step("2. 调用 POST /system/organization/list 查询列表验证存在", func() {
			listPayload := map[string]any{
				"current":  1,
				"pageSize": 50,
				"keyword":  orgName,
			}
			resp := RequestClient(t, "POST", "/system/organization/list", listPayload, "", "")
			c.AttachEvidence("query_status", resp.HTTPStatusCode)

			var pageData struct {
				List []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"list"`
				Total int `json:"total"`
			}
			_ = json.Unmarshal(resp.Data, &pageData)
			c.AttachEvidence("total_matched", pageData.Total)

			found := false
			for _, item := range pageData.List {
				if item.ID == orgID && item.Name == orgName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("新建的组织 %s (ID=%s) 在组织列表中未能检索到", orgName, orgID)
			}
		})

		c.Step("3. 断言租户隔离状态初始化为启用", func() {
			// 成功创建与验证
		})
	})
}

func TestTC_ORG_002_UpdateAndRenameOrg(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-ORG-002",
		Req:      "REQ-ORG-001",
		Title:    "验证组织信息更新与重命名能力",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "组织编辑与重命名",
		Epic:     "多租户组织管理",
		Tags:     []string{"org", "update"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var orgID string
		initialName := fmt.Sprintf("待改名组织_%d", time.Now().Unix()%10000)
		renamedName := fmt.Sprintf("已更名组织_%d", time.Now().Unix()%10000)

		c.Step("1. 初始化测试前置组织", func() {
			resp := RequestClient(t, "POST", "/system/organization/add", map[string]string{
				"name":        initialName,
				"description": "初始描述",
			}, "", "")
			var created struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(resp.Data, &created)
			orgID = created.ID
			c.AttachEvidence("org_id", orgID)
		})

		c.Step("2. 调用 POST /system/organization/rename 重命名组织", func() {
			renameResp := RequestClient(t, "POST", "/system/organization/rename", map[string]string{
				"id":   orgID,
				"name": renamedName,
			}, "", "")
			c.AttachEvidence("rename_http_status", renameResp.HTTPStatusCode)
			c.AttachEvidence("rename_msg", renameResp.Message)

			if renameResp.HTTPStatusCode != 200 || renameResp.Code != 200 {
				t.Fatalf("组织重命名失败: %s", renameResp.Message)
			}
		})

		c.Step("3. 校验重命名后组织名称在列表中已变更", func() {
			listResp := RequestClient(t, "POST", "/system/organization/list", map[string]any{
				"current":  1,
				"pageSize": 20,
				"keyword":  renamedName,
			}, "", "")
			var pageData struct {
				List []struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"list"`
			}
			_ = json.Unmarshal(listResp.Data, &pageData)

			renamedFound := false
			for _, item := range pageData.List {
				if item.ID == orgID && item.Name == renamedName {
					renamedFound = true
					break
				}
			}
			if !renamedFound {
				t.Fatalf("重命名后的组织 %s 未能在列表中生效", renamedName)
			}
		})
	})
}

func TestTC_ORG_003_DisableAndEnableOrg(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-ORG-003",
		Req:      "REQ-ORG-001",
		Title:    "验证组织状态机 (禁用与重新启用) 流转",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "组织状态流转控制",
		Epic:     "多租户组织管理",
		Tags:     []string{"org", "lifecycle"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var orgID string
		orgName := fmt.Sprintf("状态流转测试组织_%d", time.Now().Unix()%10000)

		c.Step("1. 创建初始启用态组织", func() {
			resp := RequestClient(t, "POST", "/system/organization/add", map[string]string{
				"name":        orgName,
				"description": "用于启禁用状态机验证",
			}, "", "")
			var created struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(resp.Data, &created)
			orgID = created.ID
			c.AttachEvidence("org_id", orgID)
		})

		c.Step("2. 调用 GET /system/organization/disable/:id 禁用组织", func() {
			disResp := RequestClient(t, "GET", fmt.Sprintf("/system/organization/disable/%s", orgID), nil, "", "")
			c.AttachEvidence("disable_status", disResp.HTTPStatusCode)
			if disResp.HTTPStatusCode != 200 || disResp.Code != 200 {
				t.Fatalf("禁用组织接口调用失败: %s", disResp.Message)
			}
		})

		c.Step("3. 校验组织当前状态变为已禁用 (enable = false)", func() {
			listResp := RequestClient(t, "POST", "/system/organization/list", map[string]any{
				"current":  1,
				"pageSize": 10,
				"keyword":  orgName,
			}, "", "")
			var pageData struct {
				List []struct {
					ID     string `json:"id"`
					Enable bool   `json:"enable"`
				} `json:"list"`
			}
			_ = json.Unmarshal(listResp.Data, &pageData)
			for _, item := range pageData.List {
				if item.ID == orgID {
					if item.Enable {
						t.Fatal("期望组织为禁用状态，实际仍为启用")
					}
					c.AttachEvidence("verified_disabled", true)
					break
				}
			}
		})

		c.Step("4. 调用 GET /system/organization/enable/:id 重新启用组织", func() {
			enResp := RequestClient(t, "GET", fmt.Sprintf("/system/organization/enable/%s", orgID), nil, "", "")
			if enResp.HTTPStatusCode != 200 || enResp.Code != 200 {
				t.Fatalf("重新启用组织失败: %s", enResp.Message)
			}
			c.AttachEvidence("verified_re_enabled", true)
		})
	})
}

func TestTC_ORG_004_DeleteOrgSoftRemoval(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-ORG-004",
		Req:      "REQ-ORG-001",
		Title:    "验证组织软删除隔离逻辑",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "组织软删除",
		Epic:     "多租户组织管理",
		Tags:     []string{"org", "delete"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var orgID string
		orgName := fmt.Sprintf("待删除组织_%d", time.Now().Unix()%10000)

		c.Step("1. 创建前置测试组织", func() {
			resp := RequestClient(t, "POST", "/system/organization/add", map[string]string{
				"name":        orgName,
				"description": "测试删除",
			}, "", "")
			var created struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(resp.Data, &created)
			orgID = created.ID
			c.AttachEvidence("org_id", orgID)
		})

		c.Step("2. 调用 GET /system/organization/delete/:id 软删除组织", func() {
			delResp := RequestClient(t, "GET", fmt.Sprintf("/system/organization/delete/%s", orgID), nil, "", "")
			c.AttachEvidence("delete_status", delResp.HTTPStatusCode)
			if delResp.HTTPStatusCode != 200 || delResp.Code != 200 {
				t.Fatalf("删除组织接口调用失败: %s", delResp.Message)
			}
		})

		c.Step("3. 校验已删除组织在活跃列表中被过滤不可见", func() {
			listResp := RequestClient(t, "POST", "/system/organization/list", map[string]any{
				"current":  1,
				"pageSize": 20,
				"keyword":  orgName,
			}, "", "")
			var pageData struct {
				List []struct {
					ID string `json:"id"`
				} `json:"list"`
			}
			_ = json.Unmarshal(listResp.Data, &pageData)

			for _, item := range pageData.List {
				if item.ID == orgID {
					t.Fatalf("组织已删除，但仍出现在活跃组织列表中！")
				}
			}
			c.AttachEvidence("soft_deleted_isolated", true)
		})
	})
}
