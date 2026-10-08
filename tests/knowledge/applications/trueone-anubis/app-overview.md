# TrueOne Anubis 系统架构与核心业务边界

## 1. 核心定位
`trueone-anubis` 是 TrueOne 企业级质量平台的 Go 原生核心后端服务。它统一承载了测试用例管理、质量工作空间、工作流 DAG 执行编排、组织与多项目权限体系，为平台前端（`trueone-web`）、CLI 工具（`trueone-cli`）和多语言 SDK（`trueone-sdk`）提供坚固的底层支撑。

---

## 2. 三大核心业务支柱与领域状态机

### 2.1 认证与会话域 (Auth & Session)
- **凭证体系**：支持用户 ID / 注册邮箱登录；支持服务端密码 MD5 验证与明文过渡。
- **会话状态**：登录成功颁发双重凭据（`sessionId` 与 `csrfToken`），并缓存用户上次访问的组织与项目空间（`lastOrganizationId`, `lastProjectId`）。
- **生命周期**：
  - Active: 正常登录持有 Token；
  - Inactive: 登出注销或账号被管理员禁用 (`enable = false`)。

### 2.2 多租户组织域 (Organization)
- **多租户隔离**：组织作为最高层级租户空间，承载独立的项目集与用户角色关系。
- **状态流转**：
  - `ENABLED`：正常运行，组织内成员可正常创建项目与执行测试；
  - `DISABLED`：已停用，组织下接口被拦截；
  - `DELETED`：逻辑软删除，保证历史对账数据与审计日志不可篡改。

### 2.3 项目与质量空间域 (Project)
- **隔离粒度**：项目从属于特定组织，包含独立的质量工作空间（Quality Workspace）、用例树、环境配置（Environment）及缺陷流转。
- **多级寻址**：提供公开项目池（Public Projects）与组织限定项目池（Org Projects）。

---

## 3. 防资损与安全底线 (Security & Risk Baseline)
1. **防越权切换**：用户切换项目上下文（`/project/switch`）时，服务端强制校验用户在目标项目的归属关系，禁止跨租户横向越权。
2. **审计留痕**：所有的关键增删改操作（创建组织、删除项目、重置密码等）必须通过 `service.RecordAuditLog` 写入不可篡改的审计日志表。
3. **软删除安全网**：组织与项目等核心资产全面采用 `deleted = 1` 软删除策略，禁止物理删库，避免关联的自动化测试执行历史和用例数据孤儿化。
