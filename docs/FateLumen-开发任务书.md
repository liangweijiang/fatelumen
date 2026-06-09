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
FateLumen 是一个面向**海外市场**的八字(四柱命理)在线测算网站：用户输入出生信息 → 系统**用确定性算法精确排盘** → 生成清晰、专业、易读的命理解读(简单测算出图，完整测算导出 PDF)。

### 1.2 商业模式(MVP)
- **简单测算(Quick Reading)**：免费，每日 3 次额度，输出一张可分享的图片。
- **完整测算(Full Reading)**：付费 **$5.99 / 次** 或消耗 **10 积分**，输出一份精美 PDF 报告(12 章)。
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
   │      ├── Quick: 排盘 → LLM(1次) → Renderer 出图 → Storage
   │      └── Full : 排盘 → JobQueue 异步任务
   │                    └── LLM(分批多次) → 组装 → Renderer 出 PDF → Storage
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

**简单测算(Quick，同步)**
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
  - `lunar-go` 排盘
  - 分 3–4 批调 LLM，生成 12 章结构化 JSON
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
  birth_place   VARCHAR(128) COMMENT '出生地(用于真太阳时/时区)',
  timezone      VARCHAR(48)  COMMENT '如 Asia/Shanghai',
  longitude     DECIMAL(9,6) COMMENT '经度，用于真太阳时校正(可选)',
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
  pay_method    VARCHAR(16)  NOT NULL COMMENT 'order(付费订单) / credit(扣积分)',
  content       JSON         COMMENT '12章完整报告结构化 JSON',
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

### 4.4 完整报告 JSON 结构(`reports.content`，12 章)

```json
{
  "locale": "en",
  "summary_line": "一句话命局总结",
  "chapters": [
    {"no": 1,  "key": "structure",   "title": "...", "body": "...", "tags": []},
    {"no": 2,  "key": "day_master",  "title": "...", "body": "...", "strength_score": 38},
    {"no": 3,  "key": "personality", "title": "...", "body": "..."},
    {"no": 4,  "key": "career",      "title": "...", "body": "..."},
    {"no": 5,  "key": "wealth",      "title": "...", "body": "..."},
    {"no": 6,  "key": "marriage",    "title": "...", "body": "..."},
    {"no": 7,  "key": "health",      "title": "...", "body": "..."},
    {"no": 8,  "key": "luck_cycles", "title": "...", "body": "...", "cycles": []},
    {"no": 9,  "key": "yearly",      "title": "...", "body": "...", "years": []},
    {"no": 10, "key": "elements_advice", "title": "...", "body": "..."},
    {"no": 11, "key": "relationships",   "title": "...", "body": "..."},
    {"no": 12, "key": "lifetime_summary","title": "...", "body": "..."}
  ]
}
```

**12 章 key 全集(渲染模板与 Prompt 必须严格对齐此表,不得增删改 key):**

| no | key | 含义 | 额外字段 | 生成批次 |
|---|---|---|---|---|
| 1 | `structure` | 格局结构 | — | Batch 1 |
| 2 | `day_master` | 日主强弱 | `strength_score`(int 0–100) | Batch 1 |
| 3 | `personality` | 性格特质 | — | Batch 1 |
| 4 | `career` | 事业 | — | Batch 2 |
| 5 | `wealth` | 财运 | — | Batch 2 |
| 6 | `marriage` | 婚姻感情 | — | Batch 2 |
| 7 | `health` | 健康 | — | Batch 3 |
| 8 | `luck_cycles` | 大运 | `cycles`:[{`ganzhi`,`start_age`,`start_year`,`note`}] | Batch 3 |
| 9 | `yearly` | 流年(近 5–10 年) | `years`:[{`year`,`ganzhi`,`note`}] | Batch 3 |
| 10 | `elements_advice` | 五行调理建议 | — | Batch 4 |
| 11 | `relationships` | 人际关系 | — | Batch 4 |
| 12 | `lifetime_summary` | 终身总结 | — | Batch 4 |

> 公共字段:每章必含 `no`(int)、`key`(string)、`title`(本地化标题)、`body`(180–320 词正文)。`cycles`/`years` 数组的数据**来自命盘 `chart_data` 的真实大运/流年**(LLM 只加 `note` 解读,不得编造年份/干支)。**渲染模板按 `key` 渲染各章,前端/PDF 与此表一一对应。**

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
| POST | `/api/v1/readings/full` | 创建完整报告任务(需已支付 or 扣积分)，返回 `report_id` |
| GET | `/api/v1/readings/full/{report_id}` | 轮询状态；`done` 时返回 `pdf_url` + `content` |

**`POST /api/v1/readings/full` 请求体：** `{"profile_id":123,"locale":"en","pay_method":"credit"}`
- `pay_method=credit`：直接扣 10 积分(不足返回 `code=4020 insufficient credits`)
- `pay_method=order`：要求 `order_id`,校验该订单 `status=paid` 且未被其它报告占用(渠道无关)
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

### 5.7 业务错误码表

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
│   │   ├── report.go            # 含 ReportContent 结构体(12章 JSON)
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
func WithMaxTokens(n int) Option
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

// Input 归一化的排盘输入
type Input struct {
    Gender       int    // 0 female 1 male
    CalendarType int    // 0 solar 1 lunar
    Year, Month, Day, Hour, Minute int
    IsLeapMonth  bool
    Timezone     string
    Longitude    float64 // 可选：真太阳时校正
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

`chart_hash = sha256(normalize(gender|calendar_type|year|month|day|hour|minute|is_leap_month|timezone))`，命中 `charts` 表即复用，避免重复计算。

### 7.5 calculator.go 参考实现(可直接编译,DeepSeek 照此填充)

> 以下是基于 `github.com/6tail/lunar-go` 的**完整参考实现**。lunar-go 的包路径为 `github.com/6tail/lunar-go/calendar`。**严格按此调用,不要自己发明 API。**

```go
// internal/bazi/calculator.go
package bazi

import (
    "fmt"
    "github.com/6tail/lunar-go/calendar"
    "yourmod/internal/model"
)

// Calculate 纯函数：出生信息 → 确定性命盘。无网络/无 LLM，同输入同输出。
func Calculate(in Input) (*model.ChartData, error) {
    var lunar *calendar.Lunar

    // 1. 公历/农历 → Lunar 对象
    if in.CalendarType == 0 { // solar 公历
        if in.Hour < 0 { // 时辰未知，按 12:00 占位并打标记
            solar := calendar.NewSolarFromYmdHms(in.Year, in.Month, in.Day, 12, 0, 0)
            lunar = solar.GetLunar()
        } else {
            solar := calendar.NewSolarFromYmdHms(in.Year, in.Month, in.Day, in.Hour, in.Minute, 0)
            lunar = solar.GetLunar()
        }
    } else { // lunar 农历（支持闰月：月份传负数表示闰月）
        month := in.Month
        if in.IsLeapMonth {
            month = -in.Month
        }
        h := in.Hour
        if h < 0 {
            h = 12
        }
        lunar = calendar.NewLunarFromYmdHms(in.Year, month, in.Day, h, in.Minute, 0)
    }

    // 2. 取八字（四柱）
    ec := lunar.GetEightChar()
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
        HourUnknown:  in.Hour < 0,  // 时辰不明标记，前端/解读据此弱化时柱
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
    // 1990-08-15 14:30 公历 男（北京时间）
    in := Input{Gender: 1, CalendarType: 0, Year: 1990, Month: 8, Day: 15, Hour: 14, Minute: 30}
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
    in := Input{Gender: 0, CalendarType: 0, Year: 2000, Month: 1, Day: 1, Hour: 0, Minute: 0}
    a, _ := Calculate(in)
    b, _ := Calculate(in)
    assert.Equal(t, a, b)
}
```

> **验证方法**:Case1 的期望四柱可用任意权威排盘工具核对(同一生辰任何正确实现都应得出相同四柱)。若 DeepSeek 跑出来与期望不符,说明 lunar-go 调用有误(常见错误:公历农历传反、闰月符号、时辰边界),必须修到通过为止。

---

## 8. 报告生成模块(LLM + 异步状态机)

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
- 失败自动重试**最多 3 次**(retry_count)，超过标记 failed 并退还积分(若用积分)。
- **超时控制**：单个 LLM 调用用 `context.WithTimeout`(如 60s);整个报告任务总超时(如 5min)。
- 报告 `done` 后调 `Notifier.Send`(模板 `report_ready`,按用户 locale)通知用户;PDF/图片经 `Renderer` 生成、`Storage` 上传。

### 8.2 分批生成策略(降低单次失败影响 + 控制 token)

12 章分 **4 批** 并发/串行调用 LLM(建议串行，避免触发限流):

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

### 9.3 完整测算 Prompt(Full，分 4 批)

**每批 User prompt 模板：**
```
locale: {{locale}}
chart: {{chart_json}}
chapters_to_write: {{batch_chapter_keys}}   // e.g. ["structure","day_master","personality"]

Task: For EACH requested chapter, write a detailed interpretation (180-320 words each)
grounded in the chart facts. Return STRICT JSON:
{
  "chapters": [
    {"key":"structure","title":"...","body":"..."},
    {"key":"day_master","title":"...","body":"...","strength_score": <int from chart>},
    ...
  ]
}
Constraints:
- title localized to {{locale}}.
- body references concrete chart details (pillars, elements count, strength, ten-gods, luck cycle years).
- For "luck_cycles"/"yearly" chapters, include a "cycles"/"years" array echoing the chart's actual cycle/year data with short per-period notes.
```

### 9.4 调用参数建议

| 参数 | 值 |
|---|---|
| model | `deepseek-chat`(默认,成本优先) / `gpt-4o-mini` / `gpt-4o`(质量优先),配置可切 |
| temperature | 0.7(解读需要文采，但别太发散) |
| response_format | `{"type":"json_object"}`(DeepSeek 与 OpenAI 同协议,均支持 JSON 模式) |
| max_tokens | quick ~600;full 每批 ~2000 |

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

1. Go template 把 12 章 `report.content` 填入 `full_report.html`(A4 排版,封面+目录+章节)。
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

> chromedp `PrintToPDF`(A4 / printBackground=true)。封面 + 目录 + 12 章循环。变量来自 `report.content`(4.4 的 12 章结构)。

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
  <!-- 12 章正文：按 key 渲染，luck_cycles/yearly 额外渲染表格 -->
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
    "report_single": {Type: "report",  AmountCents: 599,  Currency: "usd", Credits: 0},
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
  - `payment_succeeded` → 订单置 `paid`;若 `type=credits` 写 `credit_ledger` + 加 `users.credits`(同一事务)。
  - 写入 `payment_events` 标记已处理。
7. 前端支付成功页轮询订单状态,paid 后调 `POST /readings/full {order_id}` 触发报告生成。

### 11.5 安全与一致性红线

- **金额/商品在后端按 sku 定义**,前端禁传金额(防篡改)。
- **每个渠道独立验签**(Stripe 验 `Stripe-Signature`;PayPal 验 webhook signature;Paddle 验 HMAC)——验签在各 provider 的 `ParseWebhook` 内完成,业务层拿到的事件已可信。
- **Webhook 幂等**:`payment_events(provider,event_id)` 唯一键 + 订单状态双重保护,重复回调只处理一次。
- **下单幂等**:`orders(provider,provider_ref)` 唯一键,防同一渠道会话重复落单。
- 订单与发积分**同事务**,失败回滚。
- 退款/报告失败用积分时,自动写反向 `credit_ledger`。
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

### Phase 2 — 排盘(核心 P1)
- [ ] 接入 `lunar-go`,实现 `bazi.Calculate`(第 7 章),输出标准命盘 JSON。
- [ ] 干支/五行/十神 四语映射表。
- [ ] `POST /charts`(含 chart_hash 缓存)。
- [ ] 前端:命盘可视化组件(四柱 + 五行色点,复用落地页样式)。
- **验收**:输入出生信息能得到正确四柱/五行/十神/大运(用已知 case 对拍验证)。

### Phase 3 — 简单测算(同步出图)
- [ ] LLMProvider 接口 + **DeepSeek 实现**(默认,JSON mode);OpenAI 实现作为可切备选。
- [ ] Quick prompt(第 9.2),生成简短解读 JSON。
- [ ] `Renderer` 接口 + chromedp 实现;`quick_image.html` 模板截图 → `Storage`(R2)。
- [ ] 每日免费额度(3 次/天,Redis 或 daily_quota 表)。
- [ ] `POST /readings/quick` 全链路;前端结果页展示图片 + 分享。
- **验收**:免费用户每天 3 次,能拿到一张带命盘的解读图。

### Phase 4 — 完整测算(异步 + PDF)
- [ ] `JobQueue` 接口 + goroutine 实现(worker pool);report 状态机经队列驱动(第 8 章)。
- [ ] Full prompt 分 4 批(第 9.3),组装 12 章 JSON。
- [ ] `Renderer` 出 PDF:`full_report.html`(A4)→ `Storage`(R2)。
- [ ] `Notifier` 接口 + Resend/noop 实现;报告完成发「report_ready」邮件。
- [ ] `POST /readings/full` + 轮询接口;前端进度页 + PDF 预览/下载。
- **验收**:扣积分能生成一份完整 12 章 PDF;失败能重试/退积分;报告完成有通知。

### Phase 5 — 支付(多渠道抽象,MVP 接 Stripe)
- [ ] 先落 `payment` 包:`PaymentProvider` 接口 + Registry + Catalog(SKU)。
- [ ] 实现 `StripeProvider`(Checkout Session + ParseWebhook 验签),注册进 Registry。
- [ ] 统一 checkout/webhook handler:按 `:provider` 路由,`payment_events` 去重,订单+积分同事务。
- [ ] 单次报告购买 + 积分包购买 + 积分流水。
- [ ] 前端:`/payments/providers` 动态渲染按钮、按 `action` 跳转/SDK、支付成功页 → 触发完整报告。
- **验收**:能用 $5.99 买一份报告并成功生成;能买积分包并到账;**新增一个 mock provider 仅需实现接口 + 注册,不改业务代码**(验证抽象到位)。

### Phase 6 — 后台管理系统(Admin)
- [ ] 先落 `admin/resource` 框架:`Resource` 接口 + Registry + 查询 DSL + Schema(见第 19 章)。
- [ ] Admin 鉴权:账号密码登录 + 独立 JWT + RBAC 中间件 + 审计日志中间件。
- [ ] 注册 6 个资源:user / order / report / reading / credit / catalog,自动生成路由。
- [ ] Dashboard 聚合接口(今日营收/订单/新增用户/报告成功率)。
- [ ] 后台前端:基于资源 Schema 自动渲染列表表格 + 筛选 + 详情/编辑(数据驱动,新增资源前端零改动)。
- **验收**:能登录后台查/改用户、退款订单、重试失败报告、改商品价格;**新增一个资源仅需写一个 `xxx_resource.go` + 注册,前后端均零改动**。

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
- [ ] 完整测算:$5.99 或 10 积分,出 12 章 PDF。
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
