# FateLumen 项目开发规则

## 仓库结构(monorepo)
- backend/  : Go 1.26 后端(独立 go.mod: fatelumen/backend)
- frontend/ : Next.js 前端(后端 API 跑通后再做)
- docs/FateLumen-开发任务书.md : 前后端共用的唯一权威文档(SSOT)

## 权威文档
docs/FateLumen-开发任务书.md 是本项目唯一事实来源。架构、数据库、API、接口抽象、
目录结构、技术选型，全部严格以该文档为准。编码前先完整阅读对应章节，不要凭猜测下手。

## 不可违背的核心原则
- P1：八字排盘是确定性计算，必须用 github.com/6tail/lunar-go，严禁用 LLM 排盘。
- P2：产品/前端/文案任何地方都不得出现 "AI" 字样。
- P3：LLM 输出必须是严格 JSON(json_object 模式)。
- P4：耗时任务走 JobQueue 接口(MVP 用 goroutine 池，可切 Asynq/Redis)。
- P5：解读引擎走 LLMProvider 接口，默认 DeepSeek(OpenAI 兼容协议)。

## 技术约束
- Go 1.26 / Gin / GORM v2 / MySQL 8.0(utf8mb4, JSON 字段) / Redis(可选)
- 业务层只依赖接口；实现在 cmd/server/main.go 注入；切换实现靠 .env。
- 排盘用 lunar-go；解读用 DeepSeek，两者职责严格分离。

## 工作方式
- 按文档第 17 章 Phase 0→7 顺序推进。
- 每个 Phase 完成后停下来等我确认，再进下一个，不要一次写完全部。
- 每个 Phase 先定义接口(第 6.1 章 9 个接口)，再写实现。
- 不发明文档没有的方案；有疑问先问我，不要假设。
- 写完代码要能 go build ./... 通过。

## Git 提交规范(Conventional Commits)
每个 Phase 验收通过后提交一个 commit,格式如下:

格式：<type>(<scope>): <简短描述>

- type 取值：feat(新功能) / fix(修复) / refactor(重构) / docs(文档) /
  test(测试) / chore(杂项配置) / perf(性能) / style(格式)
- scope：模块名,如 auth / payment / bazi / llm / admin / render(可省略)
- 描述：用中文,一句话说清做了什么,不加句号

提交前置条件:
- 必须先 `go build ./...` 通过,不提交无法编译的中间状态。
- 一个 Phase 一个主 commit;Phase 内独立小块可拆多个 commit。

示例:
- feat(skeleton): Phase 0 项目骨架 + 9 个核心接口定义
- feat(auth): Phase 1 认证 + 出生档案,AuthProvider 接口与默认实现
- fix(bazi): 修正闰月排盘时柱计算
- chore(ci): 添加 Dockerfile 与 .env.example

每个 Phase 完成并通过验收后,主动执行:
git add . && git commit -m "<规范消息>" 然后告诉我你提交了什么。

## 日志规范
所有代码必须遵守本节的日志规范。

### 统一日志库
- 统一使用 `backend/internal/pkg/logger`（基于 `log/slog` 的薄封装，提供 `Info/Warn/Error/Debug/Fatal` 方法）。
- 严禁裸 `fmt.Println` / `log.Print` / `log.Printf`。
- 业务代码通过 `logger.FromCtx(ctx)` 获取带 `trace_id` 的 `*slog.Logger`，不再通过构造函数注入。
- 启动阶段仍可用 `logger.New(level)` 创建独立 logger（仅限 main 函数）。

### 错误必打
每个 `err != nil` 在边界层必须打一条日志：
- **handler 层**：处理请求时发生错误 → `slog.Error`，带 `user_id`、`profile_id`、`reading_id` 等业务标识。
- **service 层**：业务逻辑失败 → `slog.Error` / `slog.Warn`，带业务上下文。
- **外部调用处**（LLM / chromedp / R2 / Cache / 支付 provider / Notifier）：失败必打 `slog.Error`，必带 `err` 字段 + 入参摘要。

### 外部调用日志细则
| 模块 | 失败 | 关键节点 | 重试 |
|---|---|---|---|
| LLM | `Error`：带 provider、model、err | `Info`：调用耗时 | `Warn`：每次重试 |
| chromedp | `Error`：带 format、err | — | — |
| R2 Storage | `Error`：带 key、err | — | — |
| Cache | `Error`：带 key、err | — | — |
| 支付 | `Error`：带 order_id、provider、err | `Info`：checkout/webhook 接收 | `Warn`：每次重试 |
| Notifier | `Error`：带 channel、err | — | — |

### 结构化字段
使用 slog 的 key-value 对，不拼字符串：
```go
slog.Error("llm call failed", "err", err, "provider", "deepseek", "model", "deepseek-chat")
slog.Warn("quota exceeded", "user_id", uid, "count", count)
```
反例：`slog.Error(fmt.Sprintf("llm call failed: %v", err))` ← 禁止

### 禁止记录的内容
- **敏感信息**：API Key、完整 JWT Token、密码、签名密钥。
- **用户隐私**：完整出生信息、姓名、邮箱全文（可打脱敏摘要，如邮箱前 2 字符 + `***`）。
- **命盘全量**：八字排盘结果只打 chart_hash 或行数摘要，不打完整干支/藏干/纳音/大运数组。

### 日志级别
| 级别 | 用途 |
|---|---|
| `Debug` | 开发调试细节（如排盘中间结果、prompt 长度） |
| `Info` | 关键流程节点（启动、路由注册、外部调用开始/结束、支付回调接收） |
| `Warn` | 可恢复异常（重试、降级、额度超限、token 即将过期） |
| `Error` | 需人工关注的错误（外部调用失败、数据异常、DB 写入失败） |

### trace_id 全链路贯穿
- 中间件 `Trace()` 为每个请求注入/透传 `trace_id`（优先 `X-Trace-Id` 请求头，无则生成 16 位 hex）。
- 所有日志必须通过 `logger.FromCtx(ctx)` 打印以自动携带 `trace_id`。
- 异步任务：`Job.TraceID` 由 `Enqueue` 从 ctx 读取写入，worker 执行时用 `WithTraceID` 重建 ctx。
- 新增的 service/外部调用必须透传 ctx，确保所有日志串在同一 trace。

### 合规检查
后续所有 Phase 的代码审查必须确认：
1. 每个外部调用失败路径都有 `Error` 日志。
2. 日志字段为结构化 key-value。
3. 无敏感信息泄露。
4. 所有业务日志通过 `logger.FromCtx(ctx)` 打印。
