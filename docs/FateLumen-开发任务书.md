# FateLumen · 八字命理网站 — 完整开发任务书

> **致开发者(opencode / DeepSeek)**：这是一份**自包含**的开发任务书。你将从零搭建整个项目，无需向任何人提问。所有技术选型、数据库表、API 契约、目录结构、前端设计规范、分阶段任务、LLM Prompt 模板都已在本文档内给出。**严格遵循本文档执行**；如遇文档未覆盖的细节，按「业界主流最佳实践 + 保持简单(MVP 优先)」原则自行决策，并在代码注释中标注 `// DECISION:` 说明你的选择。

---

## 目录

1. 项目总览与核心原则
2. 技术选型(已锁定)
3. 系统架构
4. 数据库设计(MySQL 8.0)
5. 后端 API 契约(OpenAPI 风格)
6. 后端项目结构(Go + Gin)
7. 八字排盘模块(确定性算法)
8. 报告生成模块(LLM + 异步状态机)
9. LLM Prompt 模板
10. 渲染模块(HTML → 图片 / PDF)
11. 支付模块(多渠道可插拔)
12. 认证模块(Google OAuth + JWT)
13. 前端项目结构(Next.js)
14. 前端设计系统规范(Design Tokens)
15. 前端多语言方案(i18n)
16. 部署与运维
17. 分阶段任务拆解(Phase 0 → Phase 7)
18. 验收清单
19. 后台管理系统(Admin)

---

## 1. 项目总览与核心原则

### 1.1 产品一句话
FateLumen 是一个面向**海外市场**的八字(四柱命理)在线测算网站：用户输入出生信息 → 系统**用确定性算法精确排盘并形成可追溯事实包** → 生成清晰、专业、易读的完整命理解读报告。

### 1.2 商业模式(MVP)
- **当前实施范围**：免费排盘 + 完整测算。简单测算入口暂时隐藏，本阶段不继续建设。
- **简单测算(Quick Reading，后续恢复)**：代码、接口和数据结构保留；恢复时必须复用完整报告的事实包，只裁剪章节和文案长度，不建立第二套基础计算。
- **完整测算(Full Reading)**：付费 **$5.99 / 次** 或消耗 **10 积分**，输出一份精美 PDF 报告（10 章）。
- 订阅模式**不做**，留到 v2。

### 1.3 五条不可违背的核心原则

| # | 原则 | 说明 |
|---|---|---|
| **P1** | **排盘是确定性算法，不是 LLM** | 干支、五行、十神、大运、流年**全部由算法库精确计算**，绝不交给大模型。大模型**只负责把已算好的命盘翻译成自然语言解读**。这是整个产品可信度的根基。 |
| **P2** | **文案/页面绝不出现 "AI" 字眼** | 前端所有语言版本、文档对外文案，**一律不出现 "AI"、"人工智能"、"artificial intelligence"** 等字眼。对外统一表述为「精密算法排盘 + 专业解读」。(内部代码注释/变量名不受限。) |
| **P3** | **LLM 输出结构化 JSON，不是自由文本** | 大模型必须返回**严格的结构化 JSON**(按预定义 schema)，便于稳定渲染 PDF/图片。禁止让模型直接吐 Markdown/HTML 长文。 |
| **P4** | **报告生成异步化** | 完整报告生成耗时(秒级，多次 LLM 调用)，必须**异步 + 数据库状态机**(pending→processing→done/failed)，前端轮询。异步调度经 `JobQueue` 接口抽象:MVP 用 goroutine + worker pool 实现(不引入消息队列),量大时切 Asynq(Redis),业务逻辑零改动。 |
| **P5** | **LLM Provider 抽象成接口** | 后端用 Go `interface` 抽象 LLM 调用,**默认接 DeepSeek API**(成本低、中文强),可一键切换 OpenAI / Claude。绝不在业务代码里硬编码某家 SDK。**注意:LLM 只做解读文案,排盘永远是 lunar-go(见 P1)。** |

### 1.4 目标用户与语言
- 海外华人 + 对东方玄学好奇的非华人。
- **前端支持四种语言：英文(en，默认)、简体中文(zh)、日文(ja)、韩文(ko)。** 语言可配置化扩展。

---

## 2. 技术选型(已锁定，不要改动)

### 2.1 后端

| 层 | 选型 | 说明 |
|---|---|---|
| 语言 | **Go 1.26**(go.mod `go 1.26`) | |
| Web 框架 | **Gin** | 生态最大、资料最多、生成代码最稳。路由层做薄抽象，便于未来替换。 |
| ORM | **GORM v2** | |
| 数据库 | **MySQL 8.0** | 字符集统一 `utf8mb4`；命盘/报告用 `JSON` 字段存储。 |
| 缓存 | **Redis**(可选) | MVP 可先用内存 + DB；每日免费额度计数建议用 Redis。先抽象成接口，内存实现兜底。 |
| 排盘库 | **`github.com/6tail/lunar-go`** | 确定性四柱排盘。**这是 P1 原则的落地库。** |
| LLM(解读) | **DeepSeek API**(默认) | 仅用于**解读文案生成**(P1:排盘绝不用 LLM)。通过 `LLMProvider` 接口抽象,可切 OpenAI / Claude。DeepSeek 成本低、中文术语理解强,适合八字解读。 |
| HTML→图/PDF | **chromedp**(headless Chrome) | VPS 需装 Chrome/Chromium + 中文字体。 |
| 支付 | **PaymentProvider 接口**(可插拔) | MVP 用 Stripe(`stripe-go`)Checkout + Webhook;预留 PayPal/Paddle,新增渠道实现接口即可。 |
| 认证 | **Google OAuth 2.0 + JWT** | 单设备登录(同一账号新登录踢旧 token)。 |
| 对象存储 | **Cloudflare R2**(S3 兼容) | 存生成的图片/PDF，无出口流量费。 |
| 配置 | **Viper** + `.env` | |
| 日志 | **log/slog**(Go 标准库) | 结构化日志，零依赖；用 `pkg/logger` 薄封装,业务只依赖封装,便于以后替换。 |
| 迁移 | **golang-migrate** 或 GORM AutoMigrate | MVP 用 AutoMigrate + 手写 SQL 种子。 |

### 2.2 前端

| 层 | 选型 | 说明 |
|---|---|---|
| 框架 | **Next.js 14+(App Router)** | React 系，SSR/SEO 友好(海外引流关键)。 |
| 语言 | **TypeScript** | |
| 样式 | **Tailwind CSS** | |
| 组件库 | **shadcn/ui** + **Radix UI** | |
| 图标 | **lucide-react** | |
| 动画 | **Framer Motion** | 落地页滚动渐显等。 |
| 多语言 | **next-intl** | 中/英/日/韩，配置化(`locales/*.json`)。 |
| HTTP | **fetch / axios** | |
| 部署 | **Vercel**(免费额度) | |

### 2.3 基础设施 / 成本

| 项 | 方案 | 月成本估算 |
|---|---|---|
| 后端 VPS | Hetzner / Vultr(2C2G 起) | ~$6–10 |
| 前端托管 | Vercel 免费额度 | $0 |
| 对象存储 | Cloudflare R2 | ~$0(低用量) |
| 数据库 | VPS 自托管 MySQL(MVP) | $0(含在 VPS) |
| **合计** | | **~$10/月** |

---

## 3. 系统架构

### 3.1 架构图(文字版)

```
[ 用户浏览器 ]
      │
      ▼
[ Next.js 前端 (Vercel) ]  ── 静态页/SSR + i18n + 调用后端 API
      │  HTTPS / JSON
      ▼
[ Go + Gin 后端 (VPS) ]
   │  ── 所有外部能力均经接口抽象 + 依赖注入(main 装配) ──
   ├── AuthProvider 接口 ──► Google / Apple / Email(可插拔) + JWT
   ├── Bazi 排盘服务 ──► [6tail/lunar-go] (确定性算法，无 LLM)
   ├── Reading 服务
   │      ├── Quick(暂时隐藏): 未来裁剪 Full 事实包 → Renderer 出图 → Storage
   │      └── Full : 排盘 → 确定性事实包 + 不可变快照 → JobQueue 异步任务
   │                    └── LLM(分批多次且逐次留痕) → 组装 → Renderer 出 PDF → Storage
   │                         └── 状态机: pending→processing→done/failed → Notifier 通知
   ├── LLMProvider 接口 ──► DeepSeek(默认)/ OpenAI / Claude(可切，仅解读)
   ├── Renderer 接口 ──► chromedp(可换 wkhtmltopdf/云渲染)→ 图片/PDF
   ├── JobQueue 接口 ──► goroutine(MVP)/ Asynq+Redis(量大切换)
   ├── Notifier 接口 ──► Resend/SendGrid(邮件)/ noop
   ├── PaymentProvider 接口 ──► Stripe / PayPal / Paddle(可插拔,Registry 注册)
   ├── Storage 接口 ──► Cloudflare R2 / S3 / OSS
   ├── Cache 接口 ──► Redis / 内存兜底
   ├── Admin: Resource 接口 ──► 结构化后台(注册即生成 CRUD)
   └── Credit/Quota 管理
      │
      ├──► [ MySQL 8.0 ]  用户/订单/命盘/报告(JSON)/积分/后台
      ├──► [ Redis ]      每日额度计数 + 任务队列(可选)
      └──► [ Cloudflare R2 ] 图片/PDF 文件
```

### 3.2 关键数据流

**简单测算(Quick，同步，暂时隐藏/后续恢复)**
1. 前端提交出生信息 → `POST /api/v1/readings/quick`
2. 后端校验每日额度(免费 3 次/天)
3. `lunar-go` 排盘 → 得到结构化命盘 JSON
4. 调 LLM(1 次)生成简短解读 JSON
5. 用 HTML 模板 + chromedp 渲染成图片 → 上传 R2
6. 返回图片 URL + 命盘摘要

**完整测算(Full，异步)**
1. 前端确认付费/扣积分 → `POST /api/v1/readings/full`(需已支付或有积分)
2. 后端创建 `report` 记录，状态 `pending`，立即返回 `report_id`
3. 后台 goroutine：`pending→processing`
  - `lunar-go` 排盘并生成版本化 `InterpretationFacts`
  - 保存报告输入、时间换算、命盘和事实包的不可变快照
  - 分 4 批调 LLM；每次尝试保存Prompt、事实输入、原始输出、解析输出、Schema结果、耗时与版本
  - 组装完整报告 JSON → HTML 模板 → chromedp 渲染 PDF → 上传 R2
  - `processing→done`(写入 PDF URL);异常则 `failed`(记录错误)
4. 前端轮询 `GET /api/v1/readings/full/{report_id}` 直到 `done`，拿 PDF URL

### 3.3 可扩展性总览(接口抽象清单)

> **设计铁律**:凡是「未来可能换实现 / 加渠道 / 加能力」的东西,一律藏在接口背后,通过 `main.go` 依赖注入装配。业务层(service)只依赖接口,永不依赖具体 SDK。新增能力 = 实现接口 + 注册/注入,**业务代码零改动**。

| 能力 | 接口 | MVP 实现 | 可扩展为 | 扩展成本 |
|---|---|---|---|---|
| 大模型(解读) | `LLMProvider` | DeepSeek | OpenAI / Claude / Gemini | 实现接口 + 注入 |
| 支付 | `PaymentProvider` + Registry | Stripe | PayPal / Paddle / 钱包 | 实现接口 + 注册 |
| 登录 | `AuthProvider` + Registry | Google | Apple / Facebook / 邮箱 | 实现接口 + 注册 |
| 渲染 | `Renderer` | chromedp | wkhtmltopdf / 云渲染 / 多格式 | 实现接口 + 注入 |
| 异步任务 | `JobQueue` | goroutine 池 | Asynq + Redis / 独立 worker | 实现接口 + 注入 |
| 通知 | `Notifier` | Resend(邮件) | SendGrid / 站内信 / 短信 | 实现接口 + 注入 |
| 对象存储 | `Storage` | Cloudflare R2 | S3 / OSS / 本地盘 | 实现接口 + 注入 |
| 缓存 | `Cache` | Redis(内存兜底) | 任意 KV | 实现接口 + 注入 |
| 后台模块 | `Resource` + Registry | 6 类资源 | 任意管理模块 | 实现接口 + 注册 |

**v2 远期扩展点(本期只留口,不实现):**
- **多术数体系**:未来若加紫微斗数 / 西洋占星(海外受众广),把 `bazi` 包升级为 `DivinationEngine interface`(`Compute(profile) → ChartData`),八字作首个实现;`charts` 表已是通用 `chart_data JSON`,无需改库。
- **i18n 语言注册表**:LLM「按 locale 出语种」用一张 `locale → 语言指令` 注册表(`map[string]LocaleSpec`),加语言 = 加一行 + 一个 `locales/xx.json`,避免散落 if-else。
- **订阅制**:`orders` + `credit_ledger` 结构已能承载;v2 加 `subscriptions` 表 + 一个 `SubscriptionPlan` 概念即可,不动现有支付抽象。

---

## 4. 数据库设计(MySQL 8.0)

> 字符集统一 `utf8mb4` / `utf8mb4_0900_ai_ci`。所有时间字段用 `DATETIME`(UTC 存储)。命盘、报告内容用 `JSON` 类型。GORM 模型与下表一一对应。

### 4.1 表清单

| 表名 | 用途 |
|---|---|
| `users` | 用户(Google 登录) |
| `birth_profiles` | 出生信息档案(一个用户可存多个，如给家人测) |
| `charts` | 排盘结果(确定性命盘 JSON，可复用缓存) |
| `readings` | 简单测算记录(出图) |
| `reports` | 完整测算报告(异步状态机 + PDF) |
| `orders` | 订单(支付渠道无关) |
| `payment_events` | 支付回调事件去重(Webhook 幂等) |
| `credit_ledger` | 积分流水(充值/消费) |
| `daily_quota` | 每日免费额度计数(若不用 Redis) |
| `admin_users` | 后台账号(独立于 C 端 users) |
| `admin_roles` | 后台角色 + 权限码(RBAC) |
| `admin_audit_log` | 后台操作审计日志 |
| `geo_countries` | 固定国家基础数据（ISO 代码 + 四语名称） |
| `geo_cities` | 固定城市基础数据（GeoNames ID、四语名称、经纬度、IANA 时区） |

### 4.2 建表 DDL

```sql
-- 用户
CREATE TABLE users (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  google_sub    VARCHAR(64)  NOT NULL UNIQUE COMMENT 'Google OAuth sub',
  email         VARCHAR(255) NOT NULL,
  name          VARCHAR(128),
  avatar_url    VARCHAR(512),
  credits       INT          NOT NULL DEFAULT 0 COMMENT '积分余额',
  locale        VARCHAR(8)   NOT NULL DEFAULT 'en',
  current_token_id VARCHAR(64) COMMENT '当前有效会话ID，用于单设备登录',
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  INDEX idx_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 出生信息档案
CREATE TABLE birth_profiles (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,
  display_name  VARCHAR(64)  COMMENT '昵称/备注，如"我"、"妈妈"',
  gender        TINYINT      NOT NULL COMMENT '0=female 1=male',
  calendar_type TINYINT      NOT NULL DEFAULT 0 COMMENT '0=solar 1=lunar',
  birth_year    SMALLINT     NOT NULL,
  birth_month   TINYINT      NOT NULL,
  birth_day     TINYINT      NOT NULL,
  birth_hour    TINYINT      NOT NULL COMMENT '0-23，未知用-1',
  birth_minute  TINYINT      NOT NULL DEFAULT 0,
  is_leap_month TINYINT      NOT NULL DEFAULT 0 COMMENT '农历闰月标记',
  birth_place   VARCHAR(128) COMMENT '用户可读出生地',
  country_code  VARCHAR(8)   COMMENT 'ISO 国家代码',
  region_code   VARCHAR(32)  COMMENT '地区代码',
  city          VARCHAR(96)  COMMENT '城市或明确地点',
  place_id      VARCHAR(128) COMMENT '地点解析服务稳定标识',
  timezone      VARCHAR(64)  COMMENT 'IANA Timezone ID，如 Asia/Shanghai',
  longitude     DECIMAL(9,6) COMMENT '经度，东经为正',
  latitude      DECIMAL(9,6) COMMENT '纬度，北纬为正',
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 排盘结果(确定性，可按 profile 哈希缓存复用)
CREATE TABLE charts (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  profile_id    BIGINT UNSIGNED NOT NULL,
  chart_hash    VARCHAR(64)  NOT NULL COMMENT '出生信息归一化后的哈希，命中即复用',
  chart_data    JSON         NOT NULL COMMENT '四柱/五行/十神/大运/流年 等结构化命盘',
  created_at    DATETIME     NOT NULL,
  UNIQUE KEY uk_hash (chart_hash),
  INDEX idx_profile (profile_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 简单测算(出图)
CREATE TABLE readings (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,
  profile_id    BIGINT UNSIGNED NOT NULL,
  chart_id      BIGINT UNSIGNED NOT NULL,
  locale        VARCHAR(8)   NOT NULL DEFAULT 'en',
  content       JSON         COMMENT 'LLM 生成的简短解读 JSON',
  image_url     VARCHAR(512) COMMENT '渲染图片 R2 URL',
  created_at    DATETIME     NOT NULL,
  INDEX idx_user (user_id),
  INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 完整测算报告(异步状态机)
CREATE TABLE reports (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,
  profile_id    BIGINT UNSIGNED NOT NULL,
  chart_id      BIGINT UNSIGNED NOT NULL,
  order_id      BIGINT UNSIGNED COMMENT '关联订单(付费方式时)',
  locale        VARCHAR(8)   NOT NULL DEFAULT 'en',
  status        VARCHAR(16)  NOT NULL DEFAULT 'pending' COMMENT 'pending/processing/done/failed',
  pay_method    VARCHAR(16)  NOT NULL COMMENT 'credit(统一扣积分) / unlimited(后台体验豁免)',
  content       JSON         COMMENT '10章完整报告结构化 JSON',
  pdf_url       VARCHAR(512) COMMENT 'PDF R2 URL',
  error_msg     VARCHAR(512) COMMENT '失败原因',
  retry_count   INT          NOT NULL DEFAULT 0,
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  INDEX idx_user (user_id),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 订单(支付渠道无关，provider 字段标识具体渠道)
CREATE TABLE orders (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id         BIGINT UNSIGNED NOT NULL,
  type            VARCHAR(16)  NOT NULL COMMENT 'report(单次报告) / credits(买积分包)',
  sku             VARCHAR(32)  NOT NULL COMMENT '商品标识，如 report_single / pack_50；金额由后端按 sku 查表',
  amount_cents    INT          NOT NULL COMMENT '金额(分)，如 599',
  currency        VARCHAR(8)   NOT NULL DEFAULT 'usd',
  credits_granted INT          NOT NULL DEFAULT 0 COMMENT '若买积分包，发放积分数',
  provider        VARCHAR(24)  NOT NULL COMMENT '支付渠道:stripe/paypal/paddle/...',
  provider_ref    VARCHAR(191) COMMENT '渠道侧主标识(Stripe session_id / PayPal order_id / Paddle txn_id)',
  provider_txn_id VARCHAR(191) COMMENT '渠道侧最终交易/支付意图 ID(payment_intent / capture_id)',
  provider_meta   JSON         COMMENT '渠道原始回执片段，调试/对账用',
  status          VARCHAR(16)  NOT NULL DEFAULT 'created' COMMENT 'created/pending/paid/failed/refunded',
  created_at      DATETIME     NOT NULL,
  updated_at      DATETIME     NOT NULL,
  INDEX idx_user (user_id),
  UNIQUE KEY uk_provider_ref (provider, provider_ref)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 支付回调事件去重表(所有渠道共用，保证 Webhook 幂等)
CREATE TABLE payment_events (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  provider      VARCHAR(24)  NOT NULL COMMENT 'stripe/paypal/paddle/...',
  event_id      VARCHAR(191) NOT NULL COMMENT '渠道侧事件唯一 ID(Stripe event.id / PayPal transmission_id 等)',
  event_type    VARCHAR(64)  NOT NULL COMMENT '已归一化的事件类型',
  order_id      BIGINT UNSIGNED COMMENT '关联订单',
  processed_at  DATETIME     NOT NULL,
  UNIQUE KEY uk_provider_event (provider, event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 积分流水
CREATE TABLE credit_ledger (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,
  delta         INT          NOT NULL COMMENT '正=充值 负=消费',
  balance_after INT          NOT NULL,
  reason        VARCHAR(64)  NOT NULL COMMENT 'purchase/consume_report/refund/gift',
  ref_id        BIGINT UNSIGNED COMMENT '关联 order_id 或 report_id',
  created_at    DATETIME     NOT NULL,
  INDEX idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 每日免费额度(若不用 Redis)
CREATE TABLE daily_quota (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  user_id       BIGINT UNSIGNED NOT NULL,
  quota_date    DATE         NOT NULL,
  used_count    INT          NOT NULL DEFAULT 0,
  UNIQUE KEY uk_user_date (user_id, quota_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ===== 后台管理(Admin)相关表 =====

-- 后台账号(与 C 端 users 完全独立)
CREATE TABLE admin_users (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(64)  NOT NULL,
  password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt',
  display_name  VARCHAR(64)  NOT NULL DEFAULT '',
  role_id       BIGINT UNSIGNED NOT NULL,
  totp_secret   VARCHAR(64)  COMMENT '可选二步验证',
  status        VARCHAR(16)  NOT NULL DEFAULT 'active' COMMENT 'active/disabled',
  last_login_at DATETIME,
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  UNIQUE KEY uk_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 角色(权限以 permission code 数组存 JSON，简单够用，无需独立权限表)
CREATE TABLE admin_roles (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  name          VARCHAR(64)  NOT NULL COMMENT 'super_admin/operator/viewer',
  permissions   JSON         NOT NULL COMMENT '权限码数组，如 ["user:read","order:read","catalog:write"] 或 ["*"]',
  created_at    DATETIME     NOT NULL,
  updated_at    DATETIME     NOT NULL,
  UNIQUE KEY uk_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 后台操作审计日志(所有写操作自动记录)
CREATE TABLE admin_audit_log (
  id            BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  admin_id      BIGINT UNSIGNED NOT NULL,
  admin_name    VARCHAR(64)  NOT NULL,
  action        VARCHAR(64)  NOT NULL COMMENT '资源:操作，如 order:refund / catalog:update',
  resource      VARCHAR(64)  NOT NULL,
  resource_id   VARCHAR(64),
  detail        JSON         COMMENT '前后值/参数快照',
  ip            VARCHAR(64),
  created_at    DATETIME     NOT NULL,
  INDEX idx_admin (admin_id),
  INDEX idx_resource (resource, resource_id),
  INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 全球出生地基础数据：由固定 GeoNames 快照一次性导入，不在运行期同步外部服务
CREATE TABLE geo_countries (
  code          CHAR(2)      PRIMARY KEY COMMENT 'ISO 3166-1 alpha-2',
  name_en       VARCHAR(191) NOT NULL,
  name_zh       VARCHAR(191) NOT NULL,
  name_ja       VARCHAR(191) NOT NULL,
  name_ko       VARCHAR(191) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE geo_cities (
  geoname_id    BIGINT       PRIMARY KEY,
  country_code  CHAR(2)      NOT NULL,
  admin1_code   VARCHAR(32)  NOT NULL DEFAULT '',
  admin1_name   VARCHAR(191) NOT NULL DEFAULT '',
  name_en       VARCHAR(191) NOT NULL,
  name_zh       VARCHAR(191) NOT NULL,
  name_ja       VARCHAR(191) NOT NULL,
  name_ko       VARCHAR(191) NOT NULL,
  latitude      DECIMAL(10,7) NOT NULL,
  longitude     DECIMAL(10,7) NOT NULL,
  timezone_id   VARCHAR(64)  NOT NULL,
  INDEX idx_geo_city_country (country_code),
  INDEX idx_geo_city_country_admin (country_code, admin1_code),
  CONSTRAINT fk_geo_city_country FOREIGN KEY (country_code) REFERENCES geo_countries(code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 4.3 命盘 JSON 结构(`charts.chart_data`)

```json
{
  "pillars": {
    "year":  {"stem": "戊", "branch": "午", "stem_element": "土", "branch_element": "火", "ten_god": "比肩", "hidden_stems": ["丁","己"]},
    "month": {"stem": "辛", "branch": "卯", "stem_element": "金", "branch_element": "木", "ten_god": "正官", "hidden_stems": ["乙"]},
    "day":   {"stem": "甲", "branch": "子", "stem_element": "木", "branch_element": "水", "ten_god": "日主", "hidden_stems": ["癸"]},
    "hour":  {"stem": "丙", "branch": "寅", "stem_element": "火", "branch_element": "木", "ten_god": "食神", "hidden_stems": ["甲","丙","戊"]}
  },
  "day_master": {"stem": "甲", "element": "木", "yin_yang": "阳"},
  "five_elements_count": {"wood": 3, "fire": 2, "earth": 2, "metal": 1, "water": 1},
  "strength": {"level": "weak", "score": 38, "favorable": ["water", "wood"], "unfavorable": ["metal", "earth"]},
  "luck_cycles": [
    {"start_age": 3, "start_year": 1995, "stem": "壬", "branch": "辰", "element": "水"}
  ],
  "current_year_fortune": {"year": 2026, "stem": "丙", "branch": "午", "element": "火"},
  "meta": {"solar_date": "1992-03-15T08:30:00Z", "lunar_date": "壬申年二月十二", "gender": "male", "calc_lib": "lunar-go", "calc_version": "1.x"}
}
```
> 字段命名以 `lunar-go` 实际可计算项为准；上表是目标结构，排盘服务负责把库的输出映射成此 schema。

### 4.4 完整报告 JSON 结构(`reports.content`，10 章)

```json
{
  "locale": "en",
  "summary_line": "一句话命局总结",
  "chapters": [
    {"no": 1,  "key": "destiny_depth",   "title": "...", "body": "..."},
    {"no": 2,  "key": "ten_gods_full",  "title": "...", "body": "..."},
    {"no": 3,  "key": "luck_cycle",     "title": "...", "body": "..."},
    {"no": 4,  "key": "ten_year_years", "title": "...", "body": "...", "years": []},
    {"no": 5,  "key": "career_depth",   "title": "...", "body": "..."},
    {"no": 6,  "key": "wealth_depth",   "title": "...", "body": "..."},
    {"no": 7,  "key": "love_depth",     "title": "...", "body": "..."},
    {"no": 8,  "key": "health_depth",   "title": "...", "body": "..."},
    {"no": 9,  "key": "element_tuning", "title": "...", "body": "..."},
    {"no": 10, "key": "life_plan",      "title": "...", "body": "..."}
  ]
}
```

**10 章 key 全集（渲染模板、Prompt 与管理端编排必须读取同一注册表，不得复制定义）：**

| no | key | 含义 | 额外字段 | 生成批次 |
|---|---|---|---|---|
| 1 | `destiny_depth` | 命格深析 | — | 独立章节调用 |
| 2 | `ten_gods_full` | 十神全览 | — | 独立章节调用 |
| 3 | `luck_cycle` | 大运走势 | — | 独立章节调用 |
| 4 | `ten_year_years` | 未来十年流年 | `years`:[{`year`,`ganzhi`,`note`}] | 总览 + 分段调用 |
| 5 | `career_depth` | 事业深析 | — | 独立章节调用 |
| 6 | `wealth_depth` | 财富格局 | — | 独立章节调用 |
| 7 | `love_depth` | 情感姻缘 | — | 独立章节调用 |
| 8 | `health_depth` | 健康养生 | — | 独立章节调用 |
| 9 | `element_tuning` | 五行调候 | — | 独立章节调用 |
| 10 | `life_plan` | 人生规划 | — | 独立章节调用 |

> 公共字段：每章必含 `no`、`key`、本地化 `title` 和 `body`。`years` 的年份与干支必须来自确定性事实包，解读层只能补充 `note`，不得增删、重排或改写。十章权威运行时注册表位于 `backend/internal/llm/prompts/registry.go`。

---

## 5. 后端 API 契约

> 统一前缀 `/api/v1`。所有响应统一信封：`{"code":0,"msg":"ok","data":{...}}`，`code=0` 成功，非 0 为业务错误码。鉴权接口需 `Authorization: Bearer <JWT>`。

### 5.1 认证

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/api/v1/auth/google/login` | 返回 Google OAuth 跳转 URL | 否 |
| GET | `/api/v1/auth/google/callback` | OAuth 回调，签发 JWT | 否 |
| POST | `/api/v1/auth/logout` | 登出(失效当前 token) | 是 |
| GET | `/api/v1/me` | 当前用户信息(含积分、locale) | 是 |
| PATCH | `/api/v1/me` | 更新 locale 等 | 是 |

### 5.2 出生档案

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/profiles` | 新建出生档案 |
| GET | `/api/v1/profiles` | 列出我的档案 |
| GET | `/api/v1/profiles/{id}` | 详情 |
| DELETE | `/api/v1/profiles/{id}` | 删除 |

**`POST /api/v1/profiles` 请求体：**
```json
{
  "display_name": "me",
  "gender": 1,
  "calendar_type": 0,
  "birth_year": 1992, "birth_month": 3, "birth_day": 15,
  "birth_hour": 8, "birth_minute": 30,
  "is_leap_month": 0,
  "birth_place": "Shanghai, China",
  "timezone": "Asia/Shanghai"
}
```

### 5.3 排盘

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/charts` | 按 profile 排盘(命中缓存直接返回)，返回命盘 JSON |
| GET | `/api/v1/charts/{id}` | 取命盘 |
| POST | `/api/v1/free-charts` | 登录用户免费排盘；不扣积分、不存档案/报告，保存独立排盘历史并返回统一命盘的裁剪 DTO |
| GET | `/api/v1/free-charts` | 分页查询当前用户的免费排盘历史 |
| GET | `/api/v1/free-charts/:id` | 查询当前用户的一条免费排盘详情 |
| DELETE | `/api/v1/free-charts/:id` | 删除当前用户的一条免费排盘历史 |
| POST | `/api/v1/free-charts/batch-delete` | 批量删除当前用户的免费排盘历史（单次最多 100 条） |

> 免费排盘接口与完整报告接口的前端、API 和 DTO 相互独立，但必须共同调用第 7 章定义的 `BirthChartEngine`，并共用 `chart_hash` 与命盘缓存。不得复制时区、真太阳时或 `lunar-go` 排盘逻辑。完整规范见 `docs/全球出生时间与真太阳时架构规范.md`。

### 5.3.1 全球出生地基础数据

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/api/v1/geo/countries` | 按语言分页搜索国家 | 是 |
| GET | `/api/v1/geo/cities` | 在指定国家内按语言分页搜索城市 | 是 |
| GET | `/api/v1/geo/cities/{id}` | 读取城市四语名称、经纬度与 IANA 时区 | 是 |

> 数据来自固定 GeoNames 快照，只保留国家、城市、一级行政区、四语名称、经纬度和 IANA 时区。没有任何可用城市记录或已经废止的 `AN/AQ/BV/CS/HM/UM` 不进入出生地国家基础库。中国地点必须存在真实中文别名，禁止将英文名回填后冒充 `name_zh`；缺少真实中文名的中国聚居点不进入候选库，中国一级行政区使用中文名称。数据通过 `cmd/import-geodata` 一次性导入；目标表已有数据时命令必须拒绝重复执行。客户端仅允许“国家 + 城市”正向取得经纬度，不提供经纬度反查国家/城市。

> 免费排盘与完整报告录入页都必须显示“出生记录时区”：选择城市后自动带出默认 IANA 时区，用户可在合法时区列表中校正。中国大陆城市默认使用 `Asia/Shanghai`，新疆用户明确确认出生记录采用新疆时间时可改为 `Asia/Urumqi`。后端始终按 `place_id` 重取固定经纬度，但使用用户最终确认且通过 `time.LoadLocation` 校验的时区；仅提交经纬度时，时区必填。

管理端提供精简校对页面：`GET /api/v1/admin/geo` 按国家或地区搜索并分页展示，`PATCH /api/v1/admin/geo/{id}` 只允许修正经纬度，且必须生成审计日志。不提供新增、删除或外部数据同步功能。

### 5.4 简单测算(同步)

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/readings/quick` | 出生信息 → 排盘 → LLM → 出图。受每日免费额度限制 |
| GET | `/api/v1/readings/{id}` | 取简单测算结果 |
| GET | `/api/v1/readings` | 我的历史 |

**`POST /api/v1/readings/quick` 请求体：** `{"profile_id": 123, "locale": "en"}`
**响应：** `{"reading_id":1,"image_url":"https://...","summary_line":"...","chart": {...}}`
**额度不足错误码：** `code=4290`, `msg="daily free quota exceeded"`

### 5.5 完整测算(异步)

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/readings/full` | 扣除 10 积分并创建完整报告任务，返回 `report_id` |
| GET | `/api/v1/readings/full/{report_id}` | 轮询状态；`done` 时返回 `pdf_url` + `content` |

**`POST /api/v1/readings/full` 请求体：** `{"profile_id":123,"locale":"en"}`
- 所有普通用户统一扣 10 积分；扣减、积分流水和报告创建必须在同一事务内完成，任一步失败全部回滚。
- 余额不足返回 `code=4020 insufficient credits`，不得创建报告任务。
- 现金购买报告也先由支付成功事件发放 10 积分，再调用同一报告创建入口消费 10 积分；报告生成逻辑不区分积分来源。
- 后台明确标记的无限体验用户保留豁免，不写积分消费流水。
  **响应：** `{"report_id":7,"status":"pending"}`
  **轮询响应(done)：** `{"report_id":7,"status":"done","pdf_url":"https://...","content":{...}}`

### 5.6 支付(多渠道抽象)

> 支付渠道**可插拔**。前端只认 `provider + sku`，永不传金额。Webhook 按渠道分路由,各自验签后归一化为统一事件再交给同一套订单处理逻辑。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/v1/payments/providers` | 返回当前启用的支付渠道列表(供前端动态渲染按钮) |
| POST | `/api/v1/payments/checkout` | 创建订单 + 调用指定 provider 起支付,返回跳转/确认信息 |
| POST | `/api/v1/payments/webhook/:provider` | 渠道回调(各自验签),归一化后标记订单 paid / 发积分 |
| GET | `/api/v1/orders` | 我的订单 |

**`GET /api/v1/payments/providers` 响应：**
```json
{"providers":[
  {"id":"stripe","label":"Card","enabled":true},
  {"id":"paypal","label":"PayPal","enabled":true}
]}
```

**`POST /api/v1/payments/checkout` 请求体：**
```json
{"provider": "stripe", "type": "report", "sku": "report_single", "profile_id": 123, "locale": "en"}
// 或 {"provider":"paypal","type":"credits","sku":"pack_50"}  // 买积分包
```
**响应（统一信封）：** `{"order_id":99,"provider":"stripe","action":"redirect","checkout_url":"https://checkout.stripe.com/..."}`
> `action` 取值：`redirect`(跳转托管页,Stripe/Paddle)或 `client_confirm`(返回 `client_token`,前端 SDK 内确认,如 PayPal)。前端按 `action` 分支处理,无需关心具体渠道。

### 5.7 管理端报告追溯接口

| 方法 | 路径 | 说明 | 鉴权 |
|---|---|---|---|
| GET | `/api/v1/admin/reports/:id/overview` | 查看报告轻量总览、阶段时间及十章进度 | Admin |
| GET | `/api/v1/admin/reports/:id/facts?section=input\|time\|chart\|interpretation` | 按区块查看冻结输入、时间换算、命盘或确定性事实；每次只投影一个大字段 | Admin |
| GET | `/api/v1/admin/reports/:id/execution/preflight` | 查看冻结的调用前预检结果 | Admin |
| GET | `/api/v1/admin/reports/:id/chapters` | 查看十章轻量状态、顺序、尝试数及最终采用attempt | Admin |
| GET | `/api/v1/admin/reports/:id/chapters/:chapterId/artifact?type=content\|prompt\|terminology\|raw_output\|validation` | 按类型读取单章正文或冻结大字段；历史报告允许从旧章节payload兼容读取 | Admin |
| GET | `/api/v1/admin/reports/:id/llm-calls?chapter_id=&page=&page_size=` | 按章节分页查看实际调用和重试元数据，不读取调用大字段 | Admin |
| GET | `/api/v1/admin/reports/:id/llm-calls/:callId` | 按需查看单次请求参数、Prompt、原始输出及解析输出 | Admin |
| GET | `/api/v1/admin/reports/:id/llm-calls/:callId/validation` | 独立查看该次调用后的校验摘要及逐条规则 | Admin |
| GET | `/api/v1/admin/reports/:id/validations` | 查看报告汇总校验轮次 | Admin |
| GET | `/api/v1/admin/reports/:id/validations/:validationId` | 查看汇总校验规则详情 | Admin |
| GET | `/api/v1/admin/reports/:id/result` | 查看最终冻结报告及内容哈希 | Admin |
| GET | `/api/v1/admin/reports/:id/integrity` | 按需重算事实、执行与内容哈希，查看冻结数据是否一致 | Admin |
| GET/PUT | `/api/v1/admin/settings/report` | 读取或保存十章并发数；新报告启动时冻结 | Admin |
| GET | `/api/v1/admin/settings/audit` | 分页查看供应商、模型和报告设置的安全变更记录 | Admin |

追溯接口必须以 `report_id` 限定资源归属，轻量列表不得联查Prompt、模型原始输出、冻结事实或正文大字段；大字段按区块、章节和用途独立读取。展示用JSON的中文标签、分组、折叠与格式化可由前端完成，但通过/拒绝、重试、最终采用attempt、哈希和冻结链归属只能由后端判定。完整契约、隐私边界与持久化草案见 `docs/深度解读基础数据与追溯规范.md`。C端Token必须被拒绝；接口不得返回API Key、认证头、邮箱、电话或支付凭证。

### 5.8 业务错误码表

| code | 含义 |
|---|---|
| 0 | 成功 |
| 4000 | 参数错误 |
| 4010 | 未登录/Token 失效 |
| 4011 | Token 被新设备登录踢下线 |
| 4020 | 积分不足 |
| 4030 | 订单未支付 |
| 4290 | 每日免费额度用尽 |
| 4040 | 资源不存在 |
| 5000 | 服务器内部错误 |
| 5001 | LLM 调用失败 |
| 5002 | 渲染(PDF/图片)失败 |


---

## 6. 后端项目结构(Go + Gin)

> 采用「分层 + 依赖注入」的清晰结构。`internal` 隔离业务代码。所有外部依赖(LLM/存储/缓存/支付)都通过接口暴露，便于测试与替换。

```
fatelumen-backend/
├── cmd/
│   └── server/
│       └── main.go              # 入口：加载配置、初始化依赖、启动 Gin
├── internal/
│   ├── config/
│   │   └── config.go            # Viper 加载 .env / config.yaml
│   ├── router/
│   │   └── router.go            # Gin 路由注册(薄抽象，集中所有路由)
│   ├── middleware/
│   │   ├── auth.go              # JWT 校验 + 单设备登录检查
│   │   ├── cors.go
│   │   ├── ratelimit.go
│   │   └── recovery.go          # panic 恢复 + 统一错误响应
│   ├── handler/                 # HTTP 层(只做参数校验 + 调 service + 返回)
│   │   ├── auth_handler.go
│   │   ├── profile_handler.go
│   │   ├── chart_handler.go
│   │   ├── reading_handler.go
│   │   ├── report_handler.go
│   │   └── payment_handler.go
│   ├── admin/                   # ★ 后台管理系统(与 C 端完全隔离，见第 19 章)
│   │   ├── server.go            # Admin 路由组装(/admin/api/v1)
│   │   ├── middleware/
│   │   │   ├── auth.go          # Admin JWT(独立密钥/Cookie)
│   │   │   ├── rbac.go          # 角色-权限校验(基于 permission code)
│   │   │   └── audit.go         # 操作审计日志(自动落 admin_audit_log)
│   │   ├── resource/            # ★ 结构化资源框架(CRUD 抽象核心)
│   │   │   ├── resource.go      # Resource 接口 + 通用 List/Detail/Update/Action
│   │   │   ├── registry.go      # 资源注册中心(自动生成路由 + 菜单元数据)
│   │   │   ├── query.go         # 统一查询 DSL(分页/筛选/排序/搜索)
│   │   │   └── schema.go        # 字段 Schema(类型/可筛/可排/枚举，驱动前端表格表单)
│   │   ├── resources/           # 各业务资源实现(每个 = 一个文件)
│   │   │   ├── user_resource.go
│   │   │   ├── order_resource.go
│   │   │   ├── report_resource.go
│   │   │   ├── reading_resource.go
│   │   │   ├── credit_resource.go
│   │   │   └── catalog_resource.go   # 商品/价格管理(读写 payment.Catalog)
│   │   ├── auth/                # 后台账号登录(账号密码 + 可选 TOTP)
│   │   │   └── admin_auth.go
│   │   └── dashboard/          # 数据看板聚合查询
│   │       └── dashboard.go
│   ├── service/                 # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── profile_service.go
│   │   ├── chart_service.go     # 排盘(调 bazi 包)、缓存
│   │   ├── reading_service.go   # 简单测算(同步)
│   │   ├── report_service.go    # 完整测算(异步状态机)
│   │   ├── payment_service.go   # 统一支付编排(下单/处理回调，依赖 PaymentProvider 接口)
│   │   ├── credit_service.go    # 积分增减(事务)
│   │   └── quota_service.go     # 每日免费额度
│   ├── repository/              # 数据访问层(GORM)
│   │   ├── user_repo.go
│   │   ├── profile_repo.go
│   │   ├── chart_repo.go
│   │   ├── reading_repo.go
│   │   ├── report_repo.go
│   │   ├── order_repo.go
│   │   ├── payment_event_repo.go
│   │   └── credit_repo.go
│   ├── model/                   # GORM 模型 + JSON 结构体
│   │   ├── user.go
│   │   ├── profile.go
│   │   ├── chart.go             # 含 ChartData 结构体(命盘 JSON)
│   │   ├── reading.go
│   │   ├── report.go            # 含 ReportContent 结构体（10章 JSON）
│   │   ├── order.go
│   │   ├── payment_event.go
│   │   └── credit.go
│   ├── bazi/                    # ★ 核心：八字排盘(确定性，封装 lunar-go)
│   │   ├── calculator.go        # 输入出生信息 → 输出 ChartData
│   │   ├── mapping.go           # 干支/五行/十神 中英日韩名称映射
│   │   └── strength.go          # 身强身弱、喜用神判定
│   ├── llm/                     # ★ LLM Provider 抽象(仅解读，不排盘)
│   │   ├── provider.go          # LLMProvider 接口定义
│   │   ├── deepseek.go          # DeepSeek 实现(默认，OpenAI 兼容协议)
│   │   ├── openai.go            # OpenAI 实现(可切)
│   │   ├── claude.go            # Claude 实现(预留)
│   │   └── prompts/             # Prompt 模板(见第 9 章)
│   │       ├── quick.go
│   │       └── full.go
│   ├── payment/                 # ★ 支付 Provider 抽象(可插拔多渠道)
│   │   ├── provider.go          # PaymentProvider 接口 + CheckoutResult/PaymentEvent
│   │   ├── registry.go          # 渠道注册中心 + 启用列表
│   │   ├── catalog.go           # SKU → 金额/币种/积分(后端定义)
│   │   ├── stripe.go            # Stripe 实现(MVP)
│   │   ├── paypal.go            # PayPal 实现(预留)
│   │   └── paddle.go            # Paddle 实现(预留)
│   ├── renderer/                # ★ 渲染抽象(HTML → 图片/PDF)
│   │   ├── renderer.go          # Renderer 接口
│   │   ├── chromedp.go          # chromedp 实现(MVP)
│   │   └── templates/           # 渲染用 HTML 模板(Go template)
│   │       ├── quick_image.html
│   │       └── full_report.html
│   ├── job/                     # ★ 异步任务抽象(P4 落地)
│   │   ├── queue.go             # JobQueue 接口 + Job 定义
│   │   ├── goroutine.go         # goroutine 实现(MVP，带 worker pool)
│   │   └── asynq.go             # Redis/Asynq 实现(预留，量大时切换)
│   ├── notify/                  # ★ 通知抽象(邮件/站内信)
│   │   ├── notifier.go          # Notifier 接口
│   │   ├── resend.go            # Resend/SendGrid 实现(MVP)
│   │   └── noop.go              # 空实现(本地/未配置时兜底)
│   ├── auth/                    # ★ 第三方登录抽象(可插拔)
│   │   ├── provider.go          # AuthProvider 接口 + Registry + ExternalUser
│   │   ├── google.go            # Google 实现(MVP)
│   │   ├── apple.go             # Apple 实现(预留)
│   │   └── email.go             # 邮箱+验证码 实现(预留)
│   ├── storage/                 # 对象存储抽象
│   │   ├── storage.go           # Storage 接口
│   │   └── r2.go                # Cloudflare R2(S3 兼容)实现
│   ├── cache/                   # 缓存抽象
│   │   ├── cache.go             # Cache 接口
│   │   ├── redis.go
│   │   └── memory.go            # 内存兜底实现
│   └── pkg/
│       ├── response/            # 统一响应信封 + 错误码
│       ├── jwt/                 # JWT 签发/解析
│       ├── hash/                # chart_hash 计算
│       └── logger/              # zap 封装
├── migrations/                  # SQL 迁移文件
├── configs/
│   └── config.example.yaml
├── .env.example
├── Dockerfile
├── docker-compose.yml           # 本地：MySQL + Redis + app
├── Makefile
├── go.mod
└── README.md
```

### 6.1 关键接口定义(必须实现)

```go
// internal/llm/provider.go
package llm

import "context"

// LLMProvider 抽象所有大模型调用。P5 原则：业务层只依赖此接口。
type LLMProvider interface {
    // GenerateJSON 给定 system + user prompt，返回严格 JSON 字符串。
    // 必须开启 provider 的 JSON mode / structured output。
    GenerateJSON(ctx context.Context, system, user string, opts ...Option) (string, error)
    Name() string
}

type Option func(*callConfig)
func WithTemperature(t float32) Option
```

> **DeepSeek 接入说明(默认实现 `deepseek.go`)**:DeepSeek API **完全兼容 OpenAI Chat Completions 协议**,直接复用 `github.com/sashabaranov/go-openai`,只改 BaseURL 即可:
> ```go
> cfg := openai.DefaultConfig(os.Getenv("DEEPSEEK_API_KEY"))
> cfg.BaseURL = "https://api.deepseek.com/v1"   // 关键：指向 DeepSeek
> client := openai.NewClientWithConfig(cfg)
> // model 用 "deepseek-chat"；JSON 模式用 ResponseFormat{Type:"json_object"}
> ```
> 所以 `openai.go` 与 `deepseek.go` 共用同一套 SDK 代码,差异仅在 BaseURL + model + ApiKey,可抽公共基类。切换 provider 只改 `.env` 的 `LLM_PROVIDER`。

```go
// internal/storage/storage.go
package storage

import "context"

type Storage interface {
    // Put 上传字节流，返回可公开访问 URL。
    Put(ctx context.Context, key string, data []byte, contentType string) (url string, err error)
}
```

```go
// internal/cache/cache.go
package cache

import (
    "context"
    "time"
)

type Cache interface {
    Incr(ctx context.Context, key string) (int64, error)
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key, val string, ttl time.Duration) error
}
```

```go
// internal/renderer/renderer.go
package renderer

import "context"

// 渲染输出格式，方便未来扩展 WebP/JPEG 等
type Format string
const (
    FormatPNG Format = "png"
    FormatPDF Format = "pdf"
)

// Renderer 抽象「HTML → 二进制图片/PDF」。业务层只依赖此接口，
// 未来可从 chromedp 换成 wkhtmltopdf / 云渲染服务，不动业务代码。
type Renderer interface {
    // Render 输入已渲染好的 HTML 字符串，返回目标格式的字节流。
    Render(ctx context.Context, html string, format Format) ([]byte, error)
}
```

```go
// internal/job/queue.go
package job

import "context"

// Job 是一个可执行的异步任务（如「生成完整报告」）。
type Job struct {
    Type    string // "generate_report"
    Payload []byte // JSON 序列化的任务参数（如 report_id）
}

// Handler 处理某类 Job。
type Handler func(ctx context.Context, payload []byte) error

// JobQueue 抽象异步任务调度（P4 落地）。
// MVP 用 goroutine + worker pool 实现；量大时换 Asynq(Redis) 实现，
// report 生成逻辑零改动——只换 main 里注入的实现。
type JobQueue interface {
    // 注册某类任务的处理器（启动时调用）
    Register(jobType string, h Handler)
    // 入队一个任务（异步执行，立即返回）
    Enqueue(ctx context.Context, job Job) error
    // 启动 worker（阻塞或后台）
    Start(ctx context.Context) error
    // 优雅停机：等待在途任务完成
    Shutdown(ctx context.Context) error
}
```

```go
// internal/notify/notifier.go
package notify

import "context"

type Message struct {
    To       string                 // 收件人邮箱（或用户 ID）
    Template string                 // 模板标识，如 "report_ready" / "payment_succeeded"
    Locale   string                 // 多语言模板
    Data     map[string]interface{} // 模板变量
}

// Notifier 抽象对外通知（邮件/站内信/未来短信）。
// MVP 接 Resend；未配置时用 noop 实现（仅打日志），不阻塞主流程。
type Notifier interface {
    Send(ctx context.Context, msg Message) error
    Channel() string // "email" / "inapp"
}
```

```go
// internal/auth/provider.go
package auth

import "context"

// ExternalUser 是各登录渠道归一化后的用户信息，业务层只认这个。
type ExternalUser struct {
    Provider   string // "google" / "apple" / "email"
    ExternalID string // 渠道侧唯一 ID（Google sub / Apple sub）
    Email      string
    Name       string
    AvatarURL  string
}

// AuthProvider 抽象第三方/邮箱登录，与 PaymentProvider 同套路。
// 新增登录方式 = 实现接口 + 注册，认证业务逻辑零改动。
type AuthProvider interface {
    ID() string // "google"
    // 返回授权跳转 URL（OAuth 类）；邮箱类可返回空并走 SendCode。
    AuthURL(state string) string
    // 用回调参数换取归一化用户信息（OAuth 用 code；邮箱用 code+email）。
    Exchange(ctx context.Context, params map[string]string) (*ExternalUser, error)
}

// Registry 与支付一致：按 .env AUTH_PROVIDERS 注册启用的渠道。
type Registry struct{ m map[string]AuthProvider }
func (r *Registry) Register(p AuthProvider) { r.m[p.ID()] = p }
func (r *Registry) Get(id string) (AuthProvider, bool) { p, ok := r.m[id]; return p, ok }
func (r *Registry) Enabled() []string { /* ... */ }
```

> **统一约定**:以上接口的具体实现都在 `cmd/server/main.go` 里**依赖注入**给 service 层。切换实现(如 goroutine→Asynq、Google→Apple)只改 main 的装配,业务代码一行不动。这是整个系统可扩展性的总闸。

---

## 7. 八字排盘模块(确定性算法)

> **这是 P1 原则的核心实现。绝不允许 LLM 参与排盘。** 全部用 `github.com/6tail/lunar-go` 计算。

### 7.0 全球出生时间与共享引擎（强制）

用户输入表示出生地当时的当地民用时间。系统必须依次完成地点标准化、IANA 时区、历史 UTC Offset、DST、当地标准时间、经度修正、平太阳时、均时差与真太阳时，然后再调用 `lunar-go`。全球用户不得统一转换为北京时间排盘。

默认固定：`time_calculation_mode=TRUE_SOLAR_TIME`、`day_boundary_rule=MIDNIGHT_00`、`lunar-go EightChar sect=2`。必须先处理真太阳时的自然跨日，再按 00:00 换日。

免费排盘与完整报告拥有独立前端、API 和 DTO，但共同调用唯一 `BirthChartEngine`：

```text
FreeChartService ─┐
                  ├─ BirthChartEngine → BirthChartResult + chart_hash
ReportService ────┘
```

`BirthChartEngine` 依次依赖 `BirthInputNormalizer`、`LocationResolver`、`HistoricalTimezoneResolver`、`SolarTimeEngine`、`BaziRuleEngine`、`LunarGoCalculator` 和 `ChartRepository`。具体规范、字段和测试矩阵见 `docs/全球出生时间与真太阳时架构规范.md`。

### 7.1 lunar-go 用法要点

- 主入口：`calendar.NewSolarFromYmdHms(...)` / `calendar.NewLunarFromYmd(...)`。
- 从 `Lunar` 对象可取 `EightChar`(八字)：四柱天干地支、`getDayGan`(日主)、十神、藏干、纳音。
- 大运：`EightChar.getYun(gender)` → `Yun.getDaYun()`。
- 流年：通过大运 + 年份推算。
- **务必处理**：农历闰月、子时跨日、时辰未知(birth_hour=-1 时按「日柱法」或标注时辰不明)。

### 7.2 calculator.go 职责

```go
// internal/bazi/calculator.go
package bazi

// Input 是已经完成地点、历史时区、DST、真太阳时和换日规则后的最终排盘输入。
type Input struct {
    Gender       int    // 0 female 1 male
    Year, Month, Day, Hour, Minute int
    DayBoundaryRule string // 默认 MIDNIGHT_00，对应 lunar-go sect=2
}

// Calculate 纯函数：输入出生信息 → 输出确定性命盘。
// 不调用任何网络/LLM。同样输入必返回同样输出。
func Calculate(in Input) (*model.ChartData, error)
```

### 7.3 名称映射(mapping.go)

干支/五行/十神等术语，提供**中英日韩四语映射表**，供前端按 locale 展示。例：

| 概念 | zh | en | ja | ko |
|---|---|---|---|---|
| 五行·木 | 木 | Wood | 木 | 목 |
| 十神·正官 | 正官 | Direct Officer | 正官 | 정관 |
| 日主 | 日主 | Day Master | 日主 | 일간 |

> 排盘结果里保留**原始汉字**(干支不翻译，文化符号);**术语类**(五行名、十神名)提供译名,前端按需展示。

### 7.4 chart_hash

`chart_hash` 必须覆盖性别、原始历法与出生日期时间、闰月、经纬度、IANA 时区、历史 UTC Offset、时间模式、换日规则、地点/时区数据库版本、太阳时间算法版本和 `lunar-go` 版本。任一影响结果的规则或版本变化都必须产生新哈希；免费排盘与完整报告共同命中该缓存。具体规范见专项文档第 10 章。

### 7.5 LunarGoCalculator 参考实现

> 以下是基于 `github.com/6tail/lunar-go` 的**完整参考实现**。lunar-go 的包路径为 `github.com/6tail/lunar-go/calendar`。**严格按此调用,不要自己发明 API。**

```go
// internal/bazi/calculator.go
package bazi

import (
    "fmt"
    "github.com/6tail/lunar-go/calendar"
    "yourmod/internal/model"
)

// Calculate 纯函数：最终真太阳时 → 确定性命盘。无网络/无 LLM，同输入同输出。
// 公农历转换、地点、历史时区、DST 与真太阳时必须在 BirthChartEngine 上游完成。
func Calculate(in Input) (*model.ChartData, error) {
    // Input 已是真太阳时对应的最终公历日期时间，可能与用户输入的民用日期不同。
    solar := calendar.NewSolarFromYmdHms(in.Year, in.Month, in.Day, in.Hour, in.Minute, 0)
    lunar := solar.GetLunar()

    // 2. 取八字（四柱）
    ec := lunar.GetEightChar()
    if in.DayBoundaryRule == "LATE_ZI_23" {
        ec.SetSect(1)
    } else {
        ec.SetSect(2) // 默认 MIDNIGHT_00
    }
    // 四柱：年/月/日/时 的天干地支字符串，如 "戊午"
    yearGZ := ec.GetYear()   // 年柱
    monthGZ := ec.GetMonth() // 月柱
    dayGZ := ec.GetDay()     // 日柱
    timeGZ := ec.GetTime()   // 时柱
    dayMaster := ec.GetDayGan() // 日主（日柱天干），如 "甲"

    // 3. 各柱拆解：天干、地支、藏干、十神、纳音
    //    ec.GetYearHideGan() 返回该柱地支藏干 []string
    //    ec.GetYearShiShenGan() 返回天干十神；ec.GetYearShiShenZhi() 返回地支藏干十神 []string
    pillars := model.Pillars{
        Year:  buildPillar(yearGZ, ec.GetYearShiShenGan(), ec.GetYearHideGan(), ec.GetYearShiShenZhi(), ec.GetYearNaYin()),
        Month: buildPillar(monthGZ, ec.GetMonthShiShenGan(), ec.GetMonthHideGan(), ec.GetMonthShiShenZhi(), ec.GetMonthNaYin()),
        Day:   buildPillar(dayGZ, "日主", ec.GetDayHideGan(), ec.GetDayShiShenZhi(), ec.GetDayNaYin()),
        Time:  buildPillar(timeGZ, ec.GetTimeShiShenGan(), ec.GetTimeHideGan(), ec.GetTimeShiShenZhi(), ec.GetTimeNaYin()),
    }

    // 4. 大运：需要性别（lunar-go: 1=男 0=女，与本项目 Input.Gender 一致）
    yun := ec.GetYun(in.Gender)
    var cycles []model.LuckCycle
    for _, dy := range yun.GetDaYun() { // []*DaYun
        cycles = append(cycles, model.LuckCycle{
            GanZhi:    dy.GetGanZhi(),       // 干支，如 "丁巳"
            StartAge:  dy.GetStartAge(),     // 起运虚岁
            StartYear: dy.GetStartYear(),    // 起运公历年
        })
    }

    // 5. 五行计数 + 身强身弱（strength.go 实现，见 7.x）
    elementCount := CountElements(pillars)        // map[string]int 木火土金水
    strength, favorable := JudgeStrength(pillars, dayMaster, elementCount)

    return &model.ChartData{
        Pillars:      pillars,
        DayMaster:    dayMaster,
        ElementCount: elementCount,
        Strength:     strength,     // "strong" / "weak" / "balanced"
        Favorable:    favorable,    // 喜用神五行，如 ["水","木"]
        LuckCycles:   cycles,
        HourUnknown:  false,
    }, nil
}

// buildPillar 组装单柱：拆天干地支 + 五行 + 十神 + 藏干
func buildPillar(ganzhi, shiShenGan string, hideGan, shiShenZhi []string, naYin string) model.Pillar {
    runes := []rune(ganzhi)
    stem, branch := string(runes[0]), string(runes[1])
    return model.Pillar{
        Stem:         stem,
        Branch:       branch,
        StemElement:  StemToElement(stem),   // mapping.go：甲乙→木 …
        BranchElement: BranchToElement(branch),
        TenGodStem:   shiShenGan,             // 天干十神（日柱为"日主"）
        HiddenStems:  hideGan,                // 地支藏干
        TenGodHidden: shiShenZhi,             // 藏干十神
        NaYin:        naYin,
    }
    _ = fmt.Sprint // 占位
}
```

> **DECISION 提示给 DeepSeek**:lunar-go 的 getter 方法名以实际库版本为准;若某 getter 名称不符,按"功能等价"原则查 lunar-go 源码对应方法,并在注释标 `// DECISION:`。**核心约束不变:四柱/十神/藏干/大运全部来自 lunar-go,严禁自行推算或调用 LLM。**

### 7.6 排盘正确性测试(对拍,必须通过)

> DeepSeek **必须**写这两个测试并通过,作为排盘正确性的"焊死"验证。期望值为公认八字结果。

```go
// internal/bazi/calculator_test.go
func TestCalculate_Case1(t *testing.T) {
    // 上游已完成地点、历史时区和真太阳时处理；此处只验证最终排盘时间。
    in := Input{Gender: 1, Year: 1990, Month: 8, Day: 15, Hour: 14, Minute: 30, DayBoundaryRule: "MIDNIGHT_00"}
    c, err := Calculate(in)
    require.NoError(t, err)
    // 期望四柱：庚午 / 甲申 / 壬子 / 丁未
    // （日柱为连续 60 甲子计数，无算法分歧；任意权威万年历对 1990-08-15 均得壬子日）
    assert.Equal(t, "庚", c.Pillars.Year.Stem)
    assert.Equal(t, "午", c.Pillars.Year.Branch)
    assert.Equal(t, "甲", c.Pillars.Month.Stem)
    assert.Equal(t, "申", c.Pillars.Month.Branch)
    assert.Equal(t, "壬", c.Pillars.Day.Stem)   // 日主
    assert.Equal(t, "子", c.Pillars.Day.Branch)
    assert.Equal(t, "丁", c.Pillars.Hour.Stem)  // 14:30 = 未时，五鼠遁得丁未
    assert.Equal(t, "未", c.Pillars.Hour.Branch)
    assert.Equal(t, "壬", c.DayMaster)
}

func TestCalculate_Deterministic(t *testing.T) {
    // 同输入两次，结果必须完全一致（P1 可复现性）
    in := Input{Gender: 0, Year: 2000, Month: 1, Day: 1, Hour: 0, Minute: 0, DayBoundaryRule: "MIDNIGHT_00"}
    a, _ := Calculate(in)
    b, _ := Calculate(in)
    assert.Equal(t, a, b)
}
```

> **验证方法**：该单元测试只验证最终真太阳时到 `lunar-go` 的映射；完整验收还必须覆盖专项规范第 12 章的地点、历史时区、DST、真太阳时、跨日和双接口一致性测试。若结果不符，必须先定位是上游时间链还是 `lunar-go` 适配问题，禁止通过硬编码干支修正。

---

## 8. 报告生成模块(LLM + 异步状态机)

### 8.0 事实快照与调用追溯（强制）

每份完整报告在首次LLM调用前必须生成并保存不可变 `InterpretationFacts` 快照；每次LLM尝试（包括失败和重试）必须独立保存Prompt、实际事实输入、原始输出、解析输出、Schema校验、模型参数、token、耗时、错误摘要、trace_id及版本。历史报告不得被新规则或新Prompt覆盖。

正式报告执行链使用全新 `full_report_*` 领域表，不复用旧 `reports`、`report_fact_snapshots`、`report_llm_calls` 或旧分组调用代码，不做双写。新链采用报告、执行快照、章节、调用尝试和最终结果分层持久化；轻量元数据与事实JSON、Prompt、原始输出、正文等大字段必须拆表。每份报告固定十条章节执行记录，通过 `report_id` 组合唯一索引查询；章节和调用的大字段分别通过唯一外键批量读取，列表与统计接口不得加载 payload。完成新服务切换后删除旧表与旧代码。完整目标结构、索引与清理策略以 `docs/深度解读基础数据与追溯规范.md` 第11章为准。

每次生成均创建新的 `report_id`，相同八字和相同客户允许重复生成。十章支持1～10的可配置受控并行，并在正式执行前冻结并发数、超时、自动重试与有序模型路由。管理端不得人工重跑或替换单章；报告完成前的失败只允许由系统按冻结策略追加调用尝试。`completed` 与 `failed` 均为不可变终态，再次生成必须创建新报告。

正式生成必须自动执行统一预检。十章模型结果必须依次通过JSON格式、章节Schema、模块完整度、确定性事实一致性、目标语言与术语、产品边界及报告级跨章一致性校验；校验失败自动追加尝试，策略耗尽则整份报告失败，不交付部分报告。十章全部成功后，最终内容、哈希与 `completed` 状态必须在一个完成事务中固化。

Go契约位于 `backend/internal/model/interpretation_facts.go` 与 `report_trace_contract.go`，完整专项规范见 `docs/深度解读基础数据与追溯规范.md`。B0阶段只固定契约，不注册AutoMigrate、不新增运行时路由。

### 8.1 状态机

```
            创建任务
pending ───────────────► processing ──成功──► done
   │                          │
   │                          └──失败──► failed ──(retry < 3)──► processing
   └── (积分/支付校验失败则直接不创建任务)
```

- 创建 `reports` 记录(status=pending) → **立即返回** report_id。
- 通过 `JobQueue.Enqueue` 入队任务;worker 执行时用 `defer` + recover 保证 panic 也能落到 `failed`。
- 失败按冻结的模型路由与重试预算自动重试；预算耗尽时，在同一事务内将整份报告标记为 `failed`、退还该报告实际扣除的积分并写反向流水。
- **超时控制**：单个 LLM 调用用 `context.WithTimeout`(如 60s);整个报告任务总超时(如 5min)。
- 报告 `done` 后调 `Notifier.Send`(模板 `report_ready`,按用户 locale)通知用户;PDF/图片经 `Renderer` 生成、`Storage` 上传。

### 8.2 分批生成策略(降低单次失败影响 + 控制 token)

10 章按注册表分章调用解读引擎；未来十年流年允许拆分为总览与分段年份调用，最终仍合并为同一章节：

| 批次 | 章节 |
|---|---|
| Batch 1 | 1.格局结构 2.日主强弱 3.性格 |
| Batch 2 | 4.事业 5.财运 6.婚姻 |
| Batch 3 | 7.健康 8.十年大运 9.流年 |
| Batch 4 | 10.五行调理 11.人际关系 12.终身总结 |

每批输入都带上**完整命盘 JSON**(确定的事实)，让 LLM 只做「基于这些事实写解读」。

### 8.3 report_service.go 核心伪代码

```go
func (s *reportService) CreateFullReport(ctx, userID, profileID int64, locale, payMethod string) (reportID int64, err error) {
    // 1. 校验支付/积分(事务内扣减，失败回滚)
    // 2. 排盘(命中缓存或新算) → chartID, chartData
    // 3. 创建 report 记录 status=pending
    // 4. go s.runGeneration(reportID, chartData, locale)  // 异步
    // 5. return reportID
}

func (s *reportService) runGeneration(reportID int64, chart *ChartData, locale string) {
    defer recoverToFailed(reportID)
    s.repo.UpdateStatus(reportID, "processing")
    content := &ReportContent{Locale: locale}
    for _, batch := range batches { // 4 批
        jsonStr, err := s.llm.GenerateJSON(ctx, buildSystem(locale), buildUser(batch, chart))
        // 解析 + 校验 schema + 失败重试
        appendChapters(content, jsonStr)
    }
    // 组装 summary_line
    pdfURL := s.renderer.RenderReportPDF(content, locale) // chromedp
    url := s.storage.Put(...) 
    s.repo.SaveDone(reportID, content, pdfURL)
}
```


---

## 9. LLM Prompt 模板

> **关键(P3 原则)**：所有 prompt 都要求模型**只输出严格 JSON**，且**只基于传入的命盘事实写解读，不得重新排盘、不得编造干支**。Prompt 里**不出现 "AI" 字眼**(对模型而言无所谓，但保持习惯一致)。`{{locale}}` 决定输出语言。

### 9.1 通用 System Prompt(所有调用共用)

```
You are a professional Chinese metaphysics (Bazi / Four Pillars of Destiny) interpreter.
You will be given a PRE-CALCULATED chart as JSON. The chart is computed by a deterministic
algorithm and is the ground truth — you MUST NOT recalculate, alter, or invent any pillar,
stem, branch, element, ten-god, or luck cycle. Your only job is to INTERPRET the given chart
into clear, professional, encouraging, non-fatalistic prose for a general audience.

Rules:
- Write ALL prose in the language specified by "locale" (en/zh/ja/ko).
- Keep Chinese characters (干支, e.g. 甲子) as-is; do not transliterate pillars.
- Be specific to THIS chart; reference its actual elements/strength/ten-gods.
- Tone: insightful, warm, empowering. Avoid doom, medical/financial/legal guarantees.
- Output STRICT JSON only. No markdown, no commentary, no code fences.
- Do NOT mention algorithms, models, or how the text was produced.
```

### 9.2 简单测算 Prompt(Quick，单次)

**User prompt 模板：**
```
locale: {{locale}}
chart: {{chart_json}}

Task: Produce a concise quick reading. Return STRICT JSON:
{
  "summary_line": "one vivid sentence capturing this person's destiny essence",
  "personality": "2-3 sentences on core character from the day master & elements",
  "strengths": ["...", "..."],
  "weaknesses": ["...", "..."],
  "element_note": "1 sentence on the five-element balance and favorable element"
}
```

### 9.3 完整测算 Prompt（Full，十章注册表编排）

**每章 User prompt 模板：**
```
locale: {{locale}}
chapter_key: {{chapter_key}}
facts_hash: {{facts_hash}}
input_facts: {{filtered_facts_json}}

Task: Interpret only the supplied deterministic facts for this chapter. Return STRICT JSON:
{
  "chapters": [
    {"no": {{chapter_no}}, "key":"{{chapter_key}}", "title":"...", "body":"..."}
  ]
}
Constraints:
- `title` 与 `body` 直接使用 `locale` 指定语言生成，不做二次翻译。
- 只能引用 `input_facts` 中存在的事实，不得自行排盘或补算干支、大运、流年和月份干支。
- `ten_year_years` 的 `years` 必须原样复用后端给出的年份和干支，只允许补充 `note`。
- 章节名称、key、顺序、默认事实和必需事实统一读取 `backend/internal/llm/prompts/registry.go`。
```

**语义摘要与多语言审核门禁：**

- 章节不得把完整事实 JSON 直接塞入 Prompt；先由统一语义摘要层按章节裁剪并转成自然、紧凑的确定性事实句，原始事实仍保留用于审计追溯。
- 每个已选事实必须返回 `available`、`unavailable` 或 `pending`，并计算章节覆盖率；禁止静默遗漏。
- 专业术语和专业语义短语分别维护审核状态：`approved`、`draft`、`missing`，用于上线前筛查，但不得阻断编排或调用。
- 中文术语与短语是首个审核基线；英文、日文、韩文必须使用经过专业审核的术语及句式模板，禁止以逐字直译替代专业本地化。
- 非中文调用只注入本章实际出现且已有目标译名的术语；缺失译名时允许模型按命理语境专业翻译，并要求同一报告内保持一致。
- 管理端主编排区保留章节基础指令、语言附加指令、最终完整指令三种预览；覆盖率和审核状态放入基础数据或诊断详情，不干扰编排主界面。

### 9.4 调用参数建议

| 参数 | 值 |
|---|---|
| model | `deepseek-chat`(默认,成本优先) / `gpt-4o-mini` / `gpt-4o`(质量优先),配置可切 |
| temperature | 0.7(解读需要文采，但别太发散) |
| response_format | `{"type":"json_object"}`(DeepSeek 与 OpenAI 同协议,均支持 JSON 模式) |

报告生成不得通过 Prompt 字数要求、调用参数或校验规则干预正文长度；由章节内容范围和 JSON 结构约束完整输出。

### 9.5 健壮性

- LLM 返回非法 JSON → 重试(最多 3 次)，再失败则该批降级为占位文案并标记 `partial`。
- 解析后**校验必需字段**(章节 key 齐全),缺失则补调。

---

## 10. 渲染模块(HTML → 图片 / PDF)

### 10.1 chromedp 要点

- VPS 安装 Chromium：`apt install chromium-browser`(或用官方 headless-shell 镜像)。
- **必装中文字体**：`fonts-noto-cjk`(覆盖中日韩),否则干支/中日韩文渲染成方块。
- Docker 部署推荐基础镜像：`chromedp/headless-shell` 或自行在镜像里装 chromium + noto-cjk。

### 10.2 图片渲染(Quick)

1. Go `html/template` 把 `reading.content` + 命盘 填入 `quick_image.html`。
2. chromedp 打开该 HTML(固定视口，如 1080×1350 竖版,适合社交分享)。
3. `chromedp.CaptureScreenshot` 或 `FullScreenshot` → PNG/JPEG 字节。
4. 上传 R2 → 返回 URL。

### 10.3 PDF 渲染(Full)

1. Go template 把 10 章 `report.content` 填入 `full_report.html`（A4 排版、封面、目录和章节）。
2. chromedp `page.PrintToPDF`(设置 A4、页边距、`printBackground=true`)。
3. 上传 R2 → 返回 URL。

### 10.4 渲染模板设计要求

- **复用 FateLumen 设计系统**(见第 14 章):暖骨纸底、Playfair 标题、暖金点缀、四柱命盘可视化。
- 模板里**不得出现 "AI" 字眼**。
- PDF 封面含品牌名 FateLumen、用户昵称、命盘四柱、生成日期。

### 10.5 quick_image.html 骨架(1080×1350 竖版分享图)

> Go `html/template`。变量来自 `reading.content`(见 9.2)+ 命盘。复用第 14 章 design token(暖骨纸底/暖金/Playfair)。**无 "AI" 字眼。**

```html
<!doctype html><html><head><meta charset="utf-8">
<style>
  @page { size: 1080px 1350px; margin:0; }
  body{margin:0;width:1080px;height:1350px;background:#ede8e0;color:#1a1715;
       font-family:'Inter','Noto Sans CJK SC',sans-serif;}
  .wrap{padding:72px 80px;display:flex;flex-direction:column;height:1206px;}
  .brand{font-family:'Playfair Display',serif;font-size:34px;color:#a8851a;letter-spacing:1px;}
  .summary{font-family:'Playfair Display',serif;font-size:54px;line-height:1.25;margin:40px 0 48px;}
  .pillars{display:flex;gap:20px;margin-bottom:48px;}
  .pillar{flex:1;text-align:center;background:#f5f1ea;border:1px solid #cec6b8;border-radius:16px;padding:28px 0;}
  .pillar .gz{font-size:48px;font-family:'Playfair Display',serif;}
  .pillar .label{font-size:18px;color:#8a8178;margin-top:8px;}
  /* 五行色：木#5c7060 火#b8473e 土#a8851a 金#9a8f7a 水#3f5a6b，按 stem_element 上色 */
  .section{margin-bottom:32px;}
  .section h3{font-size:22px;color:#a8851a;margin:0 0 10px;}
  .section p{font-size:26px;line-height:1.6;color:#5a544c;margin:0;}
  .foot{margin-top:auto;font-size:18px;color:#8a8178;display:flex;justify-content:space-between;}
</style></head>
<body><div class="wrap">
  <div class="brand">FateLumen · {{.DayMasterLabel}}</div>
  <div class="summary">{{.Content.SummaryLine}}</div>
  <div class="pillars">
    {{range .Pillars}}<div class="pillar">
      <div class="gz" style="color:{{.ElementColor}}">{{.Stem}}{{.Branch}}</div>
      <div class="label">{{.PositionLabel}}</div>
    </div>{{end}}
  </div>
  <div class="section"><h3>{{.T.Personality}}</h3><p>{{.Content.Personality}}</p></div>
  <div class="section"><h3>{{.T.ElementNote}}</h3><p>{{.Content.ElementNote}}</p></div>
  <div class="foot"><span>fatelumen.com</span><span>{{.GenDate}}</span></div>
</div></body></html>
```

### 10.6 full_report.html 骨架(A4 多页 PDF)

> chromedp `PrintToPDF`（A4 / printBackground=true）。封面 + 目录 + 10 章循环。变量来自 `report.content`（4.4 的 10 章结构）。

```html
<!doctype html><html><head><meta charset="utf-8">
<style>
  @page{size:A4;margin:18mm 16mm;}
  body{font-family:'Inter','Noto Sans CJK SC',sans-serif;color:#1a1715;}
  .cover{height:261mm;display:flex;flex-direction:column;justify-content:center;
         align-items:center;text-align:center;page-break-after:always;}
  .cover .mark{font-family:'Playfair Display',serif;font-size:72px;color:#a8851a;}
  .cover h1{font-family:'Playfair Display',serif;font-size:40px;margin:24px 0 8px;}
  .cover .meta{color:#8a8178;font-size:15px;line-height:1.8;}
  .cover .pillars{display:flex;gap:10px;margin-top:36px;}
  .cover .pillar{border:1px solid #cec6b8;border-radius:10px;padding:14px 18px;font-size:24px;
                 font-family:'Playfair Display',serif;}
  .toc{page-break-after:always;}
  .toc h2,.chapter h2{font-family:'Playfair Display',serif;color:#a8851a;font-size:24px;}
  .chapter{page-break-inside:avoid;margin-bottom:14px;}
  .chapter h2{border-bottom:1px solid #ddd6ca;padding-bottom:6px;}
  .chapter p{font-size:12.5px;line-height:1.75;color:#33302c;text-align:justify;}
  .cycles td{font-size:11px;padding:4px 8px;border-bottom:1px solid #eee;}
</style></head>
<body>
  <!-- 封面 -->
  <section class="cover">
    <div class="mark">天</div>
    <h1>FateLumen</h1>
    <div class="meta">{{.ProfileName}} · {{.GenDate}}<br>{{.T.FullReportTitle}}</div>
    <div class="pillars">
      {{range .Pillars}}<div class="pillar" style="color:{{.ElementColor}}">{{.Stem}}{{.Branch}}</div>{{end}}
    </div>
  </section>
  <!-- 目录 -->
  <section class="toc"><h2>{{.T.Contents}}</h2><ol>
    {{range .Content.Chapters}}<li>{{.Title}}</li>{{end}}
  </ol></section>
  <!-- 10 章正文：按 key 渲染，ten_year_years 额外渲染年份表格 -->
  {{range .Content.Chapters}}
  <section class="chapter">
    <h2>{{.No}}. {{.Title}}</h2>
    <p>{{.Body}}</p>
    {{if eq .Key "luck_cycles"}}<table class="cycles">
      {{range .Cycles}}<tr><td>{{.StartAge}}{{$.T.AgeUnit}}</td><td>{{.GanZhi}}</td><td>{{.Note}}</td></tr>{{end}}
    </table>{{end}}
    {{if eq .Key "yearly"}}<table class="cycles">
      {{range .Years}}<tr><td>{{.Year}}</td><td>{{.GanZhi}}</td><td>{{.Note}}</td></tr>{{end}}
    </table>{{end}}
  </section>
  {{end}}
</body></html>
```

> **关键约定**:模板里所有用户可见文字(标题/标签/页眉)走 `{{.T.xxx}}` 多语言字典(后端按 report.locale 注入),**与前端 i18n key 复用**;`{{.Pillars}}` 的 `ElementColor` 由后端按五行映射好再传入。DeepSeek 只需按此骨架补全细节,不要自创视觉风格。

---

## 11. 支付模块(多渠道可插拔)

> **设计目标**:面向海外市场,未来要接入多种收款渠道(信用卡/PayPal/本地钱包等)。支付层与 LLMProvider 一样抽象成 Go 接口,新增渠道 = 实现一个接口 + 注册,**不动业务逻辑、不动数据库结构**。

### 11.1 核心抽象 — `PaymentProvider` 接口

所有渠道实现同一接口;业务层只依赖接口,不依赖任何具体 SDK。

```go
package payment

// 统一的下单结果。前端按 Action 分支，不感知具体渠道。
type CheckoutResult struct {
    Action      string // "redirect" | "client_confirm"
    CheckoutURL string // redirect 模式：托管支付页
    ClientToken string // client_confirm 模式：前端 SDK 用
    ProviderRef string // 渠道侧主标识，写入 orders.provider_ref
}

// 已归一化的回调事件（各渠道验签+解析后产出，业务层只认这个）
type PaymentEvent struct {
    Provider    string
    EventID     string // 渠道事件唯一 ID，用于幂等
    Type        EventType // PaymentSucceeded / PaymentFailed / Refunded
    ProviderRef string // 用于回查订单
    ProviderTxnID string
    AmountCents int
    Currency    string
    Raw         []byte // 原始 payload，存 orders.provider_meta 供对账
}

type EventType string
const (
    EventPaymentSucceeded EventType = "payment_succeeded"
    EventPaymentFailed    EventType = "payment_failed"
    EventRefunded         EventType = "refunded"
    EventIgnored          EventType = "ignored" // 与本业务无关的事件
)

type CheckoutInput struct {
    Order       *Order // 已落库的订单（含 sku/amount/currency/type）
    UserID      uint64
    Locale      string
    SuccessURL  string
    CancelURL   string
}

// 每个渠道实现此接口
type PaymentProvider interface {
    // 渠道标识，如 "stripe"
    ID() string
    // 起支付：创建渠道侧会话/订单
    Checkout(ctx context.Context, in CheckoutInput) (*CheckoutResult, error)
    // 验签 + 解析回调，归一化为 PaymentEvent；验签失败返回 error
    ParseWebhook(ctx context.Context, headers http.Header, body []byte) (*PaymentEvent, error)
    // 可选：退款（v2 再实现，MVP 可返回 ErrNotSupported）
    Refund(ctx context.Context, order *Order, reason string) error
}
```

### 11.2 注册中心 — `ProviderRegistry`

```go
type Registry struct{ m map[string]PaymentProvider }

func (r *Registry) Register(p PaymentProvider) { r.m[p.ID()] = p }
func (r *Registry) Get(id string) (PaymentProvider, bool) { p, ok := r.m[id]; return p, ok }
func (r *Registry) Enabled() []string { /* 返回所有已注册 id */ }
```
- 启动时按 `.env` 里 `PAYMENT_PROVIDERS=stripe,paypal` 决定注册哪些(未配置密钥的不注册)。
- `GET /api/v1/payments/providers` 直接读 Registry.Enabled()。
- **新增渠道只需**:新建 `internal/payment/paddle.go` 实现接口 → 在 wire/初始化处 `registry.Register(NewPaddle(cfg))`。其余零改动。

### 11.3 商品目录(金额后端定义,SKU 驱动)

```go
// 金额绝不来自前端。前端只传 sku，后端查表得 amount/currency/credits。
var Catalog = map[string]SKU{
    "report_single": {Type: "credits", AmountCents: 599,  Currency: "usd", Credits: 10},
    "pack_50":       {Type: "credits", AmountCents: 999,  Currency: "usd", Credits: 50},
    "pack_120":      {Type: "credits", AmountCents: 1999, Currency: "usd", Credits: 120},
}
```
> 多币种/区域定价(v2):可把 Catalog 扩展成 `map[sku]map[currency]price`,或接 provider 侧的 Price 对象。MVP 先统一 USD。

### 11.4 统一支付流程

1. 前端 `GET /payments/providers` 拿到可用渠道 → 渲染按钮(Card / PayPal …)。
2. 用户选渠道点支付 → `POST /payments/checkout {provider,sku,type,profile_id,locale}`。
3. 后端:校验 sku → 查 Catalog → **落 `orders`**(status=created, provider, amount 来自后端)→ 调 `provider.Checkout()` → 写回 `provider_ref` → 返回 `CheckoutResult`。
4. 前端按 `action`:`redirect` 跳转 `checkout_url`;`client_confirm` 用 `client_token` 走渠道 SDK。
5. 渠道回调 `POST /payments/webhook/:provider` → 路由到对应 provider → **`ParseWebhook()` 验签 + 归一化** → 得 `PaymentEvent`。
6. 统一处理器:
  - 先查 `payment_events`(provider+event_id)去重,已处理直接 200。
  - 按 `provider_ref` 回查订单(事务内)。
  - `payment_succeeded` → 订单置 `paid`;所有积分商品（含单次报告商品）写 `credit_ledger` + 加 `users.credits`(同一事务)。
  - 写入 `payment_events` 标记已处理。
7. 前端支付成功页轮询订单状态；单次报告商品到账 10 积分后，由已保存的报告创建请求调用 `POST /readings/full`，再通过统一入口扣除 10 积分并触发生成。

### 11.5 安全与一致性红线

- **金额/商品在后端按 sku 定义**,前端禁传金额(防篡改)。
- **每个渠道独立验签**(Stripe 验 `Stripe-Signature`;PayPal 验 webhook signature;Paddle 验 HMAC)——验签在各 provider 的 `ParseWebhook` 内完成,业务层拿到的事件已可信。
- **Webhook 幂等**:`payment_events(provider,event_id)` 唯一键 + 订单状态双重保护,重复回调只处理一次。
- **下单幂等**:`orders(provider,provider_ref)` 唯一键,防同一渠道会话重复落单。
- 订单与发积分**同事务**,失败回滚。
- 报告首次进入最终失败状态时，状态更新、返还实际消费积分和反向 `credit_ledger` 必须同事务完成；章节失败、模型切换及单次调用重试不触发退款。
- Webhook 路由**不走 JWT 中间件**(渠道服务器无 token),仅靠验签鉴权。

### 11.6 MVP 落地范围

- **MVP 只实现 `StripeProvider`**(Action=redirect,Checkout Session + Webhook)。
- 接口、Registry、Catalog、统一处理器、`payment_events`/`orders` 结构**全部按多渠道设计就位**,这样接 PayPal/Paddle 时纯增量、零重构。
- `Refund` MVP 可返回 `ErrNotSupported`,v2 实现。

---

## 12. 认证模块(Google OAuth + JWT)

### 12.1 流程

1. `GET /api/v1/auth/google/login` → 返回 Google 授权 URL(带 state 防 CSRF)。
2. 用户授权后回调 `GET /api/v1/auth/google/callback?code=...&state=...`。
3. 后端用 code 换 token → 拿 Google 用户 `sub/email/name/avatar`。
4. `users` 表 upsert(按 `google_sub`)。
5. 生成 JWT(claims: `user_id`, `token_id`(随机 UUID), `exp`)，**把 `token_id` 写入 `users.current_token_id`**。
6. 返回 JWT 给前端(前端存 httpOnly cookie 或 localStorage)。

### 12.2 单设备登录

- 中间件校验 JWT 时，**额外比对** claims 里的 `token_id` 是否等于 `users.current_token_id`。
- 不等 → 说明已被新设备登录顶替 → 返回 `code=4011`,前端登出。

### 12.3 JWT 配置

- 算法 HS256，密钥从 env(`JWT_SECRET`)。
- 有效期 7 天;前端可在临期静默重登(MVP 可不做刷新 token，过期重新 OAuth)。


---

## 13. 前端项目结构(Next.js 14 App Router)

```
fatelumen-web/
├── app/
│   ├── [locale]/                # i18n 动态段(en/zh/ja/ko)
│   │   ├── layout.tsx           # 注入 next-intl provider、字体、全局样式
│   │   ├── page.tsx             # 落地页(由 bazi-landing-final.html 转成的组件)
│   │   ├── reading/
│   │   │   ├── page.tsx         # 出生信息表单 → 提交
│   │   │   └── [id]/page.tsx    # 简单测算结果(出图展示)
│   │   ├── report/
│   │   │   └── [id]/page.tsx    # 完整报告状态轮询 + PDF 展示/下载
│   │   ├── pricing/page.tsx
│   │   ├── account/page.tsx     # 我的档案/订单/积分
│   │   └── auth/callback/page.tsx
│   ├── api/                     # (可选)BFF 代理，转发到 Go 后端
│   └── globals.css              # Tailwind base + design tokens(CSS 变量)
├── components/
│   ├── landing/                 # 落地页拆出的 section 组件
│   │   ├── Hero.tsx
│   │   ├── FourPillarsChart.tsx # 四柱命盘可视化(复用 HTML 样板)
│   │   ├── TrustBar.tsx
│   │   ├── HowItWorks.tsx
│   │   ├── SampleReport.tsx
│   │   ├── Pricing.tsx
│   │   ├── Faq.tsx
│   │   └── CtaBand.tsx
│   ├── ui/                      # shadcn/ui 组件
│   ├── LanguageSwitcher.tsx     # EN/中/日/한 切换
│   ├── BirthForm.tsx            # 出生信息表单(公历/农历、时辰、性别、出生地)
│   └── ChartView.tsx
├── lib/
│   ├── api.ts                   # 封装后端调用 + JWT 注入
│   └── types.ts                 # 与后端 API 对齐的 TS 类型
├── locales/                     # ★ 多语言配置(配置化)
│   ├── en.json
│   ├── zh.json
│   ├── ja.json
│   └── ko.json
├── i18n.ts                      # next-intl 配置
├── middleware.ts                # next-intl 路由中间件(locale 检测/重定向)
├── tailwind.config.ts           # ★ 注入 design tokens
├── next.config.mjs
├── package.json
└── README.md
```

### 13.1 落地页迁移说明

**已交付的 `bazi-landing-final.html` 是落地页的视觉与文案唯一真源(single source of truth)。** 你的任务：
1. 把它**逐 section 拆成 React 组件**(见 `components/landing/`),HTML 结构、class、视觉效果**严格 1:1 还原**。
2. 文案**不要硬编码**,全部抽到 `locales/*.json`,组件里用 `useTranslations()` 取(HTML 里已有的 `data-i18n` key 直接复用作为 JSON key)。
3. CSS 变量(`:root` 里的颜色/字体)迁移到 `globals.css` + `tailwind.config.ts`(见第 14 章)。
4. 交互(FAQ 手风琴、滚动渐显、语言切换)用 React 状态 + Framer Motion / IntersectionObserver 重写。

---

## 14. 前端设计系统规范(Design Tokens)

> **这是整个前端的「设计宪法」。所有页面、组件、渲染模板(含后端 PDF/图片模板)都必须遵循。** 提取自已定稿的 FateLumen 落地页。

### 14.1 色彩(Color Tokens)

| Token | 值 | 用途 |
|---|---|---|
| `--bg` | `#ede8e0` | 主背景(暖骨纸) |
| `--bg-soft` | `#e4ddd2` | 次级背景(石色) |
| `--bg-card` | `#f5f1ea` | 卡片背景(更亮) |
| `--bg-dark` | `#181410` | 深色区块(CTA 卡片) |
| `--ink` | `#1a1715` | 主文字 |
| `--ink-soft` | `#5a544c` | 次级文字 |
| `--ink-faint` | `#8a8178` | 弱化文字/标签(暖灰) |
| `--line` | `#cec6b8` | 分隔线 |
| `--line-soft` | `#ddd6ca` | 浅分隔线 |
| `--gold` | `#c9a227` | 主点缀色(暖金) |
| `--gold-deep` | `#a8851a` | 深金(hover/强调) |
| `--gold-soft` | `#ece2c4` | 浅金(徽章底) |
| `--jade` | `#5c7060` | 五行·木(辅助色) |
| 五行火 | `#b8473e` | Fire |
| 五行土 | `#a8851a` | Earth |
| 五行金 | `#9a8f7a` | Metal |
| 五行水 | `#3f5a6b` | Water |

> **总原则**：暖骨纸 + 石色中性 + 暖金点缀;**不用任何高饱和色**;五行色仅用于命盘可视化的小色点。

### 14.2 字体(Typography)

| Token | 字体 | 用途 |
|---|---|---|
| `--serif` | `'Playfair Display', Georgia, serif` | 大标题、章节标题、品牌、价格(衬线，考究) |
| `--sans` | `'Inter', -apple-system, system-ui, sans-serif` | 正文、按钮、标签 |
| 中日韩字体 | `'Noto Sans SC/JP/KR'` 按 locale 加载 | 中日韩文正文(避免方块) |

- H1 桌面 ~64px / 移动 ~42px,`font-weight:500`,`letter-spacing:-.5px`,行高 1.08。
- eyebrow 小标签:12px、`letter-spacing:3px`、大写、`--ink-faint`。
- 章节序号用**罗马数字斜体**(I. II. III.)配 Playfair italic,暖金色。

### 14.3 形状与间距

| Token | 值 |
|---|---|
| `--radius` | 10px(卡片);按钮 8px |
| 内容最大宽 | `--maxw: 1080px` |
| section 纵向 padding | 桌面 92px / 移动 68px |
| 卡片内边距 | ~36–40px |

### 14.4 质感与动效

- **纸张颗粒**:`body::before` 用 SVG `feTurbulence` 噪点低透明叠加(已在样板里)。
- **滚动渐显**:元素入视口淡入上浮(IntersectionObserver / Framer Motion),阈值 ~0.12。
- **hover 微交互**:金色按钮上浮 1px + 阴影加深;卡片描边变深。
- **FAQ 手风琴**:＋号旋转 45°,内容高度过渡。
- **深色 CTA 卡片**:含「天」字大水印(品牌符号)。

### 14.5 tailwind.config.ts 注入示例

```ts
export default {
  theme: {
    extend: {
      colors: {
        bg: '#ede8e0', 'bg-soft': '#e4ddd2', 'bg-card': '#f5f1ea', 'bg-dark': '#181410',
        ink: '#1a1715', 'ink-soft': '#5a544c', 'ink-faint': '#8a8178',
        line: '#cec6b8', 'line-soft': '#ddd6ca',
        gold: '#c9a227', 'gold-deep': '#a8851a', 'gold-soft': '#ece2c4', jade: '#5c7060',
      },
      fontFamily: {
        serif: ['"Playfair Display"', 'Georgia', 'serif'],
        sans: ['Inter', 'system-ui', 'sans-serif'],
      },
      maxWidth: { content: '1080px' },
      borderRadius: { card: '10px' },
    },
  },
}
```

---

## 15. 前端多语言方案(i18n)

> **配置化是硬要求。** 文案绝不硬编码在组件里;新增语言 = 加一个 `locales/xx.json`,代码零改动。

### 15.1 方案:next-intl

- 路由:`app/[locale]/...`,支持 `/en`、`/zh`、`/ja`、`/ko`。
- `middleware.ts` 自动检测浏览器语言 → 重定向到对应 locale;默认 `en`。
- `locales/en.json` 为基准,key 命名**直接复用落地页 HTML 里的 `data-i18n` key**(如 `hero.title`、`pricing.full` 等),迁移零摩擦。

### 15.2 locales JSON 结构(节选,以已定稿 HTML 文案为准)

```jsonc
// locales/en.json
{
  "brand": "FateLumen",
  "nav": { "how": "How it works", "pricing": "Pricing", "faq": "FAQ", "signin": "Sign in", "start": "Get started" },
  "hero": {
    "eyebrow": "Authentic Chinese Astrology",
    "title": "Decode your birth chart, read your destiny.",
    "sub": "Enter your birth details and receive a precise Bazi...",
    "cta1": "Get your free reading", "cta2": "See how it works",
    "note": "No credit card required · 3 free readings every day"
  },
  "pricing": { "full": "Full Report", "unit": "/ report", "note": "Payments securely processed with Stripe." }
  // ... 其余 key 完整对齐已定稿 HTML 的 I18N 字典(en/zh/ja/ko 四份均已在 HTML 样板中给出，直接搬运)
}
```

> **重要**:已定稿的 `bazi-landing-final.html` 内联了 en/zh/ja/ko **四份完整文案字典**。直接把它们拆成四个 JSON 文件即可,**无需重新翻译**。

### 15.3 语言切换组件

- nav 右上角下拉:English / 简体中文 / 日本語 / 한국어。
- 切换 = 切换路由 locale 段 + 持久化(cookie / localStorage)。
- 后端 `users.locale` 同步保存(登录用户)。

### 15.4 全链路 locale 传递

用户选的 locale 要一路传到后端:
- 排盘术语译名(第 7.3)、LLM 解读语言(第 9 章 `{{locale}}`)、PDF/图片渲染模板语言,**全部按该 locale**。


---

## 16. 部署与运维

### 16.1 环境变量(`.env.example`)

```bash
# --- App ---
APP_ENV=production
APP_PORT=8080
APP_BASE_URL=https://api.fatelumen.com
WEB_BASE_URL=https://fatelumen.com

# --- MySQL ---
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=fatelumen
DB_PASSWORD=__set_me__
DB_NAME=fatelumen
DB_CHARSET=utf8mb4

# --- Redis (可选) ---
REDIS_ADDR=127.0.0.1:6379
REDIS_PASSWORD=

# --- JWT ---
JWT_SECRET=__set_me__
JWT_EXPIRE_HOURS=168

# --- Admin 后台 ---
ADMIN_JWT_SECRET=__set_me__        # 必须与 C 端 JWT_SECRET 不同
ADMIN_JWT_EXPIRE_HOURS=2
ADMIN_BOOTSTRAP_USER=admin         # 首个超管账号(迁移脚本创建)
ADMIN_BOOTSTRAP_PASSWORD=__set_me__

# --- Auth Providers (可插拔登录) ---
AUTH_PROVIDERS=google         # 逗号分隔，如 google,apple,email
# Google OAuth
GOOGLE_CLIENT_ID=__set_me__
GOOGLE_CLIENT_SECRET=__set_me__
GOOGLE_REDIRECT_URL=https://api.fatelumen.com/api/v1/auth/google/callback
# Apple (预留)
APPLE_CLIENT_ID=
APPLE_TEAM_ID=
APPLE_KEY_ID=

# --- Renderer ---
RENDERER=chromedp             # chromedp(默认) | 预留其它
CHROMEDP_BIN=                 # 留空走系统 Chromium

# --- Job Queue ---
JOB_QUEUE=goroutine           # goroutine(MVP) | asynq
JOB_WORKERS=4                 # goroutine 实现的并发 worker 数

# --- Notifier ---
NOTIFIER=resend               # resend | sendgrid | noop
RESEND_API_KEY=
NOTIFY_FROM=noreply@fatelumen.com

# --- LLM (仅解读，不排盘) ---
LLM_PROVIDER=deepseek         # deepseek(默认) | openai | claude
# DeepSeek (默认，OpenAI 兼容协议)
DEEPSEEK_API_KEY=__set_me__
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_MODEL=deepseek-chat
# OpenAI (可切)
OPENAI_API_KEY=
OPENAI_MODEL=gpt-4o-mini

# --- Payment (多渠道) ---
PAYMENT_PROVIDERS=stripe          # 逗号分隔，决定注册哪些渠道，如 stripe,paypal
PAYMENT_SUCCESS_URL=https://fatelumen.com/pay/success
PAYMENT_CANCEL_URL=https://fatelumen.com/pricing
# 商品目录(sku:cents:credits，type=report 时 credits=0)
CATALOG=report_single:599:0,pack_50:999:50,pack_120:1999:120
# Stripe
STRIPE_SECRET_KEY=__set_me__
STRIPE_WEBHOOK_SECRET=__set_me__
# PayPal (预留，启用 paypal 时填)
PAYPAL_CLIENT_ID=
PAYPAL_CLIENT_SECRET=
PAYPAL_WEBHOOK_ID=
PAYPAL_ENV=sandbox                # sandbox / live

# --- Cloudflare R2 ---
R2_ACCOUNT_ID=__set_me__
R2_ACCESS_KEY_ID=__set_me__
R2_SECRET_ACCESS_KEY=__set_me__
R2_BUCKET=fatelumen
R2_PUBLIC_BASE=https://cdn.fatelumen.com

# --- Renderer ---
CHROMIUM_PATH=/usr/bin/chromium
```

### 16.2 部署拓扑

- **前端**:Vercel(连 GitHub 自动部署),环境变量配后端 API 地址。
- **后端**:VPS 上用 Docker Compose 跑 `app + mysql + redis`;Nginx/Caddy 反代 + HTTPS(Let's Encrypt)。
- **对象存储**:Cloudflare R2 + 自定义域 CDN(`cdn.fatelumen.com`)。
- **域名**:`fatelumen.com`(前端)、`api.fatelumen.com`(后端)、`cdn.fatelumen.com`(文件)。

### 16.3 Dockerfile 要点(后端含 chromium)

- 基于 `golang:1.26` 编译,运行阶段基于 `debian-slim`,装 `chromium` + `fonts-noto-cjk`。
- 或后端不内嵌 chromium,单独跑 `chromedp/headless-shell` 容器,后端通过 remote allocator 连它(更省内存,512MB 限制下更稳)。

### 16.4 海外合规/体验

- 静态资源走 CDN(R2 + Cloudflare),保证海外访问速度(对应需求里的「海外 + CDN」P0 项)。
- 隐私政策/服务条款页必须有(GDPR 友好:说明数据用途、提供删除入口)。

---

## 17. 分阶段任务拆解(Phase 0 → Phase 7)

> 按顺序执行。每个 Phase 结束都应可独立运行/验证。**先打通主流程,再做支付与打磨。**

### Phase 0 — 项目脚手架(地基)
- [ ] 后端:初始化 Go module、Gin、GORM、Viper、log/slog、目录结构(第 6 章)。
- [ ] 后端:**先定义全部核心接口**(`LLMProvider`/`PaymentProvider`/`AuthProvider`/`Renderer`/`JobQueue`/`Notifier`/`Storage`/`Cache`/`Resource`,第 6.1 章),并在 `main.go` 搭好依赖注入装配骨架。
- [ ] 后端:`docker-compose`(MySQL + Redis + app),AutoMigrate 建表(第 4 章)。
- [ ] 后端:统一响应信封 + 错误码 + recovery 中间件。
- [ ] 前端:`create-next-app`(TS + App Router + Tailwind),装 shadcn/ui、next-intl、framer-motion。
- [ ] 前端:落地页 1:1 还原(把 `bazi-landing-final.html` 拆组件 + 接入 i18n,四语 JSON 落地)。
- **验收**:落地页四语切换正常;后端 `/health` 通;DB 表建好。

### Phase 1 — 认证 + 出生档案
- [ ] `AuthProvider` 接口 + Registry + Google 实现(走通后即可扩展 Apple/邮箱)。
- [ ] JWT + 单设备登录中间件。
- [ ] `GET /me`、profiles CRUD。
- [ ] 前端:登录流程、账户页、出生信息表单(公历/农历、时辰、性别、出生地、时区)。
- **验收**:能用 Google 登录、能创建/查看出生档案;**新增 mock auth provider 仅需实现接口 + 注册**。

### Phase 2 — 管理端基础、内容运营、档案联动与排盘(核心 P1)

> **范围调整说明**：本阶段先建立独立管理端与内容运营能力，同时完成出生档案/报告资料联动和确定性排盘。管理端的“报告管理”不在本阶段实现；它依赖用户端报告创建、状态与结果链路完成后，再在后续管理端深化阶段接入，避免重复建设不稳定的字段和页面。

#### 2.1 管理端独立认证与安全边界
- [ ] 管理端使用独立入口 `/admin/login`，仅支持用户名、密码、图形验证码登录；不支持 Google 登录、注册或找回密码入口。
- [ ] C 端用户 JWT 与管理端 Admin JWT 完全隔离：独立密钥 `JWT_SECRET` / `ADMIN_JWT_SECRET`、独立 token 存储键、独立认证中间件与会话失效逻辑；两类 token 互相不能访问对方接口。
- [ ] 管理员暂不做细粒度权限；所有状态正常的管理员拥有全部后台功能。保留 `admin_users`、审计日志与后续 RBAC 扩展基础，但不得再以 `users.role` 作为后台鉴权依据。
- [ ] 提供图形验证码接口与登录接口；验证码仅存 Cache，建议有效期 5 分钟、一次性使用并限制错误次数。登录失败统一返回，不泄露用户名、密码或验证码的具体错误原因；后台登录接口需限流并记录不含敏感信息的安全日志。
- [ ] 管理员退出只清除管理端会话，不影响 C 端用户会话；反之亦然。
- [ ] 管理员初始化采用私有脚本：`backend/init/README.md` 可提交，`backend/init/private/001-admin.sql` 必须被忽略且不上传 Git。脚本须使用 bcrypt 密码哈希并可重复执行；明文密码、真实哈希、JWT 密钥均不得进入仓库。
- **验收**：管理员可用账号密码与图形验证码登录后台；普通用户不能访问后台；后台 token 与用户 token 不能混用；所有本地凭证和私有初始化数据均未进入 Git。

#### 2.2 管理端壳与用户只读管理
- [ ] 建立独立后台布局、导航、管理员信息区与退出入口；未登录访问 `/admin/*` 自动跳转 `/admin/login`，会话失效自动退出。
- [ ] 建立可复用的后台列表能力：分页、搜索、筛选、加载中、空状态、错误提示与确认弹窗，为后续内容、价格与报告模块复用。
- [ ] 实现用户只读列表与详情：展示用户 ID、昵称、脱敏邮箱、登录方式、注册/最近登录时间、账户状态、积分、出生档案数量，并预留报告数量与最近报告时间字段。
- [ ] 用户详情展示基础资料、账户资料与出生档案摘要；完整出生资料默认不得暴露。密码哈希、JWT、Google 凭证等敏感数据绝不返回。
- [ ] 提供仅管理员可调用的 `GET /api/v1/admin/users`、`GET /api/v1/admin/users/:id`，支持安全的分页、关键字搜索与状态/登录方式筛选。
- [ ] 本阶段不实现后台报告列表、报告详情、重试、删除或人工处理；报告模块在用户端报告链路稳定后再接入。
- **验收**：管理员能登录、浏览用户分页列表和详情；普通用户无法调用后台用户接口；敏感资料不会被返回或显示。

#### 2.3 内容中心：八字知识、FAQ、客户案例
- [ ] 将八字知识库、常见问题、客户案例设计为后台可维护内容，而非硬编码前端 Markdown。
- [ ] 八字知识支持标题、slug、分类、标签、摘要、封面、Markdown 正文、语言、排序、状态及发布时间的创建、编辑、草稿、预览、发布、取消发布与归档。
- [ ] FAQ 支持分类、问题、回答、语言、排序与发布状态的管理。
- [ ] 客户案例支持标题、摘要、正文、主题标签、适合人群、封面、语言、排序、状态、匿名化展示名称与授权/隐私说明；不得对外暴露真实姓名、出生资料或联系方式。
- [ ] 内容统一生命周期：`draft → preview → published → unpublished / archived`。用户端只读取 `published` 内容；优先归档而非物理删除；每次写操作记录管理端审计日志。
- [ ] 四语内容支持 en / zh / ja / ko；缺少翻译时按既定规则回退英文。Markdown 渲染必须经过安全过滤，不得直接注入原始 HTML 或脚本。
- [ ] C 端 Learn、FAQ、Cases 页面接入只读公共接口：`/api/v1/public/knowledge`、`/api/v1/public/faqs`、`/api/v1/public/cases` 及相应 slug 详情接口；后台管理接口不得暴露给用户端。
- [ ] 初期允许管理员逐篇录入现有 Markdown；批量 Markdown 导入工具作为后续独立任务，不阻塞本阶段。
- **验收**：管理员可管理并发布三类内容；用户端仅可看到已发布内容；多语言、状态控制、隐私保护和 Markdown 安全渲染有效。

#### 2.4 定价套餐管理与用户端展示
- [ ] 管理员可维护套餐 SKU、名称、展示价格、原价、币种、权益、报告/积分数量、排序、启用状态与四语文案。
- [ ] Pricing 页面改为读取公共价格接口；支付尚未开放时不得制造虚假支付成功或扣款流程。
- [ ] 价格由服务端按 SKU 定义；未来订单须保存下单时的价格快照，后续改价不得改变历史订单。
- [ ] 提供后台价格 CRUD/启停接口与 `GET /api/v1/public/pricing` 公共只读接口。
- **验收**：后台改动套餐后用户端展示同步更新；禁用套餐不对用户端展示；客户端不能自行提交或信任金额。

#### 2.5 出生档案与报告资料联动
- [ ] Google 首次登录只创建普通用户，不自动创建出生档案，也不得在后续登录覆盖用户手工修改的资料。
- [ ] 报告表单独立校验报告必填出生信息；缺少报告必填信息时不能生成报告，但用户无需先创建个人档案。
- [ ] 当报告资料完整、但用户无完整档案时，提供可选保存动作：仅生成本次报告、保存/补充到明确选定的已有档案、另存为新档案。选择不保存时报告仍须继续。
- [ ] 用户档案完整时不弹出联动提示；替他人测算时不得自动覆盖用户自己的档案，只有明确选择目标档案并确认后才可写入。
- **验收**：无档案用户可生成资料完整的报告；拒绝保存不阻塞报告；替他人测算不会误覆盖本人档案；完整档案用户不出现冗余提示。

#### 2.6 lunar-go 确定性排盘核心与命盘展示
- [ ] 接入 `lunar-go`,实现 `bazi.Calculate`(第 7 章),输出标准命盘 JSON；排盘必须是确定性计算，严禁由 LLM 生成或修正。
- [ ] 按 `docs/全球出生时间与真太阳时架构规范.md` 实现唯一 `BirthChartEngine`：输入标准化、地点解析、历史时区/DST、当地标准时间、平太阳时、均时差、真太阳时、00:00 换日后再调用 `lunar-go`。
- [ ] 免费排盘与完整报告建立独立前端、API、请求/响应 DTO；两者共用 `BirthChartEngine`、`chart_hash` 与命盘缓存，接口层不得复制核心计算逻辑。
- [ ] 免费排盘必须登录、不扣积分、不限每日次数（仅防刷），不创建报告、不进报告列表、不保存档案；保存独立排盘历史并支持分页、详情、单删和批量删除。
- [ ] 实现干支、五行、十神四语映射表与稳定 `chart_hash` 缓存。
- [ ] 提供命盘计算接口 `POST /api/v1/bazi/chart`（最终路径以第 5 章 API 契约统一为准），输入须完成出生信息、时区和历法校验。
- [ ] 前端实现命盘可视化组件，展示四柱、五行等确定性基础数据，复用既有设计系统；不把命盘展示伪装成完整解读报告。
- [ ] 排盘与后续解读严格分离：命盘结构和 `chart_hash` 可被快速测算与完整报告复用，不能由不同链路重复自行排盘。日志仅记录 `chart_hash`、脱敏摘要和 trace_id，不记录完整出生资料或完整命盘。
- **验收**：输入出生信息可得到与已知 case 对拍一致的四柱、五行、十神和大运；全球地点按出生当时历史时区/DST换算真太阳时；真太阳时自然跨日后按 00:00 换日；免费接口与完整报告接口对相同输入返回相同 `chart_hash` 与四柱；相同输入结果稳定；LLM 不参与排盘。

### Phase 3 — 深度解读基础数据与追溯
- [ ] B0：固定 `InterpretationFacts`、报告事实快照、LLM调用记录、版本与管理端只读接口契约。
- [ ] 建立版本化命理基础数据中心 `bazi-base-data-v1`，统一提供五行、天干、地支藏干、十神映射、干支关系、月令矩阵、位置权重和旺衰阈值；固定事实不进入业务数据库。管理端通过 Admin Token 只读查询、搜索、筛选和分页，不提供增删改。
- [x] 建立公共干支日历 `annual_calendar_years`：从运行年份起保存连续 60 年的年份、干支、五行、阴阳、生肖和六十甲子序号；报告与计算档案只截取计算当年起连续 10 年，再结合个人年龄和所属大运生成 `annual_fortunes`。公共数据与个人化结果分层，10 年结果及其范围说明必须固化进不可变快照。
- [ ] 隐藏简单测算可见入口；保留Quick代码、接口和表，后续以完整事实包裁剪版恢复。
- [ ] 身强身弱采用独立、确定性、版本化规则引擎；V1 规则及原始稿见 `docs/rules/八字身强身弱判定算法-v1.md` 和 `docs/rules/八字身强身弱判定算法-v1-原始稿.md`。输出必须包含月令、逐项贡献、根气、合冲刑害、从格与规则版本；本模块不推导喜用神。
- [ ] 将身强身弱完整明细写入 `InterpretationFacts.day_master_strength.analysis`；干支关系从该明细投影，禁止二次计算。`facts_hash` 必须包含强弱规则版本，但不受 Prompt 版本变化影响。
- [ ] 审计并补齐五行力量、十神结构、喜忌、大运与流年等其余确定性事实；干支关系由身强身弱规则引擎先行输出证据，后续事实包只能复用，不得重复计算。
  - [x] 十神结构 V1：按显干、藏干和位置权重计算十神原始力量，聚合比劫、印星、食伤、财星、官杀五类，复用身强身弱引擎的关系证据形成类别调整，并写入版本化事实快照。
  - [x] V2-A 五行力量：按 `bazi-power-v1.0` 输出原始、季节修正和基础生克后的有效力量，保存贡献、根气、交互、节气进度与版本证据；历史版本继续保留用于快照兼容。
  - [x] V2-B 结构交互与身强弱：按 `bazi-power-v2.0` 接入合会冲刑害破、合化置信度与有效力量；按 `strength-v2.0` 输出连续强弱分、得令得地得势、置信度与格局候选；按 `ten-god-effective-v2.0` 输出保留原始阴阳证据的有效十神。实现边界见 `docs/rules/五行结构交互与身强弱-v2-B实现规范.md`。
  - [x] 身强弱唯一结论：当前规则流水线只输出一套正式 `strength.level/score`；早期阶段产生的位置、藏干和关系贡献仅作为上游计算依据，不得作为第二套结论。已完成报告快照保持不可变，历史字段只在读取时兼容。
  - [x] V2-C 调候、格局、病药与通关：四个独立、版本化确定性模块消费 V2-B 结果，保存候选、否决原因、严重度、通关状态、规则证据及待校准警告；结果进入命盘快照、报告事实包和哈希。实现边界见 `docs/rules/调候格局病药通关-v2-C实现规范.md`。
  - [x] V2-D 喜用神：按 `useful-god-v1.1` 对五行全部候选执行静态评分、四档 What-If 重算、边际效用、副作用和硬否决；候选模拟与静态评分共用同一组动态权重，输出第一/第二用神及喜忌仇闲角色；完整过程进入命盘快照、事实包和哈希。实现边界见 `docs/rules/喜用神-v2-D实现规范.md`。
  - [ ] 喜用神：采用 `docs/rules/八字五行力量身强身弱与喜用神算法-v2.0-原始稿.md`，作为独立确定性模块按 V2-B 至 V2-D 依赖顺序实现，不得由十神结构或 LLM 猜测生成。
- [ ] 实现事实包编排、规范JSON、`facts_hash`、规则版本、章节事实白名单与隐私校验。
- [ ] 保存每份报告不可变事实快照，并为管理端提供只读追溯能力。
- **验收**：关闭LLM仍可生成完整事实包；相同输入结果及哈希稳定；每个判断可定位规则和依据；历史快照不受升级影响。

### Phase 4 — 完整测算(异步 + DeepSeek + PDF)
> 执行状态、分块验收证据与完成记录统一维护在 `docs/完整报告执行计划与验收台账.md`。开始和完成任何完整报告任务前后必须同步更新该台账，不得仅凭口头进度判断完成状态。

- [ ] `JobQueue` 接口 + goroutine 实现(worker pool);report 状态机经队列驱动(第 8 章)。
- [ ] Full prompt 按十章注册表分章编排并生成严格 JSON；未来十年流年可拆分调用后合并。
- [ ] 每次调用和重试均保存实际Prompt、事实输入、原始/解析输出、Schema结果、token、耗时、trace_id和版本。
- [ ] 正式报告按1～10可配置并发执行十章；执行前自动预检并冻结事实、十章指令、术语、输出协议、模型路由、并发、超时与重试策略。
- [ ] 实现可扩展章节校验器链与报告汇总校验；失败由系统自动追加attempt并按有序Provider/模型路由兜底，禁止管理端人工重跑单章。
- [ ] 报告、执行快照、章节、调用尝试、最终结果采用元数据/payload拆表；列表游标分页，大字段详情懒加载，按唯一关联批量查询且禁止N+1。
- [ ] `completed`/`failed`终态冻结并校验facts、execution、content三类哈希；相同八字重复生成必须创建新报告，历史链路不可覆盖。
- [ ] 报告创建时固化保留策略与过期时间；到期任务分批清理整条payload链和PDF，订单支付记录按独立财务策略保留。
- [ ] `Renderer` 出 PDF:`full_report.html`(A4)→ `Storage`(R2)。
- [ ] `Notifier` 接口 + Resend/noop 实现;报告完成发「report_ready」邮件。
- [ ] `POST /readings/full` + 轮询接口;前端进度页 + PDF 预览/下载。
- **验收**：扣积分能生成一份完整10章PDF；十章可配置并行且全部通过程序校验后才交付；网络、结构、遗漏、事实冲突等失败能按冻结策略自动重试和切换后备模型，耗尽后整份失败并按业务规则退积分；终态链路不可修改，同一八字可创建多份独立报告；报告完成有通知。

> **后续保留能力：简单测算**。待完整报告的确定性事实与生成链路稳定后恢复；它只能裁剪 `InterpretationFacts` 和完整报告章节，不得复制排盘、强弱、喜忌、大运或流年逻辑。

### Phase 5 — 支付(多渠道抽象,MVP 接 Stripe)
- [ ] 先落 `payment` 包:`PaymentProvider` 接口 + Registry + Catalog(SKU)。
- [ ] 实现 `StripeProvider`(Checkout Session + ParseWebhook 验签),注册进 Registry。
- [ ] 统一 checkout/webhook handler:按 `:provider` 路由,`payment_events` 去重,订单+积分同事务。
- [ ] 单次报告购买 + 积分包购买 + 积分流水。
- [ ] 前端:`/payments/providers` 动态渲染按钮、按 `action` 跳转/SDK、支付成功页 → 触发完整报告。
- **验收**:能用 $5.99 买一份报告并成功生成;能买积分包并到账;**新增一个 mock provider 仅需实现接口 + 注册,不改业务代码**(验证抽象到位)。

### Phase 6 — 管理端深化(依赖报告与支付完成)
- [ ] 在 Phase 2 管理端基础上，接入真实报告管理：报告列表、详情、状态、失败原因、耗时、事实快照、十章执行、系统自动尝试、校验结果与最终内容；管理端只读追溯，不提供单章重试、输出替换或终态修改操作。
- [ ] 接入订单、支付事件、积分流水与套餐历史价格快照，支持与 PaymentProvider 一致的退款/对账等操作。
- [ ] 实现 Dashboard 聚合接口：今日营收、订单、新增用户、报告成功率、支付渠道占比与趋势。
- [ ] 视真实运营需求再启用资源 Registry、查询 DSL、Schema 驱动列表/详情等通用化能力；不得为了抽象而牺牲已确认的管理端体验。
- [ ] 保留全部管理员权限的当前策略；若未来需要多人协作，再在既有 `admin_users`、审计日志基础上启用 RBAC。
- **验收**：管理端报告数据与用户端真实报告一致；订单/支付/积分数据可追溯；所有后台写操作保留审计记录。

### Phase 7 — 打磨与上线
- [ ] 隐私政策/服务条款/联系页。
- [ ] SEO(metadata、sitemap、多语言 hreflang)。
- [ ] 错误监控、日志、限流。
- [ ] 部署:Vercel + VPS + R2 + 域名 + HTTPS。
- **验收**:线上可用,海外访问流畅,四语完整,全链路跑通。

---

## 18. 验收清单(总)

**核心原则合规**
- [ ] P1:排盘全程无 LLM 参与,结果确定可复现。
- [ ] P2:前端四语 + 所有对外文案 + 渲染模板,**零 "AI" 字眼**。
- [ ] P3:LLM 全部返回结构化 JSON,渲染稳定。
- [ ] P4:完整报告异步生成,状态机完整,失败可恢复。
- [ ] P5:LLM 通过接口抽象,可切换 provider。

**可扩展性(接口抽象)**
- [ ] 九大能力均经接口抽象(LLM/支付/登录/渲染/异步任务/通知/存储/缓存/后台资源)。
- [ ] 所有具体实现在 `main.go` 依赖注入,业务层(service)不依赖任何具体 SDK。
- [ ] 抽样验证:为「支付/登录/后台资源」各加一个 mock 实现,仅需「实现接口 + 注册」,业务代码零改动。
- [ ] `.env` 可通过 `*_PROVIDER` / `*_PROVIDERS` 切换实现,无需改代码。

**功能**
- [ ] 四语(en/zh/ja/ko)全站可切换且文案完整。
- [ ] Google 登录 + 单设备登录。
- [ ] 出生档案 CRUD(公历/农历)。
- [ ] 简单测算:每日 3 次免费,出图。
- [ ] 完整测算：$5.99 或 10 积分，输出 10 章 PDF。
- [ ] 多渠道支付(Stripe MVP)+ Webhook 验签/幂等 + 积分;新增渠道零重构。
- [ ] 后台管理:登录 + RBAC + 审计;6 类资源 CRUD;新增资源前后端零改动(结构化框架验证)。

**质量/部署**
- [ ] 落地页与 `bazi-landing-final.html` 视觉 1:1。
- [ ] 设计 token 全站统一(第 14 章)。
- [ ] 海外 CDN 访问正常。
- [ ] 隐私/条款页齐全。

---

## 19. 后台管理系统(Admin)

> **设计目标**:运营/客服用的内部后台,与 C 端**物理隔离**(独立路由前缀、独立鉴权、独立账号体系、独立 RBAC、全程审计)。后台开发**全部结构化、接口化**——把"对一类资源做增删改查"抽象成统一的 `Resource` 模式,新增一个管理模块 = 写一个 `xxx_resource.go` + 注册,**路由、列表、筛选、详情、编辑前后端全部自动生成**,杜绝重复 CRUD 代码,极易扩展。

### 19.1 隔离原则(硬性)

| 维度 | C 端 | 后台 |
|---|---|---|
| 路由前缀 | `/api/v1/*` | `/admin/api/v1/*` |
| 账号体系 | `users`(Google OAuth) | `admin_users`(账号密码 + 可选 TOTP) |
| 鉴权 | 用户 JWT | **独立** Admin JWT(独立密钥 `ADMIN_JWT_SECRET`) |
| 授权 | 无(普通用户) | **RBAC**:角色 → 权限码 |
| 审计 | 无 | **所有写操作落 `admin_audit_log`** |
| 部署 | `fatelumen.com` | 建议子域 `admin.fatelumen.com` + IP 白名单/Basic 兜底 |

后台代码全部放在 `internal/admin/`,**不允许** C 端 handler 依赖 admin 包,反向只读复用 `service`/`repository`/`model`。

### 19.2 核心抽象 — `Resource` 接口

后台对每类数据(用户/订单/报告/积分/商品…)的操作,统一抽象成 `Resource`。框架据此**自动注册路由**,并向前端**暴露 Schema 元数据**驱动表格/表单渲染。

```go
package resource

// 统一查询参数(列表接口),由框架从 query string 解析
type ListQuery struct {
    Page     int                    // 页码，从 1 起
    PageSize int                    // 每页条数，默认 20，上限 100
    Sort     string                 // 排序字段，前缀 - 表示倒序，如 "-created_at"
    Search   string                 // 全局关键词(命中资源声明的 searchable 字段)
    Filters  map[string]interface{} // 字段精确/范围筛选，如 {"status":"paid","created_at__gte":"2026-01-01"}
}

type ListResult struct {
    Items interface{} `json:"items"`
    Total int64       `json:"total"`
    Page  int         `json:"page"`
    PageSize int      `json:"page_size"`
}

// 自定义操作(超出标准 CRUD 的业务动作)
type Action struct {
    Name    string // "refund" / "retry" / "ban"
    Label   string // 前端按钮文案(后台内部用，可中文)
    Perm    string // 需要的权限码，如 "order:refund"
    Handler func(ctx *AdminContext, id string, params map[string]interface{}) (interface{}, error)
}

// 每个业务资源实现此接口
type Resource interface {
    // 资源标识(= 路由路径 + 权限前缀)，如 "orders"
    Name() string
    // 字段 Schema：驱动前端列表列/筛选器/表单，并约束后端筛选/排序白名单
    Schema() []Field
    // 标准能力(返回 nil 表示该资源不支持该操作)
    List(ctx *AdminContext, q ListQuery) (*ListResult, error)
    Detail(ctx *AdminContext, id string) (interface{}, error)
    Update(ctx *AdminContext, id string, patch map[string]interface{}) (interface{}, error)
    // 自定义动作列表(退款、重试、封禁…)
    Actions() []Action
}
```

> **创建(Create)** 多数后台资源不需要(用户/订单都是 C 端产生的),故不放进必选接口;少数需要(如「手动加积分」「新建商品」)通过 `Actions()` 或单独实现,保持接口最小化。

### 19.3 字段 Schema(数据驱动前端)

`Schema()` 是整套结构化的关键——它同时:① 告诉前端**怎么渲染表格列和筛选器**;② 约束后端**哪些字段可筛、可排、可改**(白名单,防注入/越权改字段)。

```go
type Field struct {
    Key        string   // 字段名(对应 DB 列 / JSON key)
    Label      string   // 列名/表单标签
    Type       string   // string/int/money/datetime/enum/bool/json/relation
    Enum       []EnumOption // Type=enum 时的可选值(如订单状态)
    Sortable   bool     // 是否可排序
    Filterable bool     // 是否可筛选
    Searchable bool     // 是否计入全局搜索
    Editable   bool     // 详情页是否可编辑(进 Update 白名单)
    Hidden     bool     // 列表是否默认隐藏(详情仍展示)
}
```

示例(`order_resource.go` 的 Schema 节选):

```go
func (r *OrderResource) Schema() []resource.Field {
    return []resource.Field{
        {Key: "id", Label: "ID", Type: "int", Sortable: true},
        {Key: "user_id", Label: "用户", Type: "relation", Filterable: true},
        {Key: "sku", Label: "商品", Type: "string", Filterable: true},
        {Key: "amount_cents", Label: "金额", Type: "money", Sortable: true},
        {Key: "provider", Label: "渠道", Type: "enum", Filterable: true,
            Enum: []resource.EnumOption{{"stripe","Stripe"},{"paypal","PayPal"}}},
        {Key: "status", Label: "状态", Type: "enum", Filterable: true, Sortable: true,
            Enum: []resource.EnumOption{{"created","待支付"},{"paid","已支付"},{"refunded","已退款"}}},
        {Key: "created_at", Label: "创建时间", Type: "datetime", Sortable: true, Filterable: true},
    }
}
```

### 19.4 注册中心 — 自动路由 + 菜单

```go
type Registry struct{ resources map[string]Resource }

func (r *Registry) Register(res Resource) { r.resources[res.Name()] = res }

// 启动时遍历注册的资源，自动挂载标准 REST 路由：
//   GET    /admin/api/v1/{name}            -> List
//   GET    /admin/api/v1/{name}/:id        -> Detail
//   PATCH  /admin/api/v1/{name}/:id        -> Update
//   POST   /admin/api/v1/{name}/:id/actions/:action -> 自定义 Action
//   GET    /admin/api/v1/{name}/schema     -> 返回 Schema(前端渲染用)
// 每条路由自动套：AdminAuth -> RBAC(权限码 = name:read / name:write / name:{action}) -> Audit
func (r *Registry) Mount(g *gin.RouterGroup) { /* ... */ }

// 菜单元数据：前端侧边栏直接读这个，自动生成
func (r *Registry) Menu(perms []string) []MenuItem { /* 过滤无权限的 */ }
```

**新增一个管理模块的完整成本**:写一个 `internal/admin/resources/xxx_resource.go` 实现 `Resource` 接口 → `registry.Register(NewXxxResource(repo))`。路由、列表分页筛选排序、详情、编辑、权限校验、审计、前端表格表单**全部自动具备**。

### 19.5 统一查询 DSL

列表接口的 query string 统一格式,框架解析为 `ListQuery`,再由通用 GORM 构造器按 Schema 白名单生成查询:

```
GET /admin/api/v1/orders?page=1&page_size=20&sort=-created_at
    &status=paid&provider=stripe
    &created_at__gte=2026-06-01&created_at__lt=2026-07-01
    &search=alice
```
- 筛选后缀:`__gte / __lte / __gt / __lt / __like / __in`(无后缀 = 精确等于)。
- **只有 Schema 里 `Filterable=true` 的字段允许筛选**,`Sortable=true` 才允许排序,否则忽略(防止任意字段被探测/排序)。
- `search` 仅命中 `Searchable=true` 的字段(OR LIKE)。

### 19.6 RBAC 与审计

- **角色三档**(可扩展):`super_admin`(`["*"]`)、`operator`(读 + 退款/重试)、`viewer`(只读)。
- 权限码规则:`{resource}:read` / `{resource}:write` / `{resource}:{action}`(如 `order:refund`)。
- 中间件链:`AdminAuth`(校验 Admin JWT)→ `RBAC`(比对角色权限码)→ `Audit`(写操作自动落 `admin_audit_log`,记录 admin、动作、资源、前后值快照、IP)。
- 审计**自动化**:无需各 Resource 手写,框架在 Update/Action 成功后统一落库。

### 19.7 内置资源与动作(MVP)

| 资源 | 列表/详情 | 可编辑字段 | 自定义动作 |
|---|---|---|---|
| `users` | ✅ | status(封禁/解封) | `ban` / `unban` / `grant_credits`(手动加积分,走 credit_service 事务) |
| `orders` | ✅ | — | `refund`(调对应 PaymentProvider.Refund + 写反向 ledger) |
| `reports` | ✅(状态/耗时/错误) | — | `retry`(失败报告重入异步队列) |
| `readings` | ✅ | — | — |
| `credit_ledger` | ✅(只读流水) | — | — |
| `catalog` | ✅(商品/价格) | amount_cents / credits / enabled | — (改完即时生效,读写 payment.Catalog) |

### 19.8 Dashboard 聚合

`GET /admin/api/v1/dashboard?range=today|7d|30d` 返回:今日营收、订单数、新增用户、报告成功率、各渠道占比、近 N 日趋势(供前端画图)。聚合查询集中在 `admin/dashboard/dashboard.go`,直接读 MySQL(MVP 数据量小,无需数仓)。

### 19.9 后台前端(数据驱动)

> 后台前端可与 C 端同仓(`apps/admin`)或独立小项目,技术栈沿用 **Next.js + TS + Tailwind**(无需 next-intl,后台仅中文)。**强烈建议数据驱动**:页面通用,靠资源 Schema 渲染。

- **通用列表页**:读 `/{name}/schema` → 自动渲染表格列、筛选器、排序;读 `/{name}` 取数据;翻页/筛选回传 DSL。
- **通用详情/编辑页**:按 Schema 的 `Editable` 字段生成表单,PATCH 提交;`Actions` 渲染成操作按钮。
- **侧边栏**:读 `Registry.Menu()`,按当前管理员权限自动显示可见模块。
- 结果:**新增资源时前端零改动**(除非要定制特殊视图),与后端的"结构化"一一对应。

### 19.10 安全红线

- Admin 与 C 端 **JWT 密钥分离**,后台 token 短有效期(如 2h)+ 刷新。
- 后台默认部署在独立子域 + **网络层访问控制**(IP 白名单 / VPN / Cloudflare Access),不暴露公网裸跑。
- 所有写操作**强制 RBAC + 审计**,退款/封禁/改价等高危动作需 `write` 级权限。
- Update 只接受 Schema `Editable=true` 的字段(后端二次过滤,绝不信前端字段集)。
- 初始 `super_admin` 账号通过迁移脚本/CLI 创建,密码 bcrypt,严禁硬编码。

---

## 附录 A — 交付物清单(随本任务书一起)
| 文件 | 说明 |
|---|---|
| `FateLumen-开发任务书.md` | 本文档(唯一总纲) |
| `bazi-landing-final.html` | 落地页视觉与文案唯一真源(含 en/zh/ja/ko 四语字典),前端 1:1 还原 |

## 附录 B — 给开发者的执行提示

1. **严格按 Phase 顺序**,每个 Phase 自测通过再进下一个。
2. **落地页直接用交付的 HTML**,不要自己重新设计;只做「HTML → React 组件 + i18n 抽取」。
3. **四语文案直接从 HTML 的 I18N 字典搬到 locales/*.json**,不要重新翻译。
4. **排盘务必用 lunar-go**,不要自己写干支算法,更不要用 LLM 算。
5. **支付与后台都要"接口化优先"**:支付新增渠道 = 实现 `PaymentProvider`;后台新增模块 = 实现 `Resource`。先把框架(接口 + Registry)搭好再写具体实现。
6. 文档未覆盖处:遵循「主流最佳实践 + MVP 最简」,代码里用 `// DECISION:` 注释说明。
7. 安全红线:金额后端定义、Webhook 验签、JWT 校验 token_id、SQL 用参数化(GORM 默认)、后台 RBAC + 审计 + 子域隔离。

— 文档结束 —
