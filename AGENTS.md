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
