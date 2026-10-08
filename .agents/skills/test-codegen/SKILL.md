---
name: test-codegen
description: 负责生成规范的 Test-as-Code 测试用例代码，强制遵循 TrueOne SDK 规范
---

# Test Codegen Skill (测试代码生成)

本技能指导 AI 智能体生成符合工程质检标准的原生测试代码。

## 强约束规则 (Hard Rules)

1. **必须使用 SDK 注入元数据**：
   - Python: `@aegis.case(id="...", req="...", risk="P0")`
   - Go: `c := aegis.NewCase(t, "...", "...", aegis.RiskP0)`
   - Java: `@AegisCase(id="...", req="...", risk=RiskLevel.P0)`
2. **强制 AAA 三段式结构**：
   - 步骤 1: 初始化测试数据与前置上下文
   - 步骤 2: 执行核心业务操作并记录证据 (`attach_evidence`)
   - 步骤 3: 核心断言校验与副作用比对
3. **严禁弱断言**：不得仅校验 HTTP 200，必须校验关键业务字段和数据状态。
