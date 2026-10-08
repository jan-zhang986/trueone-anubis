# TrueOne Anubis 测试架构与差分状态矩阵

> **产出定位**：本文档是 `trueone-anubis` 的**测试架构设计与质量风险全貌**，与
> `diff-matrix.json`（机器可读的四层因果树）互为表里。矩阵已通过平台自身的
> `/api/quality-workspace/:id/diff-matrix/sync` 契约完成实机推送验证。
>
> **基线指纹**：`e0c8a01416a8`（47 个 Go 源文件内容摘要；该目录不是 Git 仓库）
>
> **一句话结论**：**56 个需求章节中 13 个为 GAP（当前构建违反或完全无覆盖），
> 30 个章节存在 P0 风险，17 条用例为红。质量工作台核心链路当前不可用，
> 且平台会签发错误的"可发布"结论。**

---

## 1. 交付物清单

| 文件 | 性质 | 说明 |
| --- | --- | --- |
| `requirements-baseline.md` | 需求基线 | 56 条编号条款，逆向自代码；矩阵 `lineStart/lineEnd` 的唯一锚点 |
| `diff-matrix.json` | **核心资产** | 四层因果树（需求→风险→分析→用例→证据），可直接推送 |
| `TEST-ARCHITECTURE.md` | 本文件 | 人类可读的架构与风险报告 |
| `evidence/strict-audit.log` | 运行时证据 | 真实 `go test` 全量输出（含断言失败原文） |
| `build_matrix.py` | 构建器 | 从基线+测试源码+执行日志+研究产物装配矩阵 |
| `verify_push.py` | 契约验证 | 将矩阵推送到实机 `/diff-matrix/sync` 并回读比对 |
| `raw/*.json` | 研究产物 | 3 个模块的 FMEA 深读结果（供矩阵装配） |
| `internal/**/*_test.go`、`tests/integration/*_test.go` | 测试资产 | 79 个测试函数，可执行、可回归 |

---

## 2. 方法论：四阶段流水线如何落地

本项目的特殊之处在于**没有 PRD**——`trueone-anubis` 的真实规格就是它的实现。
因此把"需求文档"替换为**逆向提炼的契约基线**，其余三阶段严格按规程执行：

```
requirements-baseline.md  56 条编号条款（带真实行号）
   │
   ├── 🔴 失效模式建模 (30×P0 / 23×P1 / 3×P2)
   │      └── 工程影响面：真实文件:行号 + 真实表名
   │             └── 💻 Test-as-Code：Go + httptest + 真实 MySQL
   │                    └── 📊 运行时证据：HTTP/包络/DB 行数/MySQL 错误码
```

**关键工程决策**：

1. **行号真实可验证**。矩阵的 `lineStart/lineEnd` 指向 `requirements-baseline.md`
   的真实行；`analysis.impactScope` 指向真实源码位置与表名。二者可独立复核。
2. **拒绝伪证据**。用例的 `codeSnippet` 不是手写示例，而是通过 AST 式扫描从
   `*_test.go` 中**抽取的真实函数体**，`filePath`/`lineNo` 自动生成；
   `status`/`durationMs` 来自真实执行日志。已校验每个片段花括号平衡。
3. **已知缺陷与回归门禁解耦**。`internal/testsupport.Contract` 提供双模运行：
   - 默认：缺陷记录为 `KNOWN DEVIATION`，套件全绿，可作 CI 回归门禁；
   - `ANUBIS_STRICT=1`：同一批断言转为失败，产出本报告的红色证据。
4. **外部依赖显式豁免**。调用 DeepSeek 的生成路径输出非确定性，默认 `SKIP`
   并由 `ANUBIS_LIVE_LLM=1` 显式开启，绝不伪装成已覆盖。

---

## 3. 差分状态推导规则

产品自身的推导逻辑存在缺陷（见 §5 第 5 条），因此本矩阵采用**显式声明的规则**，
不使用产品当前实现：

| 条件 | diffStatus | 语义 |
| --- | --- | --- |
| 存在任一 `FAILED` | 🔴 **GAP** | 当前构建**违反**该条款，属发布阻断项 |
| 用例数为 0 | 🔴 **GAP** | 该条款无任何用例设计与执行覆盖 |
| 存在任一 `MISSING` | 🟡 **WARNING** | 已有用例设计但尚未落地执行 |
| 全部 `PASSED` | 🟢 **COVERED** | 全部断言通过，具备可复现的运行时证据 |

> 注意：产品把 `GAP` 分支写在 `caseFound` 守卫内，导致"用例数为 0"这一唯一触发条件
> 永远无法到达（死代码）。本矩阵的规则修正了这一点，并在 `diff-matrix.json`
> 中对受影响章节给出 `uncoveredNotice`。

---

## 4. 质量全貌（真实统计）

### 4.1 章节差分状态

| 状态 | 章节数 | 占比 |
| --- | --- | --- |
| 🟢 COVERED | 9 | 16.1% |
| 🟡 WARNING | 34 | 60.7% |
| 🔴 **GAP** | **13** | **23.2%** |
| 合计 | 56 | 100% |

### 4.2 用例与风险分布

| 维度 | 分布 |
| --- | --- |
| 用例总数 | 121 |
| ├ PASSED | 49 |
| ├ **FAILED** | **17** |
| └ MISSING（设计待落地） | 55 |
| 风险等级 | P0 × 30，P1 × 23，P2 × 3 |
| Go 测试函数 | 79 个（78 执行 + 1 显式 SKIP） |
| 契约违规断言 | 24 条（来自 `ANUBIS_STRICT=1` 运行） |
| 路由注册总量 | 458 条（229 根 + 229 `/api`，双前缀完全等价） |

### 4.3 13 个 GAP 章节

`2.2` 会话解析与匿名回落 · `2.4` 密码修改与重置 · `2.5` 项目切换 ·
`3.4` 跨域与凭证 · `5.1` 质量工作台持久化 · `5.2` 质量统计与发布结论 ·
`5.3` 质量任务状态流转 · `5.4` 执行项运行与作用域 · `5.5` 质量报告生成 ·
`6.1` 差分矩阵同步与持久化 · `6.2` 差分状态推导 · `6.4` 未同步工作台与 AI 生成 ·
`7.2` 组织成员与租户隔离

---

## 5. 关键失效模式（按破坏力排序）

### 🔴 P0-1｜构建阻断：源码无法编译（**已修复**）

`internal/service/quality_workspace.go:823` 的 Go 原始字符串以反引号定界，
而其**文本内容**中含 ` ```json `，提前闭合了字符串，导致其后所有代码被误解析。

```
internal/service/quality_workspace.go:823:38: syntax error: unexpected literal `` in argument list
internal/service/quality_workspace.go:871:47: newline in string
internal/service/quality_workspace.go:874:4: syntax error: unexpected keyword else after top level declaration
```

**影响**：`go build ./...` 全量失败，任何测试都无法运行。运行在 8081 的进程是旧二进制。
**处置**：这是本次唯一的**源码改动**，最小化修复为字符串拼接
（`` ` + "```json" + ` ``）。修复后 `go build ./...` 与 `go vet ./...` 均通过。
> 该改动是产出"带断言证据的代码化用例"的前置条件；若不修复，本次交付只能停留在纸面。

### 🔴 P0-2｜空字符串写入 JSON 列导致核心实体无法创建

创建表单不提交 `tags` / `scope_definition` / `metadata` 等 `json` 类型列。
GORM 在 Go 字符串为零值时写入 `''`，而 MySQL 的 json 列**拒绝空字符串**：

```
Error 3140 (22032): Invalid JSON text: "The document is empty."
  at position 0 in value for column 'quality_workspace.tags'
```

**影响面**：同一缺陷以完全相同的方式命中三个写路径——
`POST /quality-workspace/save`（工作台·平台核心实体）、
`POST /quality-workspace/:id/task/save`（质量任务）、
`POST /quality-workspace/:id/report/generate`（质量报告）。
**结论**：质量工作台的创建链路当前**完全不可用**。
**证据**：`tc-ws-01`、`tc-ws-02`、`tc-report-01`（均 FAILED）。

### 🔴 P0-3｜发布门禁失效：3/3 全部失败仍报告"可发布"

`GetStats` 的 `releaseConclusion` 硬编码为 `"READY"`，`allDone` 仅统计
`TODO`/`IN_PROGRESS` 而无视 `FAILED`：

```go
stats := &model.QualityWorkspaceStats{
    ReleaseConclusion: "READY",   // 硬编码，从不参考执行结果
}
stats.AllDone = (stats.Todo == 0 && stats.InProgress == 0)  // FAILED 不计入
```

**实测**：3 条执行项全部 `FAILED` → `releaseConclusion="READY"`、`allDone=true`、`passRate=0.00`。
**为何是 P0**：质量平台最致命的失效模式不是漏报，而是**主动签发错误的放行信号**。
**证据**：`tc-stats-01`（FAILED）。计数与两个比率的算术本身正确（`tc-stats-02` PASSED）。

### 🔴 P0-4｜证据伪造：未同步的工作台会返回并持久化虚构的绿色资产

对从未分析过的工作台执行 `GET .../diff-matrix`，返回**硬编码**矩阵：
仓库指向与本案无关的 `git@github.com:vanguard/trade-payment-service.git`，
内含 5 个章节、**4 条已标记为 `PASSED` 的 Python 用例**，并且被**写回数据库**。

```
observed: repoUrl="git@github.com:vanguard/trade-payment-service.git"
          sections=5 fabricatedPassedCases=4
```

**为何是 P0**：凭空生成的绿色证据比缺失证据危险得多——它会让评审者相信
**从未发生的验证**。`AI 生成`路径同样硬编码该仓库地址。
**证据**：`tc-dm-07`、`tc-dm-09`（SKIP）。

### 🔴 P0-5｜差分状态推导断裂：失败章节可停留在 COVERED，GAP 分支不可达

```go
if caseFound {                                  // 仅当存在用例时才为真
    allPassed, hasMissing := true, false
    for _, tc := range sec.Analysis.Cases {
        if tc.Status == "FAILED" || tc.Status == "MISSING" { allPassed = false }
        if tc.Status == "MISSING" { hasMissing = true }
    }
    if len(sec.Analysis.Cases) == 0 {
        sec.DiffStatus = "GAP"                  // ← 死代码：caseFound 为真蕴含 len>0
    } else if allPassed {
        sec.DiffStatus = "COVERED"
    } else if hasMissing {
        sec.DiffStatus = "WARNING"
    }                                           // ← 仅 FAILED 时三个分支均不命中
}
```

两个缺口：
1. **仅有 FAILED 而无 MISSING 时，`diffStatus` 完全不被赋值**，保留客户端上次提交的值
   → 实测失败的章节仍显示 `COVERED`；
2. **`GAP` 分支不可达**，因为 `caseFound` 为真蕴含 `len(cases) > 0`。

**证据**：`tc-dm-04`、`tc-dm-05`（均 FAILED）；`tc-dm-03`（WARNING 路径正确，PASSED 作对照）。

### 🔴 P0-6｜鉴权绕过三连：全局中间件从不拦截

| 缺陷 | 位置 | 后果 |
| --- | --- | --- |
| `AuthMiddleware` 从不 `Abort` | `middleware/auth.go:38` | 无凭证请求照常进入业务处理函数 |
| 未知令牌被原样写入 `userId` | `middleware/auth.go:53` | 任意调用方自造令牌即可冒充任意身份 |
| `/is-login` 匿名回落 `admin` | `handler/auth.go:36` | 匿名调用方获得完整管理员会话 |

**实测**：匿名 `GET /is-login` → `HTTP 200, code 200, id="admin"`，
权限含 `*` 及 19 项系统权限。伪造令牌 `X-AUTH-TOKEN: forged-token-value`
被接受为 `userId`。
**证据**：`tc-auth-07`、`tc-auth-08`、`tc-auth-10`（均 FAILED）。

### 🔴 P0-7｜账号接管链路：匿名密码重置 + 匿名项目切换

- `ResetPassword` 从**请求体**读取目标 `userId`，**无任何授权校验**。
  实测匿名请求 `code=200`，且数据库中受害者口令的 MD5 **确实被改写**。
- `SwitchProject` 同样信任请求体 `userId`（IDOR），实测匿名改写了
  受害者的 `last_project_id`。

二者组合构成完整的账号接管：先重置口令，再改写项目上下文。
**证据**：`tc-pwd-01`、`tc-switch-01`（均 FAILED）。

### 🔴 P0-8｜跨域凭证泄露：回显任意 Origin 且允许携带凭证

`Cors()` 无条件回显调用方 `Origin`，同时声明 `Allow-Credentials: true`，
且缺失 `Vary: Origin`。任何第三方站点都能以受害者凭证发起跨域请求并读取响应体。

```
observed: Allow-Origin="https://evil.example" Allow-Credentials="true" Vary=""
```
**证据**：`tc-cors-01`（FAILED，内含 3 条违规断言）。

### 🔴 P0-9｜执行项跨工作台写入（IDOR）

`RunWorkItem` 的 `UPDATE` 仅以 `work_item_id` 为条件，URL 中的
`workspace_id` 与 `task_id` 被完全忽略。实测：归属工作台 A 的执行项，
经工作台 B 且携带不存在的 `taskId` 调用后被成功改写为 `FAILED`。
该字段正是统计与发布结论的唯一输入，污染后直接扭曲质量判断。
**证据**：`tc-run-01`（FAILED）。

### 🔴 P0-10｜跨租户数据泄漏

| 端点 | 实测 |
| --- | --- |
| `POST /organization/member/list` | 查询**完全不含** `organizationId` 限定，返回全库用户；实测 A 组织列表中出现 B 组织用户 |
| `POST /operation/log/list` | 无任何租户限定，返回全表 53 条审计轨迹 |
| `POST /organization/log/list` | 作用域取自**客户端请求头**；省略 `ORGANIZATION` 头即返回全表 |
| `POST /project/log/list` | 同理，省略 `PROJECT` 头即返回全表 |
| `GET /system/menu/user-tree` | `GetCurrentUser` 出错时 `isAdmin = true`（**fail-open**）；匿名与伪造令牌均返回完整菜单树，含 `SYSTEM:WRITE` |

**证据**：`tc-tenant-01`（FAILED）；其余由 §6 的设计阶段用例覆盖（MISSING）。

### 🔴 P0-11｜前端调用的 5 个 URL 不存在

以下路径在前端被真实调用，但服务端返回 gin 裸文本 `404 page not found`
（**非** `{code,data,message}` 包络），前端解析必然抛错：

```
/project/robot/enable/{id}                    /project/robot/{id}
/notice/message/task/get/user/{projectId}     /notice/message/template/detail/{projectId}
/notice/template/get/fields/{projectId}       /metrics/requirement-quality/story-search
```

### 🟡 P1 级要点

- **静默空操作**：`CompleteTask` 对不存在的任务返回 `code=200 message="success"`
  （`tc-task-02` FAILED）；`DeleteMenu` 对不存在的菜单返回"删除成功"且 0 条审计。
- **假成功桩**：机器人 `add` 返回**未落库**的虚构 `uuid` 与 `createdAt`，
  随后 `list` 返回 `[]`（写后即丢），且 0 条审计。
- **零值覆盖**：`POST /system/menu/update` 用请求体零值覆盖未提交字段，
  省略 `status` 会把菜单置空并从用户菜单树中消失。
- **`data: null` 而非 `[]`**：`GET /system/menu/tree`、功能用例分页、
  缺陷分页在空结果时返回 `null`，前端 `.map` 抛错。
  `SuccessPage(nil, …)` 的行为已由 `tc-page-03` 固化为可观测契约。
- **时间单位静默错配**：日志查询的 `startTime`/`endTime` 传入秒级时间戳时
  被当作毫秒比较——`startTime` 返回全表、`endTime` 返回空集，且无任何报错。
- **`content` 过滤参数被绑定后弃用**，实测对结果集无影响。
- **审计完整性**：`RecordAuditLog` 在游离 goroutine 中写库且**吞掉 error**，
  `content` 超 500 字符时静默丢弃整条日志；操作者回落到可伪造的 `X-User-Id` 头。

---

## 6. 覆盖边界与生产监控兜底（坦诚披露）

### 6.1 未执行覆盖（34 个 WARNING 章节）

`8.x` 系统设置、`9.x` 组织管理、`10.x` 功能用例/缺陷/项目/环境、
`11.x`–`14.x` 度量/菜单/消息/日志共 **34 个章节仅有用例设计（55 条 MISSING）**，
未落地为可执行测试。已为每章在 `diff-matrix.json` 的
`analysis.uncoveredNotice.onlineMonitoring` 中给出生产兜底观测建议。

**为何未覆盖**：这些模块涉及跨组织/跨项目的多租户状态构造与第三方集成
（飞书、邮件、服务集成），需要专用测试租户与外部 mock，超出本次单轮交付范围。
**这不是可以忽略的理由**——它们中已知含 5 个 P0 失效模式（见 §5 P0-10、P0-11）。

### 6.2 外部依赖豁免

| 依赖 | 处置 | 理由 |
| --- | --- | --- |
| DeepSeek Chat API（`/diff-matrix/generate`） | `SKIP`，`ANUBIS_LIVE_LLM=1` 开启 | 付费外部服务，输出非确定性，无法支撑确定性断言 |

### 6.3 真实未覆盖盲区

- **`is_last` 唯一性**：报告生成恒置 `version_no=1`、`is_latest=true`，
  多份报告会同时为"最新"。当前用例未断言唯一性约束。
- **并发幂等**：菜单 `menu_key` 的"先 count 后 create"非原子；
  权限更新的 count-then-create 同样非原子，双击可产生重复行。
  当前套件为串行执行，**未包含并发风暴压测**。
- **`quality_workspace` 硬删除语义**：模型无 `DeletedAt` 字段，
  `Delete` 为物理删除，与 `archived` 软归档语义并存。
  未断言删除后的数据可恢复性。

---

## 7. 如何复现

```bash
cd /Users/zhangjian/vanguard-platform/trueone-anubis

# DSH 文件沙箱拒绝写共享 Go 构建缓存，故重定向到工作区内
export GOCACHE=$PWD/.gocache GOTMPDIR=$PWD/.gotmp

# 1) 纯单元层（无数据库依赖，秒级）
go test ./internal/response/ ./internal/model/ ./internal/middleware/ ./config/ ./internal/router/ -count=1

# 2) 默认全量：回归门禁，全绿；已知缺陷记录为 KNOWN DEVIATION
ANUBIS_IT=1 go test ./... -count=1

# 3) 严格审计：产出本报告的红色证据（24 条契约违规）
ANUBIS_IT=1 ANUBIS_STRICT=1 go test ./... -count=1 -v | tee docs/test-architecture/evidence/strict-audit.log

# 4) 重新装配矩阵
python3 docs/test-architecture/build_matrix.py

# 5) 实机验证矩阵满足平台同步契约
python3 docs/test-architecture/verify_push.py
```

**集成测试前置**：MySQL `127.0.0.1:3307`，库 `aegis`，账号 `root/root`（见 `config/config.yaml`）。
集成套件全部 fixture 以 `TESTARCH-` 前缀命名并由 `t.Cleanup` 回收；
`ANUBIS_IT` 未置 1 时整体 `SKIP`。

### 实机验证结果

```
artifact      : docs/test-architecture/diff-matrix.json
sections      : 56
  sync        : HTTP 200 code=200 sections=56 lastSyncedAt=1790656840307
  read-back   : HTTP 200 sections=56 all section numbers round-tripped
  diffStatus  : {'COVERED': 9, 'GAP': 13, 'WARNING': 34}
  db metadata : diff_matrix.sections rows = 56
RESULT: PASS - the artifact satisfies the live sync contract
```

---

## 8. 对数据库的影响声明

本次运行对共享 `aegis` 库的写入**已全部回收并核对**：

| 表 | 结果 |
| --- | --- |
| `quality_workspace` / `quality_work_item` / `quality_task` / `quality_report` / `user` | `TESTARCH%` 残留 **0** 行 |
| `operation_log` | 已删除指向 `TESTARCH` 的孤儿审计行；**未删除** admin 的登录/登出/密码重置审计行（追加型日志，删除审计记录本身即为反模式） |

---

## 9. 建议的处置顺序

1. **P0-1（已完成）**：修复编译阻断，否则一切验证无法进行。
2. **P0-2**：为 JSON 类型列补默认值或改用 `*string`/`json.RawMessage`，
   解除工作台/任务/报告三条创建链路的阻断。
3. **P0-6 / P0-7**：`AuthMiddleware` 必须 `Abort`；`/is-login` 去除 admin 回落；
   密码重置与项目切换改为从会话取身份并校验授权。
4. **P0-3 / P0-5**：`releaseConclusion` 与 `diffStatus` 必须由执行结果推导，
   废除硬编码；补全 `GAP` 分支并移出 `caseFound` 守卫。
5. **P0-4**：移除 `buildDefaultDiffMatrix` 的虚构资产与硬编码仓库地址，
   未同步时返回空资产或 404。
6. **P0-8 / P0-10 / P0-11**：CORS 引入白名单与 `Vary`；
   全部租户作用域改由会话与成员关系校验；补齐 6 个前端在用的 URL。
7. **P1**：统一空结果为 `[]`、为状态跃迁与删除补充"行数=0 即失败"判定、
   审计写入改为同步或带重试与错误上报。
8. 将 §6.1 的 34 个 WARNING 章节按 `verificationChecklist` 逐条落地为可执行用例，
   并补齐并发幂等压测。
