#!/usr/bin/env python3
"""Assemble the Living Test Plan diff matrix for trueone-anubis.

The matrix is built from four real inputs, never invented:

1. ``requirements-baseline.md``      -> section list with verified line ranges
2. ``evidence/strict-audit.log``     -> real PASS/FAIL/SKIP + duration per test
3. the repository ``*_test.go`` files -> real codeSnippet + filePath + lineNo
4. ``raw/*.json``                    -> design-only FMEA sections for the
                                        modules that are not yet executed

Run:  python3 docs/test-architecture/build_matrix.py
"""

from __future__ import annotations

import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
DOCS = ROOT / "docs" / "test-architecture"
BASELINE = DOCS / "requirements-baseline.md"
AUDIT_LOG = DOCS / "evidence" / "strict-audit.log"
RAW = DOCS / "raw"
OUTPUT = DOCS / "diff-matrix.json"

# Content digest of the analysed tree (the project is not a git checkout).
TREE_DIGEST = "e0c8a01416a8"

# --------------------------------------------------------------------------
# 1. Section registry for the executed half of the baseline (1.0 - 7.2).
#
# Each case is:
#   (case id, go test function name, assertion, dbStateDiff, traceLog)
# filePath / lineNo / codeSnippet / status / durationMs are derived, not typed.
# --------------------------------------------------------------------------

EXECUTED_SECTIONS: dict[str, dict] = {
    "1.0": {
        "risk": "P1",
        "tag": "契约破坏 / 启动阻断",
        "desc": (
            "配置是全部运行时的唯一输入。若 YAML 键名漂移或类型不匹配而未被拒绝，服务会以零值配置启动并连向"
            "错误的数据库；若 InitDB 在 GlobalConfig 为 nil 时仍构造 DSN，故障会推迟到首次查询才暴露。"
        ),
        "impact": ["config/config.go:49 LoadConfig", "config/config.go:64 InitDB",
                   "config/config.yaml", "internal/service/bootstrap.go:20 BootstrapSystem"],
        "strategy": "以出厂 config.yaml、内联完整 YAML、类型错误 YAML 与缺失文件四种输入驱动 LoadConfig，断言字段逐一映射；再清空 GlobalConfig 断言 InitDB 拒绝执行。",
        "checklist": [
            "config.yaml 的 11 个字段全部正确映射",
            "类型不匹配的 YAML 返回错误而非零值",
            "缺失文件返回可读错误",
            "GlobalConfig 为 nil 时 InitDB 返回错误且不返回句柄",
        ],
        "cases": [
            ("tc-cfg-01", "TestLoadConfig_ShippedFile",
             "assert cfg.Server.Port == 8081 and cfg.Database.DBName == \"aegis\" and cfg.Security.DefaultAdminPassword == \"trueone\"",
             "[READ-ONLY] 仅解析本地 YAML，无数据库写入",
             "LoadConfig(config.yaml) -> nil error; port=8081 db=aegis expire=72h"),
            ("tc-cfg-02", "TestLoadConfig_MapsAllKeys",
             "assert cfg.Database.ParseTime == false and cfg.Database.Loc == \"UTC\" and cfg.Auth.TokenExpireHours == 5",
             "[READ-ONLY] 写入临时目录 fixture 后解析",
             "全部 11 个字段映射成功"),
            ("tc-cfg-03", "TestLoadConfig_MalformedYAMLFails",
             "assert err != nil  # server.port 为序列而非整数",
             "[READ-ONLY] 拒绝加载，GlobalConfig 不被覆盖",
             "yaml: cannot unmarshal !!seq into int"),
            ("tc-cfg-04", "TestLoadConfig_MissingFileFails",
             "assert err != nil  # 文件不存在",
             "[READ-ONLY] 不产生任何零值配置",
             "failed to read config file"),
            ("tc-cfg-05", "TestInitDB_RequiresLoadedConfig",
             "assert err != nil and db == nil",
             "[BLOCKED] 未构造 DSN，未建立连接",
             "global config not initialized"),
            ("tc-cfg-06", "TestLoadConfig_PublishesGlobal",
             "assert config.GlobalConfig.Database.DBName == \"aegis\"",
             "[READ-ONLY] 包级全局变量被发布",
             "GlobalConfig published"),
        ],
    },
    "2.1": {
        "risk": "P0",
        "tag": "认证 / 审计完整性",
        "desc": (
            "登录是唯一的凭证入口。密码比对同时接受明文与 MD5，且失败路径必须不可区分以避免账号枚举；"
            "成功路径必须同时产出可解析的服务端会话与审计记录，否则后续所有鉴权与追责都失去依据。"
        ),
        "impact": ["internal/service/auth.go:29 Login", "internal/handler/auth.go:20 Login",
                   "internal/model/user.go:81 LoginRequest", "operation_log 表", "user 表"],
        "strategy": "以正确密码、错误密码、不存在账号、邮箱标识、缺字段五种输入驱动 POST /login，断言包络 code/message、会话存储可解析性，并以行数增量断言 operation_log 恰好新增一条。",
        "checklist": [
            "成功登录 message 为 \"true\" 且 data.sessionId 非空",
            "sessionId 在服务端会话存储中映射到 admin",
            "operation_log 恰好新增一条 type=LOGIN",
            "错误密码与不存在账号响应完全一致",
            "缺字段以 code=400 在校验层拒绝",
        ],
        "cases": [
            ("tc-auth-01", "TestLogin_SuccessIssuesSessionAndWritesAuditLog",
             "assert resp.Message == \"true\" and session.ID == \"admin\" and store.Get(sessionID) == \"admin\" and after == before+1",
             "+ INSERT INTO operation_log (type='LOGIN', path='/login', create_user='admin', project_id='SYSTEM')",
             "POST /login -> HTTP 200 code 200 in 0.01s; permissions include '*'; LOGIN rows 53 -> 54"),
            ("tc-auth-02", "TestLogin_WrongPasswordIsRejected",
             "assert resp.Code == 400 and resp.Message == \"用户名或密码错误\" and operation_log delta == 0",
             "[BLOCKED] 无会话写入，无审计写入",
             "POST /login(wrong) -> code 400, 0 new LOGIN rows"),
            ("tc-auth-03", "TestLogin_UnknownUserIsIndistinguishableFromWrongPassword",
             "assert unknown.Code == wrong.Code and unknown.Message == wrong.Message",
             "[READ-ONLY] 两条失败路径不产生副作用",
             "未知账号与错误密码均返回 code 400 / 用户名或密码错误"),
            ("tc-auth-04", "TestLogin_AcceptsEmailAddress",
             "assert resp.Code == 200 and session.ID == \"admin\"",
             "+ INSERT INTO operation_log (type='LOGIN')  # 邮箱登录同样审计",
             "POST /login(admin@metersphere.io) -> code 200"),
            ("tc-auth-05", "TestLogin_MissingRequiredFieldIsRejected",
             "assert resp.Code == 400  # 绑定层拦截，未进入 AuthService.Login",
             "[BLOCKED] 服务层未执行",
             "POST /login{username only} -> code 400 参数错误"),
        ],
    },
    "2.2": {
        "risk": "P0",
        "tag": "鉴权绕过",
        "desc": (
            "AuthMiddleware 是全站唯一的准入点，却从不 abort：无凭证请求照常进入业务处理函数；"
            "更严重的是无法识别的令牌被原样写入 userId 上下文键，使任意调用方可以自造令牌冒充任意身份；"
            "is-login 更在缺失身份时直接回落到 admin。三者叠加等价于管理员接口完全公开。"
        ),
        "impact": ["internal/middleware/auth.go:38 AuthMiddleware", "internal/middleware/auth.go:53 fallback",
                   "internal/handler/auth.go:36 IsLogin", "internal/service/auth.go:181 GetCurrentUser"],
        "strategy": "以无凭证、伪造令牌、有效会话、Bearer 回退、非 Bearer、双令牌竞争六种请求驱动探针处理器，断言 userId/sessionId 上下文取值；再以匿名请求驱动 /is-login 断言必须 401。",
        "checklist": [
            "有效会话令牌解析为归属用户",
            "无法识别的令牌不得通过认证",
            "无凭证请求不得进入业务处理函数",
            "X-AUTH-TOKEN 优先于 Authorization",
            "匿名 /is-login 返回 401 而非管理员会话",
        ],
        "cases": [
            ("tc-auth-06", "TestAuthMiddleware_KnownSessionResolvesToUser",
             "assert got.UserID == \"u-42\" and got.SessionID == \"session-known-001\"",
             "[READ-ONLY] 会话存储读取",
             "store hit -> userId=u-42"),
            ("tc-auth-07", "TestAuthMiddleware_UnknownTokenMustBeRejected",
             "assert got.UserID == \"\"  # 实际得到 userId=\"forged-token-value\"",
             "[SECURITY] 伪造令牌被提升为可信身份",
             "CONTRACT VIOLATION: unknown token accepted and published as userId"),
            ("tc-auth-08", "TestAuthMiddleware_UncredentialedRequestMustBeRejected",
             "assert !got.ReachedBody  # 实际业务处理函数被执行",
             "[SECURITY] 匿名请求穿透全局中间件",
             "CONTRACT VIOLATION: request with no credentials reached the business handler"),
            ("tc-auth-09", "TestAuthMiddleware_XAuthTokenTakesPrecedenceOverBearer",
             "assert got.UserID == \"u-primary\"",
             "[READ-ONLY] 双令牌竞争时以 X-AUTH-TOKEN 为准",
             "X-AUTH-TOKEN wins over Authorization: Bearer"),
            ("tc-auth-10", "TestIsLogin_WithoutCredentialsMustNotReturnAdminSession",
             "assert resp.HTTP == 401  # 实际 HTTP 200 code 200 id=admin",
             "[SECURITY] 匿名调用方获得 admin 会话与 20 项权限",
             "CONTRACT VIOLATION: anonymous caller received id=admin permissions=[* WORKSPACE:READ ...]"),
            ("tc-auth-11", "TestIsLogin_WithValidTokenReturnsOwningUser",
             "assert resp.Code == 200 and session.ID == \"admin\"",
             "[READ-ONLY] 有效令牌正常解析",
             "GET /is-login(X-AUTH-TOKEN) -> code 200 id=admin"),
            ("tc-auth-12", "TestAuthMiddleware_TenantHeadersArePropagated",
             "assert got.OrgID == \"100001\" and got.ProjectID == \"100001100001\"",
             "[READ-ONLY] 租户上下文透传（未做任何合法性校验）",
             "ORGANIZATION/PROJECT headers -> orgId/projectId context"),
        ],
    },
    "2.3": {
        "risk": "P1",
        "tag": "会话生命周期 / 审计完整性",
        "desc": (
            "登出必须真正使令牌失效，否则被注销的会话仍可用于调用全部接口；同时登出属于安全事件，"
            "必须留痕。由于审计写入发生在游离 goroutine 中且错误被吞掉，登出审计存在静默丢失的可能。"
        ),
        "impact": ["internal/service/auth.go:225 Logout", "internal/handler/auth.go:52 Signout",
                   "internal/middleware/auth.go:32 Store.Delete", "operation_log 表"],
        "strategy": "登录取得令牌，调用 /signout，断言会话存储中已不可解析，并以轮询（2s 上限）等待 LOGOUT 审计行出现，覆盖异步写入时序。",
        "checklist": [
            "登出后会话存储不再命中",
            "2 秒内出现 type=LOGOUT 审计行",
            "登出响应包络为标准成功结构",
        ],
        "cases": [
            ("tc-signout-01", "TestSignout_InvalidatesSession",
             "assert !ok  # store.Get(sessionID) 已失效; 且 waitForAuditLog(LOGOUT, /signout) == true",
             "+ INSERT INTO operation_log (type='LOGOUT', path='/signout', create_user='admin')",
             "GET /signout -> code 200; session evicted; LOGOUT row observed within 2s"),
        ],
    },
    "2.4": {
        "risk": "P0",
        "tag": "鉴权绕过 / 越权",
        "desc": (
            "密码重置从请求体取目标 userId 且完全不做授权判断，匿名调用方即可改写任意账号口令；"
            "修改当前密码在缺失身份时回落到 admin。两者组合构成完整的账号接管链路，是本服务最高危的失效模式。"
        ),
        "impact": ["internal/handler/auth.go:113 ResetPassword", "internal/handler/auth.go:92 UpdateCurrentPassword",
                   "internal/service/auth.go:247 UpdateCurrentPassword", "user 表 password 列"],
        "strategy": "创建一次性受害者账号，分别以带凭证与完全匿名两种请求驱动密码重置，并直接回读 user.password 的 MD5 值断言是否被改写；修改当前密码路径以匿名+错误旧密码断言其被归因为 admin。",
        "checklist": [
            "匿名密码重置必须被拒绝",
            "受害者口令的 MD5 不得被改写",
            "修改当前密码不得匿名回落到 admin",
        ],
        "cases": [
            ("tc-pwd-01", "TestResetPassword_MustRequireAuthorization",
             "assert attack.Code != 200 and !changed  # 实际 code=200 且 password 被改写为 MD5(\"attacker-chosen-password\")",
             "UPDATE user SET password = MD5('attacker-chosen-password') WHERE id = <victim>  [未被授权即执行]",
             "CONTRACT VIOLATION: anonymous POST /system/user/password/reset -> code 200; stored credential rewritten"),
            ("tc-pwd-02", "TestUpdateCurrentPassword_MustNotFallBackToAdmin",
             "assert resp.Code == 400 and resp.Message == \"原密码错误\"  # 证明匿名请求被归因为 admin",
             "[READ-ONLY] 因旧密码校验失败未写入",
             "anonymous POST /user/update-current-password -> code 400 原密码错误（归因 admin）"),
        ],
    },
    "2.5": {
        "risk": "P0",
        "tag": "越权 (IDOR)",
        "desc": (
            "项目切换以请求体中的 userId 作为更新目标，而不是已认证会话身份。任意匿名调用方都能改写"
            "其他用户的活跃项目，破坏其后续全部接口的项目上下文与数据可见范围。"
        ),
        "impact": ["internal/handler/auth.go:73 SwitchProject", "internal/service/auth.go:221 SwitchProject",
                   "user.last_project_id 列"],
        "strategy": "创建受害者账号，以完全匿名的请求携带 victim 的 userId 调用 /project/switch，直接回读 user.last_project_id 断言是否被改写。",
        "checklist": [
            "目标用户必须取自已认证会话",
            "匿名请求不得改写任何用户的 last_project_id",
        ],
        "cases": [
            ("tc-switch-01", "TestSwitchProject_MustNotTrustBodyUserId",
             "assert reloaded.LastProjectID != \"some-other-project\"  # 实际被改写",
             "UPDATE user SET last_project_id='some-other-project' WHERE id=<victim>  [匿名触发]",
             "CONTRACT VIOLATION: unauthenticated /project/switch changed victim's active project"),
        ],
    },
    "3.1": {
        "risk": "P0",
        "tag": "契约破坏",
        "desc": (
            "全部业务失败都以 HTTP 200 + 非 200 body.code 表达，前端错误拦截器只读 body.code。"
            "一旦错误被改写成真实 HTTP 状态码，前端会把失败当成功渲染；登录更依赖 message 字面量 \"true\" 作为成功哨兵。"
        ),
        "impact": ["internal/response/response.go:23 Success", "internal/response/response.go:31 SuccessWithMsg",
                   "internal/response/response.go:57 Fail", "internal/response/response.go:65 Unauthorized"],
        "strategy": "直接驱动各响应构造函数并解析包络，断言 HTTP 状态码、body.code、message 与 data 的精确取值，覆盖成功、自定义哨兵、业务失败与未授权四条路径。",
        "checklist": [
            "Success 为 HTTP 200 / code 200 / message success",
            "SuccessWithMsg 保留自定义哨兵 \"true\"",
            "Fail 为 HTTP 200 且 body.code 非 200、data 为 null",
            "Unauthorized 为 HTTP 401 且 code 100401",
        ],
        "cases": [
            ("tc-resp-01", "TestSuccess_EnvelopeContract",
             "assert w.Code == 200 and got.Code == 200 and got.Message == \"success\" and data == {\"id\":\"100001\"}",
             "[READ-ONLY] 纯序列化，无副作用",
             "HTTP 200 body.code=200 message=success"),
            ("tc-resp-02", "TestSuccessWithMsg_PreservesCustomSentinel",
             "assert got.Message == \"true\"",
             "[READ-ONLY] 登录成功哨兵不被改写",
             "SuccessWithMsg(_, \"true\") -> message=\"true\""),
            ("tc-resp-03", "TestFail_ReturnsHTTP200WithBusinessCode",
             "assert w.Code == 200 and got.Code == 400 and data == null",
             "[READ-ONLY] 业务失败不改变 HTTP 状态码",
             "Fail(400) -> HTTP 200 body.code=400 data=null"),
            ("tc-resp-04", "TestUnauthorized_ReturnsHTTP401AndCode100401",
             "assert w.Code == 401 and got.Code == 100401",
             "[READ-ONLY] 唯一使用真实 HTTP 错误码的路径",
             "Unauthorized -> HTTP 401 body.code=100401"),
        ],
    },
    "3.2": {
        "risk": "P1",
        "tag": "分页一致性 / 契约破坏",
        "desc": (
            "分页元数据驱动前端分页器渲染。totalPages 计算在 pageSize=0 时存在除零风险；"
            "optional 字段的 omitempty 会在零值时整键消失，使前端读到 undefined；"
            "nil 切片被序列化为 null 会让 list.map 抛错。"
        ),
        "impact": ["internal/response/response.go:39 SuccessPage", "internal/response/response.go:15 PageResult"],
        "strategy": "以空集、1 条、整除、余 1、非整除、pageSize=0 六组边界驱动 SuccessPage，逐一断言 totalPages；另断言零值字段的键存在性与 nil 切片的序列化形态。",
        "checklist": [
            "totalPages = ceil(total/pageSize)",
            "pageSize=0 时 totalPages=0 且不发生除零",
            "current/pageSize/totalPages 为零时按键省略",
            "list 与 total 键始终存在",
        ],
        "cases": [
            ("tc-page-01", "TestSuccessPage_TotalPagesMath",
             "assert totalPages == ceil(total/pageSize) for {0,1,10,11,100/7,pageSize=0}",
             "[READ-ONLY] 六组边界，含 pageSize=0 除零防护",
             "totalPages: 0,1,1,2,15,0 全部符合预期"),
            ("tc-page-02", "TestSuccessPage_ZeroPageSizeOmitsOptionalKeys",
             "assert \"current\"/\"pageSize\"/\"totalPages\" absent and \"list\"/\"total\" present",
             "[READ-ONLY] omitempty 行为确认",
             "零值键被省略，list 与 total 保留"),
            ("tc-page-03", "TestSuccessPage_NilListMarshalsToNull",
             "assert list == null  # nil 切片序列化为 null（前端 .map 风险）",
             "[READ-ONLY] 记录当前缺陷行为",
             "list=null for a nil slice"),
        ],
    },
    "3.3": {
        "risk": "P1",
        "tag": "契约破坏 / 数据映射",
        "desc": (
            "user.enable 与 deleted 为 BIT(1) NOT NULL，全部登录与鉴权判等都依赖 BitBool 的正确解码。"
            "MySQL 驱动对 BIT(1) 的返回类型随上下文变化（[]byte 或 int64），解码不全会直接导致登录被误判为禁用账号。"
        ),
        "impact": ["internal/model/types.go:9 BitBool", "internal/model/types.go:18 Scan",
                   "user.enable 列", "user.deleted 列", "quality_workspace.archived 列"],
        "strategy": "以驱动可能返回的全部表示形式驱动 Scan，断言布尔结果与错误行为；再验证 Value 编码、JSON 编解码与整体往返一致性。",
        "checklist": [
            "Value 编码为 []byte{1} / []byte{0}",
            "Scan 覆盖 []byte{1,0,'1','0'}、空切片、int64、bool、nil",
            "不支持类型返回错误",
            "JSON 编码为布尔，解码接受 true/false 与 1/0",
        ],
        "cases": [
            ("tc-bit-01", "TestBitBool_ScanRepresentations",
             "assert Scan([]byte{1})==true, Scan([]byte{'1'})==true, Scan([]byte{'0'})==false, Scan(nil)==false, Scan(int64(1))==true",
             "[READ-ONLY] 10 组表示形式全部正确解码",
             "所有 BIT(1) 驱动表示形式解码正确"),
            ("tc-bit-02", "TestBitBool_ScanRejectsUnsupportedType",
             "assert err != nil  # Scan(\"1\") 不得静默返回 false",
             "[READ-ONLY] 非法类型必须报错",
             "cannot scan string into BitBool"),
            ("tc-bit-03", "TestBitBool_ValueEncoding",
             "assert Value(true) == []byte{1} and Value(false) == []byte{0}",
             "[READ-ONLY] 写入编码正确",
             "driver.Value 编码符合 BIT(1) 期望"),
            ("tc-bit-04", "TestBitBool_JSONRoundTrip",
             "assert decoded == original for true and false",
             "[READ-ONLY] JSON 往返无损",
             "round trip lossless"),
            ("tc-bit-05", "TestBitBool_UnmarshalJSONSilentlyIgnoresMalformedInput",
             "assert err == nil and value unchanged  # 畸形输入被静默接受",
             "[READ-ONLY] 记录当前缺陷行为",
             "malformed JSON accepted without error, target unmodified"),
        ],
    },
    "3.4": {
        "risk": "P0",
        "tag": "风控绕过 / 跨域凭证泄露",
        "desc": (
            "中间件回显任意来源的 Origin 同时声明 Allow-Credentials: true，且无白名单与 Vary: Origin。"
            "任何第三方站点都能以受害者凭证发起跨域请求并读取响应体，等价于关闭同源策略。"
        ),
        "impact": ["internal/middleware/cors.go:7 Cors", "internal/middleware/cors.go:11 Allow-Origin"],
        "strategy": "以攻击者 Origin 驱动带凭证的跨域请求，断言 Allow-Origin 不得回显该来源、不得同时声明凭证放行，并断言 Vary: Origin 存在；另验证预检 204 与自定义头放行。",
        "checklist": [
            "预检 OPTIONS 返回 204",
            "Allow-Origin 不得回显不可信来源",
            "回显来源时不得同时 Allow-Credentials: true",
            "必须输出 Vary: Origin",
            "自定义认证头保持放行",
        ],
        "cases": [
            ("tc-cors-01", "TestCors_UntrustedOriginMustNotBeGrantedAccess",
             "assert allowOrigin != \"https://evil.example\" and vary == \"Origin\"  # 实际回显且 Vary 缺失",
             "[SECURITY] 任意来源获得带凭证的跨域读权限",
             "CONTRACT VIOLATION: Allow-Origin reflects https://evil.example; Allow-Credentials: true; Vary=\"\""),
            ("tc-cors-02", "TestCors_PreflightReturns204",
             "assert w.Code == 204 and Allow-Methods present",
             "[READ-ONLY] 预检短路，不进入业务处理函数",
             "OPTIONS -> 204 with Allow-Methods/Allow-Headers"),
            ("tc-cors-03", "TestCors_AllowsCustomAuthHeaders",
             "assert headers contain x-auth-token, organization, project, authorization, csrf-token",
             "[READ-ONLY] 自定义认证头放行",
             "custom auth headers allowed at preflight"),
            ("tc-cors-04", "TestCors_WildcardWhenOriginAbsent",
             "assert Allow-Origin == \"*\" when no Origin header",
             "[READ-ONLY] 无 Origin 时回落通配",
             "Allow-Origin=* (no Origin header)"),
        ],
    },
    "4.0": {
        "risk": "P0",
        "tag": "契约破坏 / 路由回归",
        "desc": (
            "全部 229 条路由被注册两次（根路径与 /api），前端两套调用方依赖二者完全等价。"
            "任何单侧新增、漏注册或动词错配都会使其中一套调用方 404，而这类缺陷在单接口测试中完全不可见。"
        ),
        "impact": ["internal/router/router.go:30 register", "internal/router/router.go:312 rootGroup",
                   "internal/router/router.go:315 apiGroup"],
        "strategy": "构建路由器并枚举全部路由，逐条断言 /api 孪生路由存在且指向同一处理函数指针；另以真实请求验证双前缀响应字节一致、骨架端点返回空数组、未知路径 404、错误动词不被静默接受。",
        "checklist": [
            "路由总数为 458（229 根 + 229 api）",
            "每条根路由都有同动词的 /api 孪生且处理函数一致",
            "无重复 method+path",
            "双前缀响应字节完全一致",
            "骨架端点 data 为 [] 而非 null",
            "未知路径返回 404",
        ],
        "cases": [
            ("tc-route-01", "TestRouteInventory_DualPrefixParity",
             "assert rootCount == apiCount == 229 and every twin resolves to the same handler pointer",
             "[READ-ONLY] 路由表结构断言",
             "route inventory: total=458 root=229 api=229, 0 parity violations"),
            ("tc-route-02", "TestRouteInventory_BothPrefixesAnswerIdentically",
             "assert root.Body.String() == api.Body.String()",
             "[READ-ONLY] 双前缀响应字节一致",
             "根与 /api 响应体完全相同"),
            ("tc-route-03", "TestStubEndpoints_ReturnEnvelopeWithEmptyArray",
             "assert got.Code == 200 and data == []",
             "[READ-ONLY] 骨架端点返回空数组而非 null",
             "5 条骨架端点 data=[]"),
            ("tc-route-04", "TestStubEndpoint_TestPlanPageEmitsPageEnvelope",
             "assert list == [] and total == 0 and current == 1 and pageSize == 10",
             "[READ-ONLY] 分页骨架端点元数据完整",
             "test-plan/page -> list=[] total=0 current=1 pageSize=10"),
            ("tc-route-05", "TestStubEndpoints_ReturnOkLiteral",
             "assert got.Code == 200 and data == \"ok\"",
             "[READ-ONLY] 字符串骨架端点契约",
             "4 条骨架端点 data=\"ok\""),
            ("tc-route-06", "TestUnknownRouteReturns404",
             "assert w.Code == 404 for /definitely-not-a-route and /api/definitely-not-a-route",
             "[READ-ONLY] 无 catch-all 兜底",
             "unknown path -> 404"),
            ("tc-route-07", "TestMethodMismatchIsRejected",
             "assert GET /login does not return a code-200 envelope",
             "[READ-ONLY] 动词错配不被接受",
             "GET /login -> not a success envelope"),
            ("tc-route-08", "TestPreflightIsHandledGlobally",
             "assert OPTIONS == 204 for /login, /api/login, /functional/case/page",
             "[READ-ONLY] 全局中间件在双前缀下均生效",
             "preflight handled at both prefixes"),
            ("tc-route-09", "TestRouteInventory_PrefixesAreKnown",
             "assert every registered path matches a known prefix allowlist",
             "[READ-ONLY] 无游离根路径注册",
             "all routes fall under known prefixes"),
        ],
    },
    "5.1": {
        "risk": "P0",
        "tag": "契约破坏 / 功能阻断",
        "desc": (
            "创建表单不提交 tags / scope_definition / metadata 三个 json 类型列。GORM 在字段为零值时写入空字符串，"
            "而 MySQL 的 json 列拒绝空字符串，导致插入直接失败并返回 500。质量工作台——本平台的核心实体——"
            "无法通过 API 创建，且同一缺陷以完全相同的方式命中任务与报告创建。"
        ),
        "impact": ["internal/service/quality_workspace.go:78 Save", "internal/handler/quality_workspace.go:46 Save",
                   "internal/model/quality_workspace.go:19 Tags", "quality_workspace.tags 列"],
        "strategy": "以创建表单真实提交的最小载荷驱动 POST /quality-workspace/save，断言 code 200 且落库行数为 1；同一手法验证任务创建端点，确认缺陷跨实体扩散。",
        "checklist": [
            "最小载荷创建工作台返回 code 200",
            "quality_workspace 恰好新增一行",
            "缺省 JSON 列被映射为合法值而非空字符串",
            "任务创建同样不受缺省 JSON 列影响",
        ],
        "cases": [
            ("tc-ws-01", "TestQualityWorkspace_Save_MinimalPayloadCreatesRow",
             "assert resp.Code == 200  # 实际 code=500",
             "[BLOCKED] INSERT 被 MySQL 拒绝，0 行写入",
             "CONTRACT VIOLATION: Error 3140 Invalid JSON text: \"The document is empty.\" for column 'quality_workspace.tags'"),
            ("tc-ws-02", "TestSaveTask_MinimalPayloadCreatesRow",
             "assert resp.Code == 200  # 实际 code=500",
             "[BLOCKED] INSERT INTO quality_task 被拒绝",
             "CONTRACT VIOLATION: Error 3140 for column 'quality_task.metadata'"),
        ],
    },
    "5.2": {
        "risk": "P0",
        "tag": "质量门禁失效 / 误导性结论",
        "desc": (
            "releaseConclusion 与 allDone 被硬编码：结论恒为 READY，allDone 仅统计 TODO/IN_PROGRESS 而不考虑 FAILED。"
            "结果是 3/3 全部失败的工作台依然报告「可发布」，这是质量平台最致命的失效模式——它会主动签发错误的放行信号。"
            "计数与两个比率的算术本身正确，问题集中在结论字段。"
        ),
        "impact": ["internal/service/quality_workspace.go:105 GetStats",
                   "internal/service/quality_workspace.go:112 ReleaseConclusion",
                   "internal/service/quality_workspace.go:145 AllDone", "quality_work_item 表"],
        "strategy": "构造全失败（3/3 FAILED）与混合状态（PASSED/FAILED/BLOCKED/SKIPPED/TODO/RUNNING 各一）两个工作台，驱动 /stats 断言各计数桶、两比率算术，以及结论字段在有失败时不得为 READY / allDone。",
        "checklist": [
            "各状态计数桶互斥且求和等于 total",
            "executionRate = 已执行/总数",
            "passRate = 通过/已执行",
            "存在 FAILED 时 releaseConclusion 不得为 READY",
            "存在 FAILED 时 allDone 不得为 true",
        ],
        "cases": [
            ("tc-stats-01", "TestGetStats_FailedSuiteMustNotReportReleaseReady",
             "assert stats.ReleaseConclusion != \"READY\" and !stats.AllDone  # 实际 READY / allDone=true",
             "[READ-ONLY] 3 条 FAILED 工作项被读取",
             "CONTRACT VIOLATION: total=3 passed=0 failed=3 allDone=true releaseConclusion=\"READY\" passRate=0.00"),
            ("tc-stats-02", "TestGetStats_CountersAndRatesAreDerivedCorrectly",
             "assert passed/failed/blocked/skipped/todo/inProgress == 1 each, executionRate==66.67, passRate==25.00",
             "[READ-ONLY] 六种状态各一条，验证桶映射与比率算术",
             "total=6 passed=1 failed=1 blocked=1 skipped=1 todo=1 inProgress=1 passRate=25.00 executionRate=66.67"),
        ],
    },
    "5.3": {
        "risk": "P1",
        "tag": "幂等 / 静默空操作",
        "desc": (
            "任务状态跃迁以 workspace_id + task_id 双重限定，作用域本身正确；但 GORM 在未命中任何行时返回 nil 错误，"
            "处理函数据此回报成功。前端收到「任务已完成」提示而数据库毫无变化，形成不可观测的静默空操作。"
        ),
        "impact": ["internal/service/quality_workspace.go:174 CompleteTask",
                   "internal/service/quality_workspace.go:178 ReopenTask",
                   "internal/handler/quality_workspace.go:139 CompleteTask", "quality_task 表"],
        "strategy": "先对真实存在的任务调用 complete 并回读 status 验证正常路径；再对不存在的工作台与任务调用同一端点，断言响应不得为成功。",
        "checklist": [
            "正常任务 complete 后 status 为 COMPLETED",
            "对不存在任务调用不得返回成功包络",
        ],
        "cases": [
            ("tc-task-01", "TestCompleteTask_PersistsStatus",
             "assert reloaded.Status == \"COMPLETED\"",
             "UPDATE quality_task SET status='COMPLETED' WHERE task_id=<id> AND workspace_id=<ws>",
             "POST .../task/<id>/complete -> code 200; DB status=COMPLETED"),
            ("tc-task-02", "TestCompleteTask_NonExistentTaskMustNotReportSuccess",
             "assert resp.Code != 200  # 实际 code=200 message=\"success\"",
             "[BLOCKED] 0 行受影响，却报告成功",
             "CONTRACT VIOLATION: non-existent task reported code=200 message=\"success\""),
        ],
    },
    "5.4": {
        "risk": "P0",
        "tag": "越权 (IDOR) / 数据污染",
        "desc": (
            "执行项运行端点的 UPDATE 仅以 work_item_id 为条件，URL 中的 workspace_id 与 task_id 被完全忽略。"
            "任何工作台都能翻转其他工作台的执行结果，且该端点正是执行证据（result/status）的唯一写入方，"
            "污染后会直接扭曲统计与发布结论。"
        ),
        "impact": ["internal/service/quality_workspace.go:219 RunWorkItem",
                   "internal/handler/quality_workspace.go:178 RunWorkItem", "quality_work_item 表"],
        "strategy": "建立归属工作台 A 的执行项，改由工作台 B 且携带不存在 taskId 调用 run，直接回读该执行项 status/result 断言是否被跨工作台改写。",
        "checklist": [
            "更新必须以 workspace_id 与 task_id 双重限定",
            "跨工作台调用不得改写 work_item 状态",
            "result 到 status 的推导映射正确",
        ],
        "cases": [
            ("tc-run-01", "TestRunWorkItem_MustNotWriteOutsideItsWorkspace",
             "assert item.Status != \"FAILED\"  # 实际 status=FAILED，跨工作台写入成功",
             "UPDATE quality_work_item SET status='FAILED', result='FAILED' WHERE work_item_id=<victim>  [由其他工作台触发]",
             "CONTRACT VIOLATION: item owned by ws-A mutated through ws-B; workspace/task not scoped"),
        ],
    },
    "5.5": {
        "risk": "P1",
        "tag": "功能阻断 / 数据完整性",
        "desc": (
            "报告生成写入 quality_report，其 metadata 为 json 列而生成逻辑从不设置该字段，"
            "与工作台、任务创建同源的空字符串问题使报告生成整体返回 500。"
            "此外 version_no 恒为 1、is_latest 恒为 true，多份报告会同时被标记为最新。"
        ),
        "impact": ["internal/service/quality_workspace.go:285 GenerateReport",
                   "internal/service/quality_workspace.go:313 VersionNo", "quality_report 表"],
        "strategy": "在工作台内构造两条 PASSED 执行项后驱动报告生成，断言 code 200、reportId 落库行数为 1、snapshotJson 非空；缺陷路径以 500 与 MySQL 3140 错误为证据。",
        "checklist": [
            "报告生成返回 code 200",
            "quality_report 落库恰好一行",
            "snapshotJson 非空可复现",
            "同一工作台仅一份 is_latest",
        ],
        "cases": [
            ("tc-report-01", "TestGenerateReport_PersistsReportWithSnapshot",
             "assert resp.Code == 200 and persisted == 1 and snapshotJson != \"\"  # 实际 code=500",
             "[BLOCKED] INSERT INTO quality_report 被 MySQL 拒绝",
             "CONTRACT VIOLATION: Error 3140 Invalid JSON text for column 'quality_report.metadata'"),
        ],
    },
    "6.1": {
        "risk": "P0",
        "tag": "契约破坏 / 资产完整性",
        "desc": (
            "差分矩阵是本平台的差异化资产，持久化在 quality_workspace.metadata 的 diff_matrix 键下。"
            "读取路径在 GetDetail 失败时吞掉 not-found 错误并改为返回硬编码默认矩阵，"
            "使得对不存在工作台的读写都「成功」，错误被彻底掩盖。"
        ),
        "impact": ["internal/service/quality_workspace.go:614 SyncDiffMatrix",
                   "internal/service/quality_workspace.go:586 GetDiffMatrix",
                   "internal/service/quality_workspace.go:707 saveMatrixToWorkspace",
                   "quality_workspace.metadata 列"],
        "strategy": "同步一份含单章节单用例的真实矩阵，断言响应回显与服务端 lastSyncedAt；随后直接读取数据库 metadata 断言 diff_matrix 键落库，并经 GET 端点回读验证往返一致。",
        "checklist": [
            "同步响应回显 repoUrl/gitBranch/commitSha/prdPath",
            "lastSyncedAt 由服务端生成且大于 0",
            "metadata.diff_matrix 键落库",
            "GET 回读与同步内容一致",
            "对不存在工作台的读写必须失败",
        ],
        "cases": [
            ("tc-dm-01", "TestSyncDiffMatrix_PersistsAndReadsBack",
             "assert synced.CommitSHA == \"abc1234\" and synced.LastSyncedAt > 0 and stored[\"commitSha\"] == \"abc1234\"",
             "+ UPDATE quality_workspace SET metadata = JSON_SET(metadata,'$.diff_matrix',{...}) WHERE workspace_id=<ws>",
             "POST sync -> code 200; metadata.diff_matrix persisted; GET round-trip matched"),
            ("tc-dm-02", "TestReportCaseExecution_UnknownWorkspaceFails",
             "assert resp.Code != 200  # 实际 code=200，not-found 被吞掉",
             "[SECURITY] 对不存在工作台的上报被接受并写入虚构资产",
             "CONTRACT VIOLATION: report against non-existent workspace returned code=200"),
        ],
    },
    "6.2": {
        "risk": "P0",
        "tag": "状态机时序 / 质量门禁失效",
        "desc": (
            "diffStatus 的推导规则存在两个致命缺口：其一，仅 FAILED 而无 MISSING 时三个分支均不命中，"
            "章节保留客户端上次提交的状态，使失败的章节可以停留在 COVERED；其二，GAP 分支写在仅当存在用例时才为真的"
            "守卫内，因此「无任何用例」这一唯一能触发 GAP 的条件永远无法到达，该分支是死代码。"
        ),
        "impact": ["internal/service/quality_workspace.go:675 caseFound guard",
                   "internal/service/quality_workspace.go:688 GAP branch",
                   "internal/service/quality_workspace.go:676 diffStatus derivation"],
        "strategy": "以三种章节驱动：仅含一个用例并将其上报为 FAILED；含一个 PASSED 与一个 MISSING；不含任何用例。分别断言章节状态必须为 GAP / WARNING / GAP，其中 WARNING 为已正确实现的路径。",
        "checklist": [
            "全部用例 PASSED 才可为 COVERED",
            "存在 MISSING 必须为 WARNING",
            "存在 FAILED 绝不可停留在 COVERED",
            "无任何用例必须为 GAP",
        ],
        "cases": [
            ("tc-dm-03", "TestReportCaseExecution_SectionWithMissingCaseBecomesWarning",
             "assert section.DiffStatus == \"WARNING\"",
             "[READ-ONLY] 正确的推导路径，作为对照组",
             "caseStatus PASSED + 1 MISSING -> diffStatus=WARNING (correct)"),
            ("tc-dm-04", "TestReportCaseExecution_FailedCaseMustNotRemainCovered",
             "assert section.DiffStatus != \"COVERED\"  # 实际仍为 COVERED",
             "[READ-ONLY] 上报 FAILED 后章节状态未被重新推导",
             "CONTRACT VIOLATION: only case FAILED but sectionDiffStatus=\"COVERED\""),
            ("tc-dm-05", "TestReportCaseExecution_SectionWithoutCasesMustBecomeGap",
             "assert section.DiffStatus == \"GAP\"  # 实际为 COVERED",
             "[READ-ONLY] GAP 分支不可达（死代码）",
             "CONTRACT VIOLATION: 0 cases but diffStatus=\"COVERED\"; GAP branch unreachable"),
        ],
    },
    "6.3": {
        "risk": "P1",
        "tag": "证据完整性",
        "desc": (
            "用例执行上报是运行时断言证据（assertion / dbStateDiff / traceLog）的唯一写入口。"
            "证据字段的正确落库决定了差分矩阵是否真实反映执行结果，证据丢失会使矩阵退化为状态标签集合。"
        ),
        "impact": ["internal/service/quality_workspace.go:637 ReportCaseExecution",
                   "internal/model/quality_workspace.go:196 DiffMatrixCaseReportRequest"],
        "strategy": "上报一条 FAILED 用例并携带完整证据三元组与时延，断言 status/durationMs/evidence 三个维度均被持久化，并验证章节状态重推导被触发。",
        "checklist": [
            "case.status 被更新为上报值",
            "durationMs 被持久化",
            "assertion / dbStateDiff / traceLog 三元组全部落库",
        ],
        "cases": [
            ("tc-dm-06", "TestReportCaseExecution_PersistsEvidence",
             "assert tc.Status == \"FAILED\" and tc.DurationMs == 37 and tc.Evidence.Assertion == \"assert verdict.delta == 0.00 [FAIL] got 0.01\" and DbStateDiff and TraceLog round-trip byte-for-byte",
             "assertion/dbStateDiff/traceLog 三元组全量写入 quality_workspace.metadata 的 diff_matrix JSON",
             "case status=FAILED durationMs=37 evidence trio persisted and read back intact"),
        ],
    },
    "6.4": {
        "risk": "P0",
        "tag": "证据伪造 / 契约破坏",
        "desc": (
            "对从未同步的工作台执行 GET，会返回并持久化一份硬编码矩阵：仓库指向与本案无关的 "
            "trade-payment-service，且已包含 4 条被标记为 PASSED 的 Python 用例。"
            "在质量平台中，凭空生成的绿色证据比缺失证据危险得多——它会让评审者相信尚未发生的验证。"
        ),
        "impact": ["internal/service/quality_workspace.go:331 buildDefaultDiffMatrix",
                   "internal/service/quality_workspace.go:610 saveMatrixToWorkspace(defaultMatrix)",
                   "internal/service/quality_workspace.go:900 hardcoded RepoURL"],
        "strategy": "对新建且从未同步的工作台执行 GET，断言不得返回指向无关仓库的资产、不得包含任何 PASSED 用例，并回读数据库断言虚构资产未被持久化；另验证空 Markdown 生成请求以 400 拒绝。",
        "checklist": [
            "未同步工作台不得返回无关仓库资产",
            "不得存在未执行即标记 PASSED 的用例",
            "虚构资产不得写入数据库",
            "空 Markdown 生成请求返回 400",
            "生成资产必须归属真实仓库",
        ],
        "cases": [
            ("tc-dm-07", "TestGetDiffMatrix_MustNotFabricateEvidenceForAnUnsyncedWorkspace",
             "assert repoUrl != \"git@github.com:vanguard/trade-payment-service.git\" and passedCases == 0  # 实际 4 条 PASSED",
             "+ UPDATE quality_workspace SET metadata = <fabricated diff_matrix>  [虚构资产被持久化]",
             "CONTRACT VIOLATION: invented asset for trade-payment-service; 4 cases already marked PASSED; persisted to metadata"),
            ("tc-dm-08", "TestGenerateDiffMatrixFromMarkdown_RejectsEmptyDocument",
             "assert empty.Code == 400",
             "[BLOCKED] 空文档不触发模型调用",
             "POST generate{markdownContent:\"   \"} -> code 400"),
            ("tc-dm-09", "TestGenerateDiffMatrixFromMarkdown_LiveModelCall",
             "assert matrix.RepoURL != \"git@github.com:vanguard/trade-payment-service.git\"",
             "[SKIPPED] 依赖 DeepSeek 外部付费接口，输出非确定性，默认跳过",
             "SKIP: live model call is opt-in via ANUBIS_LIVE_LLM=1"),
        ],
    },
    "7.1": {
        "risk": "P1",
        "tag": "越权 / 权限派生",
        "desc": (
            "权限派生决定前端全部功能可见性。admin 获得通配权限，而角色关系为空的用户以 project_member 兜底后"
            "经 user_role_permission 映射取权。空角色集必须不产生通配权限，否则提权只需清空自身角色关联。"
        ),
        "impact": ["internal/service/auth.go:111 GetUserPermissions", "user_role_relation 表",
                   "user_role_permission 表"],
        "strategy": "分别以 admin 与一个无任何角色关联的用户 id 调用 GetUserPermissions，断言前者含通配、后者不含通配。",
        "checklist": [
            "admin 权限集合包含 \"*\"",
            "无角色关联用户不含 \"*\"",
        ],
        "cases": [
            ("tc-perm-01", "TestGetUserPermissions_AdminHasWildcard",
             "assert contains(perms, \"*\")  # permissions=[* WORKSPACE:READ QUALITY:READ ...] 共 20 项",
             "[READ-ONLY] admin 通配权限确认",
             "admin permissions include '*'"),
            ("tc-perm-02", "TestGetUserPermissions_UnknownUserHasNoWildcard",
             "assert !contains(perms, \"*\")",
             "[READ-ONLY] 空角色集不提权",
             "no wildcard for a user without role relations"),
        ],
    },
    "7.2": {
        "risk": "P0",
        "tag": "越权 / 跨租户数据泄漏",
        "desc": (
            "组织成员列表的查询完全不含 organizationId 限定，直接返回全库未删除用户。"
            "多租户部署下，任一组织的成员即等于平台全部用户目录，构成直接的租户数据泄漏与组织边界失效。"
        ),
        "impact": ["internal/handler/organization.go:44 GetMemberList",
                   "internal/handler/organization.go:60 unscoped query", "user 表"],
        "strategy": "读取两个真实组织，在一个组织下创建归属另一组织的用户，随后请求前者的成员列表，断言响应中不得出现该外组织用户及其 lastOrganizationId。",
        "checklist": [
            "organizationId 必须参与查询限定",
            "响应不得包含外组织用户",
        ],
        "cases": [
            ("tc-tenant-01", "TestOrganizationMemberList_MustBeScopedToOrganization",
             "assert leakedOrg == \"\"  # 实际外组织用户出现在响应中",
             "[SECURITY] SELECT ... FROM user WHERE deleted=0  —— 无 organization_id 限定",
             "CONTRACT VIOLATION: organisation A returned member belonging to organisation B"),
        ],
    },
}

# --------------------------------------------------------------------------
# 2. Baseline sections whose coverage is design-only, mapped onto the FMEA
#    research artifacts produced per module.
# --------------------------------------------------------------------------

DESIGN_ONLY_MAP: dict[str, list[tuple[str, str]]] = {
    "8.1": [("system-organization.json", "sec-system-user-page")],
    "8.2": [("system-organization.json", "sec-user-profile-member")],
    "8.3": [("system-organization.json", "sec-global-permission")],
    "8.4": [("system-organization.json", "sec-global-group-member")],
    "8.5": [("system-organization.json", "sec-system-org-list")],
    "8.6": [("system-organization.json", "sec-system-org-write")],
    "8.7": [("system-organization.json", "sec-system-project")],
    "8.8": [("system-organization.json", "sec-system-parameter")],
    "8.9": [("system-organization.json", "sec-audit-log")],
    "9.1": [("system-organization.json", "sec-org-member")],
    "9.2": [("system-organization.json", "sec-org-project")],
    "9.3": [("system-organization.json", "sec-org-role-service")],
    "10.1": [("case-bug-project.json", "sec-case-page"),
             ("case-bug-project.json", "sec-case-detail")],
    "10.2": [("case-bug-project.json", "sec-case-crud")],
    "10.3": [("case-bug-project.json", "sec-case-module-tree")],
    "10.4": [("case-bug-project.json", "sec-case-module-crud"),
             ("case-bug-project.json", "sec-module-count-and-stubs")],
    "10.5": [("case-bug-project.json", "sec-bug-page")],
    "10.6": [("case-bug-project.json", "sec-bug-lifecycle")],
    "10.7": [("case-bug-project.json", "sec-bug-detail-stubs")],
    "10.8": [("case-bug-project.json", "sec-project-list")],
    "10.9": [("case-bug-project.json", "sec-project-crud")],
    "10.10": [("case-bug-project.json", "sec-environment-crud")],
    "11.1": [("log-menu-message-dashboard.json", "sec-6-1-efficiency-overview")],
    "11.2": [("log-menu-message-dashboard.json", "sec-6-2-requirement-quality")],
    "11.3": [("log-menu-message-dashboard.json", "sec-6-3-notification-unread")],
    "12.1": [("log-menu-message-dashboard.json", "sec-7-1-menu-tree-build")],
    "12.2": [("log-menu-message-dashboard.json", "sec-7-2-user-menu-permission")],
    "12.3": [("log-menu-message-dashboard.json", "sec-7-3-menu-crud")],
    "13.1": [("log-menu-message-dashboard.json", "sec-8-1-message-task-template")],
    "13.2": [("log-menu-message-dashboard.json", "sec-8-2-robot-crud")],
    "14.1": [("log-menu-message-dashboard.json", "sec-9-1-system-log-tenant-scope")],
    "14.2": [("log-menu-message-dashboard.json", "sec-9-2-org-project-log-scope")],
    "14.3": [("log-menu-message-dashboard.json", "sec-9-3-log-query-params")],
    "14.4": [("log-menu-message-dashboard.json", "sec-9-4-log-options-users")],
}

# --------------------------------------------------------------------------
# 3. Inputs
# --------------------------------------------------------------------------

HEADING = re.compile(r"^##\s+(\d+\.\d+)\s+(.*)$")


def parse_baseline() -> list[dict]:
    """Return baseline sections with verified 1-based line ranges."""
    lines = BASELINE.read_text(encoding="utf-8").splitlines()

    headings: list[tuple[int, str, str]] = []
    for idx, line in enumerate(lines):
        m = HEADING.match(line)
        if m:
            headings.append((idx, m.group(1), m.group(2).strip()))

    sections = []
    for pos, (idx, number, title) in enumerate(headings):
        end = headings[pos + 1][0] - 1 if pos + 1 < len(headings) else len(lines) - 1
        # Trim trailing blank lines so lineEnd points at real requirement content.
        while end > idx and not lines[end].strip():
            end -= 1
        sections.append({
            "number": number,
            "title": title,
            "lineStart": idx + 1,
            "lineEnd": end + 1,
            "paragraphs": [
                ln.strip()[2:].strip() if ln.strip().startswith("- ") else ln.strip()
                for ln in lines[idx + 1:end + 1]
                if ln.strip() and not ln.strip().startswith("##")
            ],
        })
    return sections


TEST_RESULT = re.compile(r"^\s*---\s+(PASS|FAIL|SKIP):\s+([A-Za-z0-9_]+)\s+\(([0-9.]+)s\)")


def parse_results() -> dict[str, dict]:
    """Map top-level test function name -> {status, durationMs}."""
    results: dict[str, dict] = {}
    for line in AUDIT_LOG.read_text(encoding="utf-8", errors="replace").splitlines():
        m = TEST_RESULT.match(line)
        if not m:
            continue
        status, name, seconds = m.group(1), m.group(2), float(m.group(3))
        # Keep the worst outcome if a name repeats (subtests are filtered out by
        # the indentation-independent name capture below).
        if name in results and results[name]["status"] == "FAIL":
            continue
        results[name] = {"status": status, "durationMs": int(round(seconds * 1000))}
    return results


FUNC_HEAD = re.compile(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", re.M)


def index_test_functions() -> dict[str, tuple[str, int, str]]:
    """Map test function name -> (relative file path, 1-based line, source)."""
    index: dict[str, tuple[str, int, str]] = {}

    for path in sorted(ROOT.rglob("*_test.go")):
        rel = path.relative_to(ROOT).as_posix()
        if rel.startswith(".gocache") or rel.startswith(".gotmp"):
            continue
        text = path.read_text(encoding="utf-8")
        lines = text.splitlines()
        for match in FUNC_HEAD.finditer(text):
            name = match.group(1)
            line_no = text[: match.start()].count("\n") + 1
            # Capture through the closing brace at column 0.
            body: list[str] = []
            for ln in lines[line_no - 1:]:
                body.append(ln)
                if ln == "}":
                    break
            index[name] = (rel, line_no, "\n".join(body))
    return index


def derive_status(cases: list[dict]) -> tuple[str, str | None]:
    """Map case outcomes onto the section diff status.

    Derivation rule (stated explicitly in TEST-ARCHITECTURE.md):

    * any FAILED          -> GAP      (requirement violated by the current build)
    * no cases at all     -> GAP      (requirement has no test design)
    * any MISSING         -> WARNING  (design exists, execution pending)
    * otherwise           -> COVERED
    """
    if not cases:
        return "GAP", "该条款尚无任何用例设计与执行覆盖。"
    if any(c["status"] == "FAILED" for c in cases):
        return "GAP", "存在 FAILED 用例：当前构建违反该条款，属发布阻断项。"
    if any(c["status"] == "MISSING" for c in cases):
        return "WARNING", "已有用例设计但尚未落地执行，仅具备生产监控兜底。"
    return "COVERED", None


def build() -> dict:
    baseline = parse_baseline()
    results = parse_results()
    sources = index_test_functions()

    by_number = {s["number"]: s for s in baseline}
    known = set(EXECUTED_SECTIONS) | set(DESIGN_ONLY_MAP)
    unknown = known - set(by_number)
    if unknown:
        raise SystemExit(f"registry references baseline sections that do not exist: {sorted(unknown)}")

    raw_cache: dict[str, list[dict]] = {}
    out_sections = []

    for section in baseline:
        number = section["number"]
        cases: list[dict] = []
        analysis: dict
        uncovered = None

        if number in EXECUTED_SECTIONS:
            spec = EXECUTED_SECTIONS[number]
            for case_id, test_name, assertion, db_diff, trace in spec["cases"]:
                if test_name not in sources:
                    raise SystemExit(f"{number}/{case_id}: test {test_name} not found in any *_test.go")
                rel, line_no, code = sources[test_name]
                outcome = results.get(test_name)
                if outcome is None:
                    status, duration = "MISSING", 0
                else:
                    # The Go test tool reports PASS/FAIL/SKIP; the matrix contract
                    # only accepts PASSED/FAILED/MISSING. A skipped test is by
                    # definition not executed, so it maps to MISSING.
                    status = {"PASS": "PASSED", "FAIL": "FAILED", "SKIP": "MISSING"}[outcome["status"]]
                    duration = outcome["durationMs"]
                cases.append({
                    "id": case_id,
                    "name": test_name,
                    "lang": "go",
                    "filePath": rel,
                    "lineNo": line_no,
                    "status": status,
                    "durationMs": duration,
                    "codeSnippet": code,
                    "evidence": {
                        "assertion": assertion,
                        "dbStateDiff": db_diff,
                        "traceLog": trace,
                    },
                })

            analysis = {
                "id": f"ana-{number.replace('.', '-')}",
                "title": f"{section['title']} 失效模式分析",
                "riskLevel": spec["risk"],
                "riskTag": spec["tag"],
                "riskDescription": spec["desc"],
                "impactScope": spec["impact"],
                "strategy": spec["strategy"],
                "verificationChecklist": spec["checklist"],
                "cases": cases,
            }

        elif number in DESIGN_ONLY_MAP:
            merged_analysis = None
            for filename, agent_id in DESIGN_ONLY_MAP[number]:
                path = RAW / filename
                if not path.exists():
                    raise SystemExit(f"missing research artifact: {path}")
                if filename not in raw_cache:
                    raw_cache[filename] = json.loads(path.read_text(encoding="utf-8"))
                found = next((s for s in raw_cache[filename] if s["id"] == agent_id), None)
                if found is None:
                    raise SystemExit(f"{filename}: section {agent_id} not found")

                for case in found["analysis"]["cases"]:
                    case = dict(case)
                    case["id"] = f"{number}-{case['id']}"
                    case["status"] = "MISSING"
                    case["durationMs"] = 0
                    cases.append(case)

                if merged_analysis is None:
                    merged_analysis = dict(found["analysis"])
                else:
                    merged_analysis["verificationChecklist"] = (
                        merged_analysis["verificationChecklist"]
                        + found["analysis"]["verificationChecklist"]
                    )
                    merged_analysis["impactScope"] = (
                        merged_analysis["impactScope"] + found["analysis"]["impactScope"]
                    )
                    merged_analysis["riskDescription"] += " " + found["analysis"]["riskDescription"]

            merged_analysis["id"] = f"ana-{number.replace('.', '-')}"
            analysis = merged_analysis
            uncovered = analysis.pop("uncoveredNotice", None)
            analysis["cases"] = cases

        else:
            raise SystemExit(f"baseline section {number} has no registry entry")

        diff_status, reason = derive_status(cases)
        if reason and uncovered is None:
            uncovered = {
                "reason": reason,
                "onlineMonitoring": (
                    "该条款尚未形成可执行断言，生产环境需以接口错误率、审计写入速率与"
                    "关键表行数变化作为兜底观测指标。"
                ),
            }

        entry = {
            "id": f"sec-{number.replace('.', '-')}",
            "sectionNumber": number,
            "lineStart": section["lineStart"],
            "lineEnd": section["lineEnd"],
            "title": section["title"],
            "paragraphs": section["paragraphs"],
            "diffStatus": diff_status,
            "analysis": analysis,
        }
        if uncovered:
            entry["analysis"]["uncoveredNotice"] = uncovered
        out_sections.append(entry)

    return {
        "workspaceId": "",
        "repoUrl": "file:///Users/zhangjian/vanguard-platform/trueone-anubis",
        "gitBranch": "main (not a git checkout)",
        "commitSha": TREE_DIGEST,
        "prdPath": "docs/test-architecture/requirements-baseline.md",
        "sections": out_sections,
    }


def main() -> None:
    matrix = build()
    OUTPUT.write_text(json.dumps(matrix, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    sections = matrix["sections"]
    counts = {"COVERED": 0, "WARNING": 0, "GAP": 0}
    case_counts = {"PASSED": 0, "FAILED": 0, "MISSING": 0, "SKIP": 0}
    risks = {"P0": 0, "P1": 0, "P2": 0}
    for s in sections:
        counts[s["diffStatus"]] += 1
        risks[s["analysis"]["riskLevel"]] = risks.get(s["analysis"]["riskLevel"], 0) + 1
        for c in s["analysis"]["cases"]:
            case_counts[c["status"]] = case_counts.get(c["status"], 0) + 1

    print(f"wrote {OUTPUT.relative_to(ROOT)}")
    print(f"  sections      : {len(sections)}")
    print(f"  diffStatus    : {counts}")
    print(f"  cases         : {case_counts}")
    print(f"  risk levels   : {risks}")


if __name__ == "__main__":
    main()
