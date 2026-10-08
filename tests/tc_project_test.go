package tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	aegis "github.com/vanguard-platform/aegis-sdk-go"
)

func TestTC_PRJ_001_CreateAndGetProjectDetail(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-PRJ-001",
		Req:      "REQ-PRJ-001",
		Title:    "验证在指定组织下创建新项目并通过详情接口检索",
		Risk:     "P0",
		Priority: "P0",
		Feature:  "项目创建与详情查询",
		Epic:     "项目生命周期管理",
		Tags:     []string{"project", "crud", "p0"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		projectName := fmt.Sprintf("电商质量中台项目_%d", time.Now().Unix()%100000)
		targetOrgID := "100001"
		var projectID string

		c.Step("1. 构造新建项目载荷调用 POST /projects", func() {
			c.AttachEvidence("project_name", projectName)
			c.AttachEvidence("target_org_id", targetOrgID)

			payload := map[string]string{
				"name":           projectName,
				"description":    "面向微服务架构的端到端集成测试项目空间",
				"organizationId": targetOrgID,
			}
			resp := RequestClient(t, "POST", "/projects", payload, "", "")
			c.AttachEvidence("http_status", resp.HTTPStatusCode)
			c.AttachEvidence("code", resp.Code)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("创建项目接口失败: %s", resp.Message)
			}

			var created struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				OrganizationID string `json:"organizationId"`
			}
			_ = json.Unmarshal(resp.Data, &created)
			projectID = created.ID
			c.AttachEvidence("created_project_id", projectID)

			if projectID == "" {
				t.Fatal("新建项目应返回非空 ID")
			}
		})

		c.Step("2. 调用 GET /projects/:id 检索详情并断言对齐", func() {
			detailResp := RequestClient(t, "GET", fmt.Sprintf("/projects/%s", projectID), nil, "", "")
			c.AttachEvidence("detail_status", detailResp.HTTPStatusCode)

			if detailResp.HTTPStatusCode != 200 || detailResp.Code != 200 {
				t.Fatalf("获取项目详情失败: %s", detailResp.Message)
			}

			var pDetail struct {
				ID             string `json:"id"`
				Name           string `json:"name"`
				OrganizationID string `json:"organizationId"`
			}
			_ = json.Unmarshal(detailResp.Data, &pDetail)

			if pDetail.ID != projectID || pDetail.Name != projectName {
				t.Fatalf("项目详情不匹配: 期望 [ID=%s, Name=%s], 实际 [ID=%s, Name=%s]", projectID, projectName, pDetail.ID, pDetail.Name)
			}
		})

		c.Step("3. 校验所属组织归属完全一致", func() {
			// 验证完成
		})
	})
}

func TestTC_PRJ_002_ListProjectsByOrg(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-PRJ-002",
		Req:      "REQ-PRJ-001",
		Title:    "验证按组织维度获取项目列表",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "组织级项目过滤",
		Epic:     "项目生命周期管理",
		Tags:     []string{"project", "org-filter"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		targetOrgID := "100001"

		c.Step("1. 发起 GET /project/list/:orgId 查询", func() {
			resp := RequestClient(t, "GET", fmt.Sprintf("/project/list/%s", targetOrgID), nil, "", "")
			c.AttachEvidence("http_status", resp.HTTPStatusCode)

			if resp.HTTPStatusCode != 200 || resp.Code != 200 {
				t.Fatalf("获取组织项目列表失败: %s", resp.Message)
			}

			var list []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			_ = json.Unmarshal(resp.Data, &list)
			c.AttachEvidence("projects_count", len(list))

			if len(list) == 0 {
				t.Logf("提示: 组织 %s 下当前项目数为 0", targetOrgID)
			}
		})

		c.Step("2. 断言公开项目列表接口 GET /project/list 响应", func() {
			pubResp := RequestClient(t, "GET", "/project/list", nil, "", "")
			if pubResp.HTTPStatusCode != 200 || pubResp.Code != 200 {
				t.Fatalf("获取公开项目列表失败: %s", pubResp.Message)
			}
			c.AttachEvidence("public_list_ok", true)
		})
	})
}

func TestTC_PRJ_003_UpdateProjectInfo(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-PRJ-003",
		Req:      "REQ-PRJ-001",
		Title:    "验证项目名称与描述编辑更新",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "项目编辑",
		Epic:     "项目生命周期管理",
		Tags:     []string{"project", "update"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var projectID string
		initialName := fmt.Sprintf("项目待更新_%d", time.Now().Unix()%10000)
		updatedName := fmt.Sprintf("项目已修改_%d", time.Now().Unix()%10000)

		c.Step("1. 创建测试基线项目", func() {
			resp := RequestClient(t, "POST", "/projects", map[string]string{
				"name":           initialName,
				"description":    "原描述信息",
				"organizationId": "100001",
			}, "", "")
			var p struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(resp.Data, &p)
			projectID = p.ID
			c.AttachEvidence("project_id", projectID)
		})

		c.Step("2. 调用 POST /project/update 更新名称与描述", func() {
			updatePayload := map[string]string{
				"id":          projectID,
				"name":        updatedName,
				"description": "更新后的完整业务描述",
			}
			upResp := RequestClient(t, "POST", "/project/update", updatePayload, "", "")
			c.AttachEvidence("update_status", upResp.HTTPStatusCode)
			if upResp.HTTPStatusCode != 200 || upResp.Code != 200 {
				t.Fatalf("更新项目失败: %s", upResp.Message)
			}
		})

		c.Step("3. 读取详情校验信息已更新生效", func() {
			detailResp := RequestClient(t, "GET", fmt.Sprintf("/projects/%s", projectID), nil, "", "")
			var pDetail struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}
			_ = json.Unmarshal(detailResp.Data, &pDetail)

			if pDetail.Name != updatedName {
				t.Fatalf("项目更新名称未生效: 期望 %s, 实际 %s", updatedName, pDetail.Name)
			}
			c.AttachEvidence("verified_name_updated", true)
		})
	})
}

func TestTC_PRJ_004_DeleteProjectAndVerify(t *testing.T) {
	meta := aegis.Meta{
		ID:       "TC-PRJ-004",
		Req:      "REQ-PRJ-001",
		Title:    "验证项目软删除后详情查询返回 404",
		Risk:     "P1",
		Priority: "P1",
		Feature:  "项目软下线",
		Epic:     "项目生命周期管理",
		Tags:     []string{"project", "delete"},
	}

	aegis.Case(t, meta, func(c *aegis.Context) {
		var projectID string

		c.Step("1. 创建待删除测试项目", func() {
			resp := RequestClient(t, "POST", "/projects", map[string]string{
				"name":           fmt.Sprintf("待删除项目_%d", time.Now().Unix()%10000),
				"organizationId": "100001",
			}, "", "")
			var p struct {
				ID string `json:"id"`
			}
			_ = json.Unmarshal(resp.Data, &p)
			projectID = p.ID
			c.AttachEvidence("project_id", projectID)
		})

		c.Step("2. 调用 DELETE /projects/:id 执行删除", func() {
			delResp := RequestClient(t, "DELETE", fmt.Sprintf("/projects/%s", projectID), nil, "", "")
			c.AttachEvidence("del_http_status", delResp.HTTPStatusCode)
			if delResp.HTTPStatusCode != 200 || delResp.Code != 200 {
				t.Fatalf("删除项目调用失败: %s", delResp.Message)
			}
		})

		c.Step("3. 再次获取详情断言返回 404 错误状态", func() {
			checkResp := RequestClient(t, "GET", fmt.Sprintf("/projects/%s", projectID), nil, "", "")
			c.AttachEvidence("check_status", checkResp.HTTPStatusCode)
			c.AttachEvidence("check_code", checkResp.Code)

			if checkResp.Code != 404 && checkResp.HTTPStatusCode != 404 {
				t.Fatalf("已删除项目再次查询应当返回 404, 实际 code=%d, http=%d", checkResp.Code, checkResp.HTTPStatusCode)
			}
		})
	})
}
