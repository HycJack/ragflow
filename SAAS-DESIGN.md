# RAGFlow 风格 SaaS 平台设计 —— 用户服务 / 前端界面 / 后台管理

> 本文档参考 RAGFlow 现有架构（`internal/` Go 后端 + `web/` React 前端），设计一个**真正的 SaaS 化平台**。
> RAGFlow 本身更接近"可控的开源私有部署"，因此我们要做的不是重复它的代码，而是**把它可复用的模式抽出来**，补齐 SaaS 特有的能力：多租户、注册/邀请、订阅计费、配额与计量、以及面向平台运营方的管理后台。

---

## 1. 设计目标与核心差异

### 1.1 从 RAGFlow 继承的成熟模式

| RAGFlow 现有实现 | 我们 SaaS 化的复用方式 |
|---|---|
| `entity.User` / `entity.Tenant` / `entity.UserTenant`（多对多，含 `role`、`invited_by`） | 演进为 **Organization(组织) + Membership(成员)** 的多租户模型 |
| `entity.InvitationCode`（邀请码+校验+状态） | 保留并强化，用于"注册白名单/团队邀请" |
| `entity.Tenant.Credit`（额度累计） | 演进为 **Quota(配额/计费)**，绑定订阅计划 |
| `internal/handler` + `internal/service` 分层 | 沿用 `handler → service → dao → entity` 分层 |
| `internal/admin` 独立后台（`/api/v1/admin`） | 沿用，扩展为平台运营后台 |
| `web/`（React + TS + Vite、shadcn/ui、TanStack Query、Zustand、i18n） | 沿用整套前端技术栈与组件约定 |
| JWT + RSA 登录、`auth_via_api_token` | 沿用，并补充组织上下文（当前 workspace）解析 |

### 1.2 SaaS 相比自托管多出来的部分

1. **多租户隔离**：注册即建组织；用户属于多个组织；每个组织独立数据集、Agent、额度。
2. **自助注册与邀请**：邮箱注册、邮箱验证、邀请码/团队邀请、组织切换。
3. **订阅与计费**：免费/专业/企业套餐；按量计费；配额动态调整；账单与用量查询。
4. **配额与计量**：存储、Token 解析、检索/对话调用、成员数，实时扣减与上限。
5. **平台管理后台**：管理用户、组织、订阅、全球用量、服务状态、权限。

---

## 2. 总体架构

```
┌──────────────────────────── 用户侧 (web/src/pages/workspace/*) ────────────────────────────┐
│  登录/注册  组织选择  概览  数据集  智能体  对话  账单与用量  个人设置  配额  邀请成员        │
└───────────────────────────────────────────┬───────────────────────────────────────────────┘
                                            │ (axios + JWT + 当前 org_id)
┌─────────────────────────────── 平台后台 (web/src/pages/admin/*) ────────────────────────────┐
│  登录  用户管理  组织管理  订阅&套餐  计量&用量  服务状态  角色&权限  系统设置                  │
└───────────────────────────────────────────┬───────────────────────────────────────────────┘
                                            │ (独立 Admin JWT)
┌───────────────────────────────────────────┴───────────────────────────────────────────────┐
│                             后端 Go (参考 internal/ 结构)                                   │
│                                                                                           │
│  EntryPoints:  cmd/ragflow_server.go   (--api / --admin / --ingestor / --syncer)           │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐  ┌───────────────┐  ┌──────────────┐  ┌────┐ │
│  │ auth      │  │ user      │  │ org    │  │ subscription │  │ usage        │  │... │ │
│  └─────┬─────┘  └─────┬─────┘  └─────┬─────┘  └───────┬───────┘  └──────┬───────┘  └────┘ │
│        │ handler → service → dao(entity) 分层，handler 只做 HTTP 解析，service 管业务     │
└───────────────────────────────────────────┬───────────────────────────────────────────────┘
                                            │
        MySQL(用户/组织/订阅/元数据)   Kvrocks/JWT(session)   MinIO(文件)   ES/Infinity(检索)  NATS(任务)
```

关键点：
- **两个独立的 Gin 路由组**：`/api/v1/...`（用户侧，走 `AuthMiddleware`）和 `/api/v1/admin/...`（平台后台，走 `AdminAuthMiddleware`），与 RAGFlow 的 `AuthHandler` + `internal/admin/router.go` 一致。
- **组织上下文**：用户侧请求通过 Header `X-Workspace-ID` 指定当前组织，鉴权中间件校验该用户在该组织内的成员角色。
- **配额中间件**：在写操作（上传、解析、检索、对话、建智能体）前统一校验配额，超出返回业务错误码。

---

## 3. 后端【用户服务 / SaaS 服务】设计

### 3.1 数据模型（`internal/entity/`）

> 沿用 RAGFlow 的 `BaseModel`（`create_time / create_date / update_time / update_date`），全部用 `gorm` tag + `BaseModel`。

#### 3.1.1 平台用户（替代 `User`，新增稳定 `external_id` 与安全字段）

```go
type User struct {
    ID             string   `gorm:"column:id;size:32;primaryKey" json:"id"`
    Email          string   `gorm:"column:email;size:255;uniqueIndex:uni_user_email" json:"email"`
    PasswordHash   string   `gorm:"column:password_hash;size:255" json:"-"`
    Nickname       string   `gorm:"column:nickname;size:100;index" json:"nickname"`
    Avatar         *string  `gorm:"column:avatar;type:longtext" json:"avatar,omitempty"`
    Locale         *string  `gorm:"column:locale;size:16;default:'en'" json:"locale,omitempty"`
    Timezone       *string  `gorm:"column:timezone;size:64" json:"timezone,omitempty"`
    EmailVerified  bool     `gorm:"column:email_verified;default:0;index" json:"email_verified"`
    Status         string   `gorm:"column:status;size:1;default:1;index" json:"status"` // 1 active / 0 disabled
    LastLoginAt    *time.Time `gorm:"column:last_login_at;index" json:"last_login_at,omitempty"`
    IsPlatformAdmin bool    `gorm:"column:is_platform_admin;default:0;index" json:"is_platform_admin"` // 是否运营后台管理员
    BaseModel
}
```

#### 3.1.2 组织（对应 RAGFlow `Tenant`，SaaS 化演进）

```go
type Organization struct {
    ID          string   `gorm:"column:id;size:32;primaryKey" json:"id"`
    Name        string   `gorm:"column:name;size:100;index" json:"name"`
    Slug        string   `gorm:"column:slug;size:100;uniqueIndex:uni_org_slug" json:"slug"` // 唯一友好名称
    Avatar      *string  `gorm:"column:avatar;type:longtext" json:"avatar,omitempty"`
    PlanID      *string  `gorm:"column:plan_id;size:32;index" json:"plan_id"`     // 关联订阅计划
    Status      string   `gorm:"column:status;size:1;default:1;index" json:"status"` // 1 active / 0 suspended
    OwnerUserID string   `gorm:"column:owner_user_id;size:32;index" json:"owner_user_id"`
    BaseModel
}
// TableName: "organization"
```

#### 3.1.3 组织成员（对应 `UserTenant`，加入角色与邀请来源）

```go
type Membership struct {
    ID       string `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID    string `gorm:"column:org_id;size:32;not null;index" json:"org_id"`
    UserID   string `gorm:"column:user_id;size:32;not null;index" json:"user_id"`
    Role     string `gorm:"column:role;size:32;not null;index" json:"role"` // owner / admin / member / viewer
    InvitedBy *string `gorm:"column:invited_by;size:32;index" json:"invited_by,omitempty"`
    Status   string `gorm:"column:status;size:1;default:1;index" json:"status"`
    BaseModel
}
// TableName: "membership"
// 联合唯一：org_id + user_id
```

#### 3.1.4 订阅计划 & 组织订阅（计费核心）

```go
// 平台定义的可售套餐（由运营后台维护）
type BillingPlan struct {
    ID          string    `gorm:"column:id;size:32;primaryKey" json:"id"`
    Name        string    `gorm:"column:name;size:100;not null" json:"name"`
    Code        string    `gorm:"column:code;size:50;uniqueIndex:uni_bp_code" json:"code"` // free/pro/enterprise
    PriceMonthlyCents int64 `gorm:"column:price_monthly_cents;default:0" json:"price_monthly_cents"`
    Active      bool      `gorm:"column:active;default:1" json:"active"`
    // 配套限额（另表），或直接内嵌关键值：
    StorageBytesLimit int64 `gorm:"column:storage_bytes_limit" json:"storage_bytes_limit"`
    TokenToParseLimit int64 `gorm:"column:token_to_parse_limit" json:"token_to_parse_limit"`
    SeatLimit          int32 `gorm:"column:seat_limit" json:"seat_limit"`          // 最多成员数
    LLMCallsLimit      int64 `gorm:"column:llm_calls_limit;default:-1" json:"llm_calls_limit"` // -1 不限
    BaseModel
}

type OrganizationSubscription struct {
    ID            string    `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID         string    `gorm:"column:org_id;size:32;uniqueIndex:uni_os_org" json:"org_id"`
    PlanID        string    `gorm:"column:plan_id;size:32;index" json:"plan_id"`
    BillingCycle  string    `gorm:"column:billing_cycle;size:16;default:'monthly'" json:"billing_cycle"` // monthly/annual
    Status        string    `gorm:"column:status;size:32;default:'trialing'" json:"status"` // trialing/active/past_due/canceled
    StartAt       *time.Time `gorm:"column:start_at" json:"start_at,omitempty"`
    RenewsAt      *time.Time `gorm:"column:renews_at" json:"renews_at,omitempty"`
    BaseModel
}

type PaymentMethod struct { // 简化：可接 Stripe/Paddle
    ID           string `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID        string `gorm:"column:org_id;size:32;index" json:"org_id"`
    Provider     string `gorm:"column:provider;size:32" json:"provider"` // stripe/paddle/sandbox
    ExternalCustomerID *string `gorm:"column:external_customer_id;size:128" json:"external_customer_id,omitempty"`
    BaseModel
}
```

#### 3.1.5 配额与用量（计量计时）

```go
// 每个组织每个计费周期一段（按月重置）
type UsageLedger struct {
    ID         string `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID      string `gorm:"column:org_id;size:32;index" json:"org_id"`
    Period     string `gorm:"column:period;size:16;index" json:"period"` // "2026-05"
    Meter      string `gorm:"column:meter;size:64;index" json:"meter"`   // storage_bytes/token_to_parse/llm_calls/seats
    Used       int64  `gorm:"column:used;default:0" json:"used"`
    BaseModel
}
// 组织(meter,period) 联合唯一

// 会话用量流水（永续，用于账单与后台统计）
type UsageEvent struct {
    ID        string    `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID     string    `gorm:"column:org_id;size:32;index" json:"org_id"`
    UserID    string    `gorm:"column:user_id;size:32;index" json:"user_id"`
    Meter     string    `gorm:"column:meter;size:64;index" json:"meter"`
    Amount    int64     `gorm:"column:amount;default:0" json:"amount"`
    RefKind   string    `gorm:"column:ref_kind;size:32" json:"ref_kind,omitempty"` // dataset/doc/chat/ingest
    RefID     string    `gorm:"column:ref_id;size:64" json:"ref_id,omitempty"`
    Billable  bool      `gorm:"column:billable;default:1" json:"billable"`
    OccurredAt time.Time `gorm:"column:occurred_at;index" json:"occurred_at"`
    BaseModel
}

type Invitation struct { // 邀请链接/邀请码
    ID        string     `gorm:"column:id;size:32;primaryKey" json:"id"`
    OrgID     string     `gorm:"column:org_id;size:32;index" json:"org_id"`
    Code      string     `gorm:"column:code;size:96;uniqueIndex:uni_inv_code" json:"code"`
    Role      string     `gorm:"column:role;size:32;default:'member'" json:"role"` // 被邀请者初始角色
    Email     *string    `gorm:"column:email;size:255;index" json:"email,omitempty"` // 限定邮箱可选
    ExpiresAt *time.Time `gorm:"column:expires_at" json:"expires_at,omitempty"`
    MaxUses   int        `gorm:"column:max_uses;default:1" json:"max_uses"`
    UsedCount int        `gorm:"column:used_count;default:0" json:"used_count"`
    Status    string     `gorm:"column:status;size:1;default:1" json:"status"`
    BaseModel
}
```

### 3.2 服务层（`internal/service/`）

沿用 RAGFlow「`handler` 只做解析、`service` 管业务、`dao` 管持久化」的分层。

| 服务 | 职责 | 关键方法 |
|---|---|---|
| `AuthService` | 注册/登录/登出/验证/刷新、鉴权 | `Register`、`LoginWithPassword`、`VerifyEmail`、`ExchangeOrgToken`、`GetUserByToken` |
| `UserService` | 个人档案、密码、会话 | `GetProfile`、`UpdateProfile`、`ChangePassword`、`ListSessions`、`RevokeSession` |
| `OrganizationService` | 组织生命周期与成员 | `CreateOrg`、`UpdateOrg`、`ListMyOrgs`、`SwitchOrg`、`InviteMember`、`AcceptInvitation`、`RemoveMember`、`UpdateMemberRole` |
| `SubscriptionService` | 套餐与支付 | `ListPlans`、`GetCurrentPlan`、`ChangePlan`、`CreateCheckout`、`HandleWebhook`、`GetInvoice` |
| `UsageMeterService` | 配额校验与计量 | `TryConsume(orgID, meter, amount)`、`ResetPeriodLedger`、`ReportEvent`、`GetUsageBreakdown` |
| `TenantLLMService` 等 | 复用 RAGFlow 原有 LLM/数据集能力（抬头到组织） | 沿用 `internal/service/tenant_llm.go` 思路 |

#### 配额校验的临界区（关键正确性点）

计量累加必须**原子**，且校验-扣减要在一个事务/一条 SQL 内完成，避免并发超卖。参考实现：

```go
// dao/usage.go —— 单条原子 UPDATE，带 where 条件做超限拒绝
func (d *UsageDAO) Consume(ctx context.Context, orgID, period, meter string, amount, limit int64) (bool, error) {
    // 若 limit >= 0 且 used+amount > limit 则拒绝；limit<0 表示不限
    if limit >= 0 {
        res := d.db.Exec(
            `UPDATE usage_ledger SET used = used + ? , update_time = NOW()
             WHERE org_id = ? AND period = ? AND meter = ? AND used + ? <= ?`,
            amount, orgID, period, meter, amount, limit,
        )
        if res.RowsAffected == 0 {
            return false, nil // 超限
        }
        return true, nil
    }
    // 不限：直接累加（用 upsert）
    ...
}
```

> ⚠️ 参照 AGENTS.md 的 GORM 陷阱：任何 `Take/First(&x, "字符串ID")` 都要显式绑定参数（`Where("id = ?", id)`），本项目沿用此约定。

#### 计费 Webhook 与订阅状态机

```go
const (
  SubTrialing = "trialing"
  SubActive   = "active"
  SubPastDue  = "past_due"
  SubCanceled = "canceled"
)
// 过渡：trialing → active → past_due →(renew)→ active / →canceled
```
- `active/past_due`：配额继续放行（past_due 提示）。
- `canceled`：本周期结束前配额不变，`renews_at` 到期后降级到免费计划。
- 平台对**欠费组织做软暂停**：`organization.status=0`，登录后看到"账户已暂停"引导续费页。

---

## 4. 用户侧前端界面设计（`web/src/pages/workspace/`）

### 4.1 路由与布局

沿用 RAGFlow 的 `Routes` 常量 + `navigation-layout` / `root-layout` 模式。

```
/login                          # 登录
/register                       # 注册（含邀请码预填）
/forgot-password                # 找回密码
/onboarding                     # 首次登录：创建组织 / 接受邀请
/workspace                      # 主工作区（受登录保护）
  ├── /overview                 # 概览（用量卡片、最近活动）
  ├── /datasets/*               # 数据集（沿用 RAGFlow 数据集页）
  ├── /agents/*                 # 智能体 / Agent（沿用）
  ├── /chats/*                  # 对话（沿用）
  ├── /members                  # 成员与邀请
  ├── /billing                  # 订阅 & 计费
  ├── /usage                    # 用量明细
  └── /settings                 # 组织设置 / 个人设置
```

**组织切换器**：放在顶部导航栏（Avatar 下拉），列出 `ListMyOrgs`，切换后调用 `ExchangeOrgToken` 重签发带 `org_id` 上下文的 JWT（或前端在后续请求附 `X-Workspace-ID`）并刷新工作区数据。

### 4.2 关键技术约定（遵循 web/AGENTS.md）

- 组件：shadcn/ui；颜色一律用 `src/tailwind.css` 的 token（`bg-bg-base`、`text-text-primary`、`border-border-button`），不写死十六进制。
- 数据请求：三层 —— `useXxxRequest` hook → `src/services/xxx-service.ts`（`registerNextServer`）→ `src/utils/next-request.ts`（统一 token/401/错误）。**hook 不直连 request 层**。
- 查询键用 **Query Key Factory**（如 `OrgKeys.members(orgID)`、`BillingKeys.usage(orgID, period)`），`as const`，不写裸数组。
- 查询就近挂到消费视图组件，父组件不提前拉子视图数据。
- 全局加载态/错误态复用 `TableEmpty`、`table-skeleton`、`Spotlight` 等既有组件。
- i18n：新文案只加到 `src/locales/en.ts` 与 `src/locales/zh.ts`。
- React：抽取具名 handler（`onClick={handleX}`），避免内联箭头；复杂逻辑抽 hook。

### 4.3 页面与关键交互

#### (a) 登录 / 注册 / 找回密码
复用 `login-next/` 的设计语言（Spotlight 背景 + 居中卡片 + 主题切换）。注册页：
- 正常注册：邮箱 + 密码 + 昵称 → `AuthService.Register` → 发验证邮件。
- 邀请注册：`/register?invite=<code>` 预填组织名与角色 → 接受邀请。
- 登录成功进入 `/onboarding`（若无组织）或 `/workspace`（有组织）。

#### (b) 概览 `/overview`
- 顶部：当前组织 + 套餐徽章。
- 用量卡片（每 Meter 一个 ProgressBar）：存储 / 解析 Token / LLM 调用 / 席位，显示 `used / limit`，接近上限变警示色。
- 最近活动列表（`UsageEvent` 聚合）。
- 配额不足的呼出：`<UsageLimitCallout> → 跳转 /billing`。

#### (c) 订阅 & 计费 `/billing`
- 套餐对比卡片（免费 / 专业 / 企业），选择后 `CreateCheckout` 进支付。
- 当前套餐卡片：周期、`renews_at`、是否 `past_due`（红色横幅 + 续费按钮）。
- 账单列表（`GetInvoice`），支持导出。
- 支付方式管理（`PaymentMethod`）。

#### (d) 用量 `/usage`
- 按月粒度，Meter 维度表格 + 折线/柱状趋势图（`dayjs` 处理时间、复用图表组件）。
- 只读，来自 `UsageLedger` / `UsageEvent` 聚合。

#### (e) 成员与邀请 `/members`
- 成员表格（复用 `@tanstack/react-table` 模式，参考 `admin/users.tsx`）：姓名、邮箱、角色、状态、加入时间、操作。
- 角色用下拉就地编辑（owner 只可转给他人、不可删除自己）。
- "邀请成员"弹窗：生成邀请链接/邀请码 → `Invitation`（可设邮箱限定、过期时间、初始角色）。
- 配额侧栏显示"席位 N/上限"，满员时禁用邀请按钮。

#### (f) 个人 / 组织设置
- 个人：昵称、头像、（邮箱验证态）、语言、时区、改密码、会话管理（退出其它设备）。
- 组织：名称、Slug、Logo；危险区：转让 owner / 删除组织。

---

## 5. 后台【管理界面】设计（`web/src/pages/admin/`）

沿用 RAGFlow `admin/` 目录结构与 `navigation-layout`（左侧导航 + 右上角版本 + 主题 + 退出），入口为独立 `/admin` 路由与**独立登录**，走 `/api/v1/admin`。

| 菜单 | 页面 | 说明 |
|---|---|---|
| 概览/大盘 | `admin/overview.tsx` | 平台全局：注册/付费组织数、活跃用户、本月营收（MRR）、总解析 Token |
| 用户管理 | `admin/users.tsx`（沿用 RAGFlow 表格） | 列表：邮箱/昵称/状态/是否平台管理员；筛选+模糊搜索+分页；操作：详情、改密、停用/启用、删除、管理员开关 |
| 用户详情 | `admin/user-detail.tsx`（沿用） | 头部资料卡 + Tabs（组织 / 数据集 / 智能体） |
| 组织管理 | `admin/organizations.tsx` | 组织列表：名称/Slug/套餐/成员数/状态/owner；操作：暂停/恢复、改套餐、删除 |
| 组织详情 | `admin/organization-detail.tsx` | 该组织套餐、用量、成员、订阅状态、操作（延后到期 / 立即暂停） |
| 套餐管理 | `admin/plans.tsx` | 增删改 `BillingPlan`（售价、各 Meter 限额、active 开关） |
| 计量用量 | `admin/usage.tsx` | 全局用量表，可按组织/Meter/周期过滤 |
| 服务状态 | `admin/service-status.tsx`（沿用） | MySQL/Kvrocks/MinIO/ES/Ingestor 健康与重启 |
| 角色&权限 | `admin/roles.tsx`（沿用企业门控） | RBAC，按资源+动作授权 |
| 系统设置 | `admin/sandbox-settings.tsx`（沿用） | 全局变量、环境、日志级别、版本 |

### 5.1 平台管理员鉴权
- 复用 RAGFlow 的 `admin/AuthMiddleware` + 独立 `admin JWT`。
- `user.is_platform_admin` 二级门控：非平台管理员请求 `/api/v1/admin` 返回 403。
- 前端 `admin/root-layout.tsx` 校验登录态，未登录跳 `/admin`。

---

## 6. API 一览（用户侧 & 后台）

### 6.1 用户侧 `/api/v1`
```
POST /auth/register                 # 注册
POST /auth/verify-email             # 邮箱验证（token）
POST /auth/login                    # 登录（JWT）
POST /auth/logout
POST /auth/refresh
POST /auth/forgot-password
POST /auth/reset-password

GET  /orgs                          # 我加入的组织
POST /orgs                          # 创建组织
GET  /orgs/:id
PUT  /orgs/:id
DELETE /orgs/:id                    # 仅 owner
POST /orgs/:id/switch               # 换当前组织（重签发 org 上下文 token）

GET  /orgs/:id/members
POST /orgs/:id/members              # 直接加成员（邀请）
PUT  /orgs/:id/members/:uid/role
DELETE /orgs/:id/members/:uid
GET  /orgs/:id/invitations
POST /orgs/:id/invitations
POST /invitations/:code/accept

GET  /billing/plans
GET  /billing/subscription
POST /billing/checkout              # 创建支付会话
POST /billing/webhook               # 支付回调
GET  /billing/invoices
GET  /billing/payment-methods

GET  /usage/current                 # 本周期用量(含 limit)
GET  /usage/events                  # 用量流水

GET/PUT /me                          # 个人设置
POST /me/change-password
POST /me/sessions/:sid/revoke
GET  /me/notifications
```

### 6.2 后台 `/api/v1/admin`
```
POST /admin/login  |  POST /admin/logout  |  GET /admin/auth
GET /admin/users            POST /admin/users            GET/PUT/DELETE /admin/users/:username
PUT /admin/users/:username/password | /activate | /admin
GET /admin/organizations    POST /admin/organizations    GET/PUT/DELETE /admin/organizations/:id
GET /admin/plans            POST /admin/plans            PUT/DELETE /admin/plans/:id
GET /admin/usage?org_id&period&meter
GET /admin/stats/overview                # MRR、活跃用户等大盘数字
GET /admin/services         PUT/DELETE/POST /admin/services/:name
GET /admin/variables    PUT /admin/variables    GET/PUT /admin/config/log
GET /admin/roles     POST /admin/roles    PUT/DELETE /admin/roles/:role
GET /admin/whitelist  POST/PUT/DELETE /admin/whitelist
GET /admin/version
```

---

## 7. 安全与治理要点

1. **密码**：注册/登录沿用 RAGFlow「前端 `rsaPsw()` 加密、服务端散列」；`PasswordHash` 用 bcrypt/argon2，绝不明文入库；日志/JSON 一律 `json:"-"`。
2. **接入令牌**：沿用 `api_token.go` 模式；对外提供每个组织独立的 API Key（`org.api_key`），创建/轮换/撤销全在设置页完成。
3. **多租户隔离**：所有数据查询强制带 `org_id` 过滤；Service 层提供 `RequireMembership(ctx, orgID, roleThreshold)` 统一校验；写操作前过配额中间件。
4. **ID 与 SQL**：沿用 AGENTS.md 的 GORM 绑定参数约定，杜绝字符串主键拼接注入。
5. **后台最小权限**：`is_platform_admin` 与用户侧 `is_superuser` 语义分离，防止普通超管触及平台运营数据。
6. **审计**：`UsageEvent` 复用为操作审计（记录 billable + ref），后台可检索。
7. **欠费治理**：`past_due` 超期 → 组织 `status=0` 软暂停 → 登录拦截引导续费 → 续费后恢复。数据不删除。

---

## 8. 落地路径（建议分阶段）

| 阶段 | 范围 | 依赖 |
|---|---|---|
| **Phase 1** | 多租户建模 + 登录/注册/邀请 + 组织切换 | `User/Organization/Membership/Invitation`、`AuthService`、基础登录页与导航 |
| **Phase 2** | 配额与计量中间件 + 用量页 + 概览卡片 | `UsageLedger/UsageEvent/UsageMeterService`、原子扣减 DAO |
| **Phase 3** | 订阅 & 计费（套餐/支付/账单） | `BillingPlan/Subscription/PaymentMethod`、支付 Webhook、`/billing` 页 |
| **Phase 4** | 平台后台（用户/组织/套餐/用量/大盘） | `/api/v1/admin` 扩展 + `admin` 各页 |
| **Phase 5** | 治理：RBAC 强化、审计、欠费软暂停、操作日志 | `roles.tsx`/`monitoring.tsx` 企业门控能力 |

> 遵循 AGENTS.md：后端新代码走 **Go**（`internal/`），不新增 Python；新前端页面遵循 `web/` 约定；测试按单元/集成/e2e 分级并保留 Qtty 关键路径（用量原子扣减）单测。

---

## 9. 可运行切片 `saas-app/` 的实现状态

上面 1–8 节是**目标设计**（含 SQL/GORM、bcrypt、独立 admin JWT、套餐 CRUD）。仓库内的 `saas-app/` 是一个可运行的纵向切片，刻意与目标设计存在差异，原因记录在此，避免把切片误读成"设计未完成"。

### 9.1 与目标设计的偏差及原因

| 目标设计 | 切片实现 | 原因 |
|---|---|---|
| MySQL/PostgreSQL + GORM | 纯标准库快照持久化（`internal/dao/persist.go`） | 构建环境无法访问 Go module 代理，无法引入任何第三方驱动 |
| bcrypt / argon2 | 标准库 `sha256` + 随机盐 + 10 万次迭代（`internal/crypto`） | 同上；`PasswordHash` 仍按 `json:"-"` 隐藏 |
| JWT + 独立 admin 登录 | 随机 opaque token + `is_platform_admin` 门控 | 切片阶段够用；换 JWT 只需替换 `AuthService` |
| 套餐 CRUD（`admin/plans.tsx`） | 只读套餐目录 + 组织改套餐 | 套餐表尚未可写 |
| 邮箱验证 / 找回密码 / 计费 | 未实现 | 属 Phase 3/5 |

### 9.2 持久化模型

- 落盘文件：`saas-app/data/saas-store.json`（`SAAS_DATA_FILE` 可改，设为空串则纯内存）。
- 每次写操作生成整库快照 → 写临时文件 → `fsync` → 原子 `rename` 覆盖。崩溃只会看到旧快照或新快照，不会看到半截文件。
- 只落主表（users / organizations / memberships / invitations / usage_ledger / usage_events），内存索引（email、slug、邀请码、成员、周期分桶）在加载时重建，因此索引不会与文件不一致。
- 代价是每次写 O(数据量)；切片数据量下可接受，规模化时替换为 SQL 后端——`dao.Store` 的方法签名本就是按 DB 形状设计的，替换面只有这一层。

### 9.3 平台后台当前能力

| 能力 | 状态 |
|---|---|
| 概览（组织数 / 用户数 / 活跃用户 / 平台管理员数 / 套餐分布） | ✅ |
| 用户列表 | ✅ |
| 停用 / 启用用户 | ✅ 平台管理员自身受保护，不可被停用（防运营后台自锁） |
| 组织列表 | ✅ |
| 修改组织套餐 | ✅ 未知套餐返回 400 |
| 删除组织 | ✅ 级联删除该组织的成员、邀请、用量台账与事件 |
| 改密 / 删除用户 / 组织详情 / 套餐 CRUD / 用量大盘 | ⬜ 未实现 |

前端入口 `/admin`（`RequireAdmin` 守卫）；平台管理员在登录、onboarding、工作区三处都会被导回后台，不会落进用户侧工作区。

### 9.4 多租户约束

- 受邀用户（持有任意非 owner 席位）**不能**自建组织，因此不能把自己升级成另一个组织的 owner；新注册用户与现有 owner 不受影响。
- 后端 `CreateOrg` 守卫 + 前端 onboarding 只给"选择组织"、不给"创建组织"表单，两侧一致。