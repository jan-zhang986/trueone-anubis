# TrueOne Anubis 接口契约基线 (Requirements Baseline)

> **文档定位**：本文档是 `trueone-anubis` 服务的**需求与契约基线**，作为 Living Test Plan
> 差分矩阵 (`lineStart` / `lineEnd`) 的唯一定位基准。
>
> **来源与溯源**：`trueone-anubis` 仓库不包含原始 PRD，其真实规格即代码实现本身。
> 本文档由 `internal/`、`config/` 与 `config/config.yaml` 逆向提炼而成，**代码是唯一事实来源**。
> 每条约束均可回溯到具体实现位置（见矩阵 `analysis.impactScope`）。
>
> **基线指纹**：`e0c8a01416a8`（47 个 Go 源文件的内容摘要，非 Git 仓库）
>
> **配套证据**：`docs/test-architecture/evidence/strict-audit.log`（真实执行输出）

---

## 1.0 服务启动与配置装配

- `LoadConfig` 读取 `config/config.yaml` 并发布到 `config.GlobalConfig`；文件缺失或 YAML 类型不匹配必须返回错误而非零值配置。
- `InitDB` 在 `GlobalConfig` 为 nil 时必须拒绝执行，不得构造未定义的 DSN。
- 出厂配置固定：端口 `8081`、MySQL `127.0.0.1:3307/aegis`、`parseTime=true`、`token_expire_hours=72`、`default_admin_password=trueone`。
- `BootstrapSystem` 幂等地播种组织 `100001`、项目 `100001100001`、6 个基础角色，并把 `admin` 密码同步为 `MD5(default_admin_password)`。

## 2.1 登录与会话签发

- `POST /login` 接受 `{username, password}`；`username` 同时匹配 `user.id` 与 `user.email`。
- 登录成功返回 HTTP 200，包络 `{code:200, message:"true"}`，`data` 携带 `id`、`sessionId`、`csrfToken`、`lastOrganizationId`、`lastProjectId`、`userRoles`、`userRoleRelations`、`permissions`。
- 签发的 `sessionId` 必须可在服务端会话存储中解析，且映射到已认证用户 id。
- 每次成功登录必须追加且仅追加一条 `type=LOGIN`、`path=/login` 的 `operation_log` 记录。
- 密码错误与账号不存在必须返回**完全相同**的 `code=400` 与 `用户名或密码错误`，不得泄露账号是否存在。
- `enable=0` 的禁用账号必须被拒绝，提示 `账号已被禁用`。
- 缺少 `username` 或 `password` 必须在校验层以 `code=400` 拒绝，不得进入服务层。

## 2.2 会话解析与匿名回落

- `AuthMiddleware` 优先读取 `X-AUTH-TOKEN`，其次解析 `Authorization: Bearer <token>`。
- 令牌命中会话存储时写入 `userId`（归属用户）与 `sessionId`（令牌本身）。
- `ORGANIZATION` 与 `PROJECT` 请求头必须透传到 `orgId` 与 `projectId` 上下文键。
- `GET /is-login` 必须解析已认证用户并返回会话载荷。
- 未携带任何凭证的调用方必须收到 HTTP 401 与包络 `code=100401`。
- 无法识别的令牌不得使调用方通过认证。

## 2.3 登出与会话失效

- `GET /signout` 必须从会话存储移除 `sessionId`，使该令牌失效。
- 对已知会话必须追加一条 `type=LOGOUT` 的 `operation_log` 记录。

## 2.4 密码修改与重置

- `POST /user/update-current-password` 校验 `oldPassword` 与存储凭据或其 MD5 一致，随后写入 `MD5(newPassword)`。
- `POST /system/user/password/reset` 把目标用户密码设为 `newPassword`，缺失时回退 `password`，再回退配置的默认密码。
- 两个端点均必须要求**已认证且已授权**的调用方；目标用户与操作者身份都不得从请求体推断，也不得匿名回落到管理员。

## 2.5 项目切换

- `POST /project/switch` 更新 `user.last_project_id`。
- 目标用户必须取自已认证会话，不得取自请求体 `userId`。

## 3.1 统一响应包络

- `Success`：HTTP 200，`{code:200, message:"success", data:<payload>}`。
- `SuccessWithMsg`：结构同上，`message` 由调用方指定；登录依赖字面量 `"true"`。
- `Fail`：HTTP 200 且 **body.code 非 200**、`data=null`；前端仅依据 `body.code` 判定失败。
- `Unauthorized`：HTTP 401 且 `body.code=100401`，这是唯一允许使用真实 HTTP 错误码的路径。

## 3.2 分页契约

- `SuccessPage` 输出 `{list, total, current, pageSize, totalPages}`，其中 `totalPages = ceil(total/pageSize)`；`pageSize=0` 时 `totalPages=0`（不得除零）。
- `current`、`pageSize`、`totalPages` 为零值时按 `omitempty` 省略。
- `list` 必须是 JSON 数组；nil 切片不得序列化为 `null`。

## 3.3 BIT(1) 布尔列映射

- `BitBool` 编码为 `[]byte{1}` / `[]byte{0}`。
- `Scan` 必须支持 `[]byte{1}`、`[]byte{0}`、`[]byte{'1'}`、`[]byte{'0'}`、空切片、`int64`、`bool`；`NULL` 解码为 `false`；不支持的类型必须返回错误。
- JSON 编码输出布尔值；解码接受 `true`/`false` 与 `1`/`0`。

## 3.4 跨域与凭证

- 预检 `OPTIONS` 返回 204，并声明允许的方法与自定义请求头。
- `Access-Control-Allow-Origin` 必须受可信白名单约束；在 `Access-Control-Allow-Credentials: true` 的前提下，**不得回显任意调用方提供的 Origin**。
- 必须输出 `Vary: Origin`，使共享缓存按请求来源分区。
- `X-AUTH-TOKEN`、`ORGANIZATION`、`PROJECT`、`Authorization`、`CSRF-TOKEN` 必须保持允许。

## 4.0 路由注册与双前缀等价

- 每条路由同时注册在根路径与 `/api` 前缀下，且指向**同一**处理函数。
- 不存在重复的 `method + path` 组合。
- 骨架端点必须返回包络且 `data` 为空数组而非 `null`。
- 未知路径返回 404；以错误动词访问已注册路径不得被静默接受。

## 5.1 质量工作台持久化

- `POST /quality-workspace/save` 在 `workspaceId` 为空时创建记录，生成 UUID 并把 `status` 默认置为 `DRAFT`。
- 创建表单不提交 JSON 类型列（`tags`、`scope_definition`、`metadata`）；持久化层必须把「字段缺省」映射为合法值，而非写入空字符串。
- 更新路径必须确认目标行存在，且仅在实际写入行数大于 0 时报告成功。

## 5.2 质量统计与发布结论

- 每个执行状态必须且仅归入一个计数桶；各桶之和等于 `total`。
- `executionRate = (passed+failed+blocked+skipped)/total`；`passRate = passed/executed`。
- `releaseConclusion` 必须反映真实结果，**存在任何 FAILED 时不得报告 `READY`**。
- 存在 FAILED 时不得报告 `allDone=true`。

## 5.3 质量任务状态流转

- `Complete` 置 `status=COMPLETED`；`Reopen` 置 `status=PENDING`；两者都必须以 `workspace_id` 与 `task_id` 双重限定。
- 未命中任何行的状态跃迁必须报告失败，不得返回成功。

## 5.4 执行项运行与作用域

- `POST /quality-workspace/:id/task/:taskId/work-item/:workItemId/run` 由 `result` 推导 `status`：`FAILED`/`FAIL` → FAILED，`BLOCKED` → BLOCKED，其余 → PASSED。
- 更新必须以 URL 中的 `workspace_id` 与 `task_id` 限定，不得跨工作台写入。

## 5.5 质量报告生成

- `POST /quality-workspace/:id/report/generate` 必须落库 `quality_report`，包含 JSON 统计快照与生成的 Markdown 正文。
- 同一工作台仅允许一份报告被标记为最新（`is_latest=1`）。

## 6.1 差分矩阵同步与持久化

- `POST /quality-workspace/:id/diff-matrix/sync` 把 `repoUrl`、`gitBranch`、`commitSha`、`prdPath` 与 `sections` 持久化到 `quality_workspace.metadata` 的 `diff_matrix` 键下。
- `lastSyncedAt` 必须由服务端生成。
- `GET /quality-workspace/:id/diff-matrix` 必须回读已持久化的资产。
- 对不存在的工作台的同步或读取必须失败，不得凭空构造资产。

## 6.2 差分状态推导

- 全部用例 PASSED 时章节才可为 `COVERED`。
- 存在任一 MISSING 用例的章节必须为 `WARNING`。
- 无任何用例的章节必须为 `GAP`。
- **存在 FAILED 用例的章节绝不可停留在 `COVERED`。**

## 6.3 用例执行上报

- `POST /quality-workspace/:id/diff-matrix/case/report` 更新匹配用例的 `status`、`durationMs` 与 `evidence`。
- 更新后必须重新推导所属章节的 `diffStatus`。

## 6.4 未同步工作台与 AI 生成

- 对从未同步的工作台执行读取，不得虚构指向无关仓库的资产，也不得在未经执行的情况下把任何用例标记为 `PASSED`。
- `POST /quality-workspace/:id/diff-matrix/generate` 对空 Markdown 必须以 `code=400` 拒绝，且不得发起模型调用。
- 生成的资产必须归属工作台的真实仓库，不得硬编码无关仓库。

## 7.1 权限派生

- 管理员必须获得通配权限 `*`。
- 无任何角色关联的用户不得获得通配权限。

## 7.2 组织成员与租户隔离

- `POST /organization/member/list` 必须仅返回所请求组织的成员。
- 任何响应都不得包含 `last_organization_id` 与所请求组织不一致的用户。

## 8.1 系统用户列表与分页

- `GET /system/user` 与 `POST /system/user/page` 必须要求已认证且已授权。
- 响应不得包含 `cft_token`、`lark_open_id`、`lark_union_id` 等敏感字段。
- 用户携带的角色必须按作用域过滤，不得混入其他组织或项目的角色。

## 8.2 用户资料与项目成员

- `GET /user/profile` 必须基于已认证身份返回资料，不得匿名回落到管理员。
- 查询未命中时必须返回 404，不得返回零值用户并报告成功。
- 项目成员列表必须按 `projectId` 过滤，不得返回全库用户。

## 8.3 全局角色权限设置

- `GET /user/role/global/permission/setting/:roleId` 必须基于 `:roleId` 返回该角色的真实权限。
- `POST /user/role/global/permission/update` 必须校验请求体字段与前端契约一致，字段不匹配时不得写出孤立权限行。
- 权限更新必须具备幂等性，不得因重试产生重复行。

## 8.4 全局角色成员关系

- `POST /user/role/relation/global/add` 必须解析请求体并真实落库。
- `GET /user/role/relation/global/delete/:relationId` 必须校验调用方对该关系的操作权限，且必须写入审计日志。

## 8.5 系统组织列表

- `POST /system/organization/page` 返回的分页元数据必须完整，`total=0` 时 `totalPages` 键不得被整体省略。
- 成员数必须为真实统计值，不得硬编码。

## 8.6 系统组织写入

- 组织标识必须由服务端生成且具备唯一性保证，不得使用毫秒时间戳作为唯一键。
- 写入错误必须被检查并向上传播，不得忽略 `Create` 的 error。
- 更新操作不得把未提交的字段覆盖为空值，且必须以 `deleted=0` 限定。

## 8.7 系统项目维护

- `POST /system/project/update` 不得允许把项目迁移到其他组织而不做校验。
- 更新与重命名必须以 `deleted=0` 限定。
- 项目成员与项目管理员列表必须按项目维度过滤，不得返回全库用户或恒定值。

## 8.8 系统参数与第三方集成

- `POST /system/parameter/edit/email-info`、`edit/clean-config`、`POST /lark/save` 必须真实落库，保存后读取必须返回已保存的值。
- `POST /system/parameter/test/email` 不得恒定返回成功。
- 平台信息中的地址不得硬编码为开发机地址。

## 8.9 审计日志写入

- 审计日志的操作者必须取自已认证会话，不得回落到可伪造的 `X-User-Id` 请求头或恒定 `admin`。
- 组织与项目上下文缺失时不得硬编码为固定租户。
- `Create` 的错误不得被静默吞掉；超长 `content` 必须被截断或报错，不得静默丢失整条日志。

## 9.1 组织成员管理

- `POST /organization/member/list` 的 `organizationId` 必须参与查询限定。
- 成员的角色名称必须来自真实关联数据，不得硬编码为 `org_admin`。
- 增删成员接口必须真实落库，不得返回固定 `"ok"` 桩响应。

## 9.2 组织项目维护

- `EnableProject` / `DisableProject` 必须与其他写操作一致地写入审计日志。
- 新增项目必须校验 `organizationId` 非空，不得产生悬空项目。
- 项目查询必须以 `deleted=0` 限定。

## 9.3 组织角色与服务集成

- 空结果集时不得凭空构造不存在的成员记录，也不得篡改 `total`。
- 成员字段命名必须在全局与组织两个变体间保持一致。
- `OrgPermissionSetting` 必须基于 `:roleId` 返回真实权限，`OrgPermissionUpdate` 必须真实落库。
- 服务集成列表的响应形态必须与平台其他列表一致。

## 10.1 功能用例分页与详情

- 分页参数缺省时必须由服务端兜底为 `current=1`、`pageSize=10`，响应必须包含分页字段。
- `list` 必须是数组而非 `null`。
- 详情的绑定错误必须被处理，不得把空 id 查询伪装成 404。

## 10.2 功能用例增删改

- 更新与删除必须以用例 id 加项目维度双重限定。
- 删除请求缺少必要参数时必须返回错误，不得返回 200。
- 新增不得允许调用方注入 `id`、`num`、`deleted` 等受控字段。

## 10.3 用例模块树

- 模块树必须以数组返回，根节点为空时不得返回 `null`。
- 子节点为空时不得通过 `omitempty` 省略 `children` 键。
- 父节点缺失的悬空节点必须有明确归属策略，不得静默提升为根。

## 10.4 用例模块增删与计数

- 删除模块必须级联处理子孙节点，不得产生孤儿节点。
- 删除必须为软删除以保留用例归属。
- 回收站模块计数必须返回真实统计值。

## 10.5 缺陷分页

- 分页结果集必须以数组返回，不得为 `null`。
- 排序必须包含稳定的次级排序键，避免翻页时记录漂移。

## 10.6 缺陷生命周期

- 状态变更必须受状态白名单约束，不得接受任意字符串。
- 缺陷更新与删除必须以项目维度限定，不得跨项目操作。
- 更新未命中任何行时必须报告失败。

## 10.7 缺陷详情与固定桩

- 缺陷详情必须以项目维度限定。
- 当前平台、自定义字段表头、模板选项等接口必须返回真实数据，不得返回与项目无关的固定桩。

## 10.8 项目列表与跨租户泄漏

- `GET /project/list/public` 必须明确其语义，不得因空 `orgId` 而跳过组织限定返回全量项目。
- 项目列表必须尊重调用方的组织上下文。

## 10.9 项目增删改

- 项目写操作必须要求已认证身份，不得匿名回落到 `admin`。
- 删除必须以组织维度限定。

## 10.10 环境管理

- 环境更新必须能够写入 `false` 等零值字段，不得因结构体更新跳过零值而永久无法关闭配置。
- 删除必须为软删除，且必须以项目维度隔离。

## 11.1 效能概览与活动趋势

- `POST/GET /metrics/efficiency/overview` 必须按请求体与查询参数中的项目范围过滤，并返回前端实际读取的字段集合。
- 项目级过滤必须同时作用于全部子指标，不得仅过滤部分指标而其余保持全局统计。
- `GET /metrics/efficiency/activity` 必须返回真实趋势与分解数据，不得返回空壳结构。

## 11.2 需求质量度量

- `/metrics/requirement-quality` 系列必须基于真实数据聚合，不得返回硬编码常量。
- 列表接口必须返回完整分页元数据并提供真实数据。
- 详情接口必须基于 `:storyId` 返回对应需求，不得返回空对象。

## 11.3 未读通知计数

- `GET /notification/un-read/:projectId` 必须按用户与项目维度统计真实未读数。

## 12.1 菜单树构建

- `GET /system/menu/tree` 必须以数组返回；无菜单时不得返回 `null`。
- 父节点缺失的孤儿菜单必须有明确归属策略，不得被静默丢弃。
- 树构建必须具备环检测，不得无限递归。

## 12.2 用户菜单树与权限过滤

- `GET /system/menu/user-tree` 必须要求已认证身份，不得回落到管理员。
- 身份解析失败时不得默认授予管理员可见性。
- 菜单必须按权限码过滤，且权限过滤不得导致已授权的子菜单消失。

## 12.3 菜单增删改

- `POST /system/menu/add` 的 `menu_key` 唯一性校验必须具备原子性。
- `POST /system/menu/update` 不得用请求中的零值覆盖未提交字段。
- 删除不存在的菜单必须返回错误，不得返回成功。
- 菜单的写入与删除必须写入审计日志。

## 13.1 消息任务与模板

- 消息任务列表与保存必须真实落库并以项目维度限定。
- 消息模板字段与详情必须返回真实数据，不得返回空壳。
- 前端调用的消息相关 URL 必须存在并返回标准包络。

## 13.2 机器人管理

- 机器人增删改与启停必须真实落库并以项目维度限定。
- 不得返回未落库的虚构 `id` 与 `createdAt`。
- 写入操作必须写入审计日志。
- 前端调用的机器人 URL 必须存在并返回标准包络。

## 14.1 系统日志与租户隔离

- `POST /operation/log/list` 必须要求已认证且已授权。
- 系统级日志视图不得在无授权情况下向匿名调用方暴露全量审计轨迹。

## 14.2 组织与项目日志作用域

- 组织日志必须以组织上下文限定；上下文缺失时必须拒绝，不得返回全表。
- 项目日志必须以项目上下文限定；上下文缺失时必须拒绝，不得返回全表。
- 租户上下文必须来自已认证会话或经校验的成员关系，不得仅信任客户端请求头。

## 14.3 日志查询参数与时间范围

- `content` 过滤参数必须生效，不得被绑定后弃用。
- 时间范围过滤必须与毫秒时间戳单位一致；无法识别单位时必须报错，不得静默返回全表或空集。
- 绑定失败时的分页兜底行为必须明确且可预期。

## 14.4 日志选项与操作人候选

- 筛选项与操作人候选必须按组织或项目维度限定，不得返回全量用户目录。
- `:organizationId` 与 `:projectId` 路径参数必须生效。
