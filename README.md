# TrueOne Anubis (`trueone-anubis`)

TrueOne 质量平台核心服务端（Go 原生微服务），提供质量工作台、测试计划、Git Diff 因果对账与代码 AST 静态分析引擎。

## 🎯 核心能力

1. **纯云端 AST 语法树解析与对账引擎**：
   - 毫秒级遍历代码仓库中的测试文件，无编译依赖提取测试函数、行号、断言与代码切片；
   - 自动关联 PRD 规约文档（`requirements-baseline.md`），推导覆盖状态（`COVERED` / `WARNING` / `BLOCKED` / `GAP`）。
2. **测试计划与三联屏数据中枢**：
   - 提供活体测试计划因果矩阵数据接口（`GET /quality-workspace/:id/diff-matrix`）；
   - 支持动态自动解析刷新（`POST /quality-workspace/:id/diff-matrix/auto-parse`）；
   - 支持端侧执行器（Runner）实时上报断言和数据库状态差分快照。
3. **企业级权限、项目与配置管理**：
   - 组织、项目、成员、菜单与操作审计日志全流程支撑。

---

## 🏗️ 架构与技术栈

- **框架**：Gin Web Framework
- **ORM**：GORM (MySQL 8.0)
- **配置与安全**：YAML 配置驱动 + JWT 鉴权 + 密码加盐哈希
- **端口**：默认监听 `8081`

---

## 🚀 本地开发与启动

```bash
# 1. 确保 MySQL 运行且配置正确 (config/config.yaml)
# 2. 编译并运行
go build -o trueone-anubis ./cmd/server
./trueone-anubis
```

健康检查：
```bash
curl http://localhost:8081/quality-workspace/page -H "Content-Type: application/json" -d '{"page":1,"pageSize":10}'
```

---

## 🔗 相关生态组件

- **`trueone-web`**：React 18 + Vite 前端三联屏大盘
- **`trueone-cli`**：面向 AI 与研发的 Test-as-Code 脚手架
- **`trueone-sdk`**：多语言统一契约与事件上报 SDK
