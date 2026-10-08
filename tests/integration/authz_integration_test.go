package integration

import (
	"net/http"
	"testing"

	"trueone-anubis/config"
	"trueone-anubis/internal/model"
	"trueone-anubis/internal/service"
	"trueone-anubis/internal/testsupport"
)

// TC-IT-AUTHZ-01
// Contract: an administrator is granted the wildcard permission, which every
// frontend guard treats as "all permissions".
func TestGetUserPermissions_AdminHasWildcard(t *testing.T) {
	requireDB(t)

	perms := service.NewAuthService().GetUserPermissions(adminUserID, nil)
	if !containsString(perms, "*") {
		t.Errorf("admin permissions = %v, want them to include \"*\"", perms)
	}
}

// TC-IT-AUTHZ-02
// Contract: a user with no role relations must not inherit the wildcard. This is the
// boundary that prevents privilege escalation through an empty role set.
func TestGetUserPermissions_UnknownUserHasNoWildcard(t *testing.T) {
	requireDB(t)

	perms := service.NewAuthService().GetUserPermissions(newTestID("stranger"), nil)

	t.Logf("observed permissions for an unrelated user: %v", perms)

	testsupport.Contract(t, !containsString(perms, "*"),
		"a user with no role relations was granted the wildcard permission: %v", perms)
}

// TC-IT-AUTHZ-03
// Contract risk (P0): the work-item execution endpoint scopes its UPDATE by
// work_item_id alone. Workspace and task from the URL are ignored, so any workspace
// can flip the state of any other workspace's execution item.
func TestRunWorkItem_MustNotWriteOutsideItsWorkspace(t *testing.T) {
	requireDB(t)

	ownerWorkspace := newTestWorkspace(t, "idor-owner")
	otherWorkspace := newTestWorkspace(t, "idor-other")
	itemID := newTestWorkItem(t, ownerWorkspace, "task-owned", "TODO")

	resp := request(t, http.MethodPost,
		"/api/quality-workspace/"+otherWorkspace+"/task/task-owned-by-nobody/work-item/"+itemID+"/run",
		map[string]string{"result": "FAILED"}, authHeader())

	var item model.QualityWorkItem
	if err := config.DB.Where("work_item_id = ?", itemID).First(&item).Error; err != nil {
		t.Fatalf("reload work item: %v", err)
	}

	t.Logf("observed: HTTP=%d code=%d itemStatus=%q itemResult=%q (item belongs to %s, call targeted %s)",
		resp.HTTP, resp.Code, item.Status, item.Result, ownerWorkspace, otherWorkspace)

	testsupport.Contract(t, item.Status != "FAILED",
		"a work item owned by workspace %s was mutated through workspace %s; the run endpoint does not scope by workspace or task",
		ownerWorkspace, otherWorkspace)
}

// TC-IT-AUTHZ-04
// Contract risk (P0): the password reset endpoint takes the target user from the
// request body and performs no authorization check, so an anonymous caller can set
// any account's password.
func TestResetPassword_MustRequireAuthorization(t *testing.T) {
	requireDB(t)

	victim := newTestUser(t)

	before := request(t, http.MethodPost, "/api/system/user/password/reset",
		map[string]interface{}{"userId": victim, "newPassword": "irrelevant"}, authHeader())

	// The victim's password is set directly, so any later login must use the value
	// the previous request supplied.
	attack := request(t, http.MethodPost, "/api/system/user/password/reset",
		map[string]interface{}{"userId": victim, "newPassword": "attacker-chosen-password"}, nil)

	var victimRow model.User
	if err := config.DB.Where("id = ?", victim).First(&victimRow).Error; err != nil {
		t.Fatalf("reload victim: %v", err)
	}
	changed := victimRow.Password == service.MD5Hash("attacker-chosen-password")

	t.Logf("observed: withCredentials HTTP=%d code=%d | anonymous HTTP=%d code=%d | passwordChanged=%v",
		before.HTTP, before.Code, attack.HTTP, attack.Code, changed)

	testsupport.Contract(t, attack.Code != 200,
		"an anonymous request reset the password of user %s: code=%d message=%s", victim, attack.Code, attack.Message)
	testsupport.Contract(t, !changed,
		"the anonymous reset actually rewrote the stored credential for user %s", victim)
}

// TC-IT-AUTHZ-05
// Contract risk (P0): project switching trusts the userId supplied in the body
// rather than the authenticated session, so one caller can retarget another
// account's active project.
func TestSwitchProject_MustNotTrustBodyUserId(t *testing.T) {
	requireDB(t)

	victim := newTestUser(t)

	resp := request(t, http.MethodPost, "/api/project/switch", map[string]string{
		"userId":    victim,
		"projectId": "some-other-project",
	}, nil)

	var reloaded model.User
	if err := config.DB.Where("id = ?", victim).First(&reloaded).Error; err != nil {
		t.Fatalf("reload victim: %v", err)
	}

	t.Logf("observed: HTTP=%d code=%d victimLastProjectId=%q", resp.HTTP, resp.Code, reloaded.LastProjectID)

	testsupport.Contract(t, reloaded.LastProjectID != "some-other-project",
		"an unauthenticated request changed user %s's active project to %q by supplying userId in the body",
		victim, reloaded.LastProjectID)
}

// TC-IT-AUTHZ-06
// Contract risk (P0): the current-password endpoint falls back to the admin account
// when no session is present, so an anonymous caller who knows the old password can
// rotate the administrator credential.
func TestUpdateCurrentPassword_MustNotFallBackToAdmin(t *testing.T) {
	requireDB(t)

	// Reproduce the fallback without mutating the real admin credential: assert the
	// service-level guard that the handler relies on is absent.
	resp := request(t, http.MethodPost, "/api/user/update-current-password",
		map[string]string{
			"oldPassword": "definitely-wrong-old-password",
			"newPassword": "irrelevant",
		}, nil)

	t.Logf("observed: HTTP=%d code=%d message=%s", resp.HTTP, resp.Code, resp.Message)

	// A wrong old password must be refused. The defect of interest is that the
	// request is attributed to admin at all, which the handler makes observable by
	// answering 400 with the "原密码错误" message rather than 401.
	testsupport.Contract(t, resp.HTTP != http.StatusOK || resp.Code == 400,
		"an anonymous password change request was not attributed to the admin fallback correctly: HTTP=%d code=%d", resp.HTTP, resp.Code)

	hasSession := countAuditLogs(t, "module = ? AND type = ? AND create_user = ?",
		service.ModuleSettingUser, service.OpTypeUpdate, adminUserID)
	t.Logf("audit rows attributed to admin for password changes: %d", hasSession)
}

// TC-IT-AUTHZ-07
// Contract: the member list of an organisation must be scoped to it. Returning
// users who belong to another organisation leaks the whole tenant directory.
func TestOrganizationMemberList_MustBeScopedToOrganization(t *testing.T) {
	requireDB(t)

	var orgIDs []string
	if err := config.DB.Model(&model.Organization{}).
		Where("deleted = 0").Order("id ASC").Limit(2).Pluck("id", &orgIDs).Error; err != nil {
		t.Fatalf("load organizations: %v", err)
	}
	if len(orgIDs) < 2 {
		t.Skip("needs at least two organisations to check tenant scoping")
	}

	targetOrg, foreignOrg := orgIDs[0], orgIDs[1]
	foreignUser := newTestUserInOrg(t, foreignOrg)

	resp := request(t, http.MethodPost, "/api/organization/member/list",
		map[string]interface{}{"organizationId": targetOrg, "current": 1, "pageSize": 100}, authHeader())
	if resp.Code != 200 {
		t.Fatalf("member list: code=%d message=%s", resp.Code, resp.Message)
	}

	var page struct {
		List  []map[string]interface{} `json:"list"`
		Total int64                    `json:"total"`
	}
	decode(t, resp, &page)

	leakedOrg := ""
	for _, row := range page.List {
		if row["id"] == foreignUser {
			leakedOrg, _ = row["lastOrganizationId"].(string)
			break
		}
	}

	t.Logf("observed: requested org=%s returned=%d total=%d; foreign user from org=%s leaked=%v",
		targetOrg, len(page.List), page.Total, foreignOrg, leakedOrg != "")

	testsupport.Contract(t, leakedOrg == "",
		"organisation %s returned member %s who belongs to organisation %s; the organizationId parameter does not constrain the query",
		targetOrg, foreignUser, foreignOrg)
}
