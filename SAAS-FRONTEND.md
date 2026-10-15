# SaaS 平台前端 —— 页面线框与组件结构设计

> 前置：这是 [SAAS-DESIGN.md](SAAS-DESIGN.md) 的前端细化篇。全部遵循 `web/AGENTS.md`：shadcn/ui 组件、颜色一律用 `src/tailwind.css` token、TanStack Query + **Query Key Factory**、三层请求（hook → service → next-request）、i18n 只加 `en.ts`/`zh.ts`、`dayjs` 处理时间、ahooks 复用。

---

## 1. 前端目录结构

```
web/src/pages/
├── login/                     # 用户侧登录/注册/找回密码（复用 login-next 设计语言）
│   ├── index.tsx              # /login + /register 共用一个 3D 翻转卡片容器
│   ├── components/
│   │   ├── auth-card.tsx      # 复用 FlipCard3D，两面登录/注册
│   │   ├── login-form.tsx
│   │   ├── register-form.tsx
│   │   ├── forgot-form.tsx
│   │   └── email-verify.tsx   # 邮箱验证结果页
│   └── constant/index.ts
├── onboarding/
│   ├── index.tsx              # /onboarding 首次登录引导
│   ├── create-org.tsx         # 创建组织
│   └── accept-invite.tsx      # 接受邀请
├── workspace/                 # 主工作区（登录后，受 OrgGuard 保护）
│   ├── __layout.tsx           # WorkspaceLayout（顶部导航 + Outlet）
│   ├── __org-switcher.tsx     # 顶部组织切换器
│   ├── overview/index.tsx     # 概览
│   ├── datasets/              # 沿用既有数据集页路径
│   ├── agents/                # 沿用既有智能体页
│   ├── chats/                 # 沿用既有对话页
│   ├── members/index.tsx      # 成员与邀请
│   ├── billing/index.tsx      # 订阅 & 计费
│   ├── usage/index.tsx        # 用量明细
│   └── settings/              # 个人 & 组织设置
└── admin/                     # （沿用 RAGFlow admin/，新增 organization/ billing/usage/overview）
    └── (见第 7 节)
```

---

## 2. 共享基础设施（新增于 `web/src/`）

### 2.1 请求层与类型

| 文件 | 内容 |
|---|---|
| `src/services/auth-service.ts` | `register`、`verifyEmail`、`login`、`logout`、`refresh`、`forgotPassword`、`resetPassword` → `registerNextServer('/api/v1/...')` |
| `src/services/org-service.ts` | 组织 CRUD、切换、成员、邀请 |
| `src/services/billing-service.ts` | 套餐、订阅、支付、账单、支付方式 |
| `src/services/usage-service.ts` | 当前用量、用量流水 |
| `src/interfaces/database/org.ts` | `Organization`/`Membership`/`Invitation` 响应形状 |
| `src/interfaces/database/billing.ts` | `BillingPlan`/`Subscription`/`Invoice` |
| `src/interfaces/request/*.ts` | 各接口入参 |

### 2.2 Query Key Factory（强制，`as const`）

```ts
// src/hooks/keys.ts 或按域分文件
export const AuthKeys = {
  me: () => ['auth', 'me'] as const,
  notifications: () => ['auth', 'notifications'] as const,
};

export const OrgKeys = {
  list: () => ['org', 'list'] as const,
  detail: (orgId: string) => ['org', 'detail', orgId] as const,
  members: (orgId: string) => ['org', 'members', orgId] as const,
  invitations: (orgId: string) => ['org', 'invitations', orgId] as const,
};

export const BillingKeys = {
  plans: () => ['billing', 'plans'] as const,
  subscription: (orgId: string) => ['billing', 'subscription', orgId] as const,
  invoices: (orgId: string) => ['billing', 'invoices', orgId] as const,
};

export const UsageKeys = {
  current: (orgId: string) => ['usage', 'current', orgId] as const,
  events: (orgId: string, period: string, meter: string) =>
    ['usage', 'events', orgId, period, meter] as const,
};
```

### 2.3 组织上下文（Zustand store）

```ts
// src/stores/workspace-store.ts
type WorkspaceState = {
  org: Organization | null;            // 当前组织
  memberships: Membership[];           // 我加入的所有组织
  setOrg: (org: Organization) => void;
  setMemberships: (m: Membership[]) => void;
};
```

`next-request.ts` 拦截器在请求头附加 `X-Workspace-ID: state.org.id`。

---

## 3. 复用 RAGFlow 既有组件映射

| 本设计用到的能力 | 复用 RAGFlow 现有组件/文件 |
|---|---|
| 登录卡片翻转 | `pages/login-next/card.tsx`（`FlipCard3D`）、`bg.tsx`（`BgSvg`）、`index.less` |
| 数据表格 + 筛选 + 分页 | `admin/users.tsx` 的 `@tanstack/react-table` 模式、`RAGFlowPagination`、`TableEmpty`、`createColumnFilterFn`/`createFuzzySearchFn`（`admin/utils.tsx`） |
| 表单（受控+校验） | `react-hook-form` + `zodResolver`，参考 `admin/forms/*` |
| 弹窗 | `ui/dialog`、`ui/drawer` |
| 头像 | `components/ragflow-avatar` |
| 徽章/状态点 | `ui/badge`、`LucideDot` |
| 主题切换/右上角 | `components/theme-switch` |
| 空态 | `components/empty/empty`、`components/table-skeleton` |
| 授权头 & 401 跳转 | `utils/authorization-util`、`utils/request` |
| 时间工具 | `dayjs` |
| 复用 hook | `ahooks`（`useDebounce`、`useSetState`） |

---

## 4. 用户侧页面线框

### 4.1 登录 / 注册（`/login`、`/register`）

复用 `login-next` 的 Spotlight 背景 + 3D 翻转卡片。`/login` 正面登录、`/register` 背面注册，共用容器（保留 FlipCard3D 交互）。

```
┌──────────────────────────────────────────────────────────┐
│                         Spotlight 背景                     │
│                                                          │
│                  ┌──────────────────────────┐            │
│                  │   [Logo]   SaaS 平台      │  ——正面(登录)│
│                  │  ┌────────────────────┐  │            │
│                  │  │ 邮箱  [__________]  │  │            │
│                  │  │ 密码  [__________]  │  │            │
│                  │  │         [登录]      │  │            │
│                  │  │ 忘记密码?  注册→     │  │            │
│                  │  │  [Google/GitHub]   │  │            │
│                  │  └────────────────────┘  │            │
│                  │        [主题切换]          │            │
│                  └──────────────────────────┘            │
└──────────────────────────────────────────────────────────┘
```

**组件树**（`login/`）

```
<LoginPage>                     # /login, /register 共用
 ├─ <BgSvg/>                    # 背景
 ├─ <ThemeSwitch/>
 └─ <FlipCard3D isLoginPage={path==='/login'}>
     ├─ <Front>/<AuthCard> → <LoginForm>      # 正面
     │      ├─ <RAGFlowFormItem email/password/>
     │      ├─ <Button type="submit">登录</Button>
     │      └─ <Link to="/forgot">忘记密码</Link>
     └─ <Back>/<AuthCard> → <RegisterForm>    # 背面
            ├─ <RAGFlowFormItem email/nickname/password/confirm/>
            ├─ invite 参数预填（useSearchParams）
            ├─ <Button>注册&创建组织</Button>
            └─ <Link to="/login">已有账号</Link>
```

`LoginForm` 用 `LoginKeys` hook：

```ts
// src/hooks/use-auth-request.ts
const useLogin = () =>
  useMutation({
    mutationKey: AuthKeys.me(),
    mutationFn: ({ email, password }) =>
      login({ email, password: rsaPsw(password) as string }),
    // onSuccess 存 token（authorizationUtil）+ 跳转 /onboarding 或 /workspace
  });
```

### 4.2 首次登录引导（`/onboarding`）

```
┌─────────────────────┐   ┌─────────────────────┐
│  创建你的组织          │   │  你被邀请了           │
│  ┌────────────────┐  │   │  ┌────────────────┐ │
│  │ 组织名 [______]  │  │   │  团队: Acme Inc.  │ │
│  │ 友好名 [______]  │  │   │  角色: Member     │ │
│  │ 套餐: Free      │  │   │  ┌──────────────┐ │ │
│  │     [创建]      │  │   │  │  [接受邀请]    │ │ │
│  └────────────────┘  │   │  └──────────────┘ │ │
└─────────────────────┘   └─────────────────────┘
  有邀请码时显示右侧卡片         由左侧创建组织链接
```

**组件树**：`OnboardingPage` → `CreateOrgForm` / `AcceptInviteCard`（`OrgKeys.list` 刷新后跳 `/workspace`）。

### 4.3 主工作区布局（`/workspace`）

```
┌──────────────────────────────────────────────────────────────────┐
│ [Logo]  [组织切换▾]              (顶部导航)         USER▾  主题  │
│                                 ↗ 概览 / 数据集 / 智能体 / 对话       │
├──────────────────────────────────────────────────────────────────┤
│   (Outline 路由渲染当前页)                                          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**`WorkspaceLayout`**：由 `useAuth()` 守卫登录态 + `OrgGuard` 守卫已选组织。顶部 `<OrgSwitcher>`（下拉列出 `useQuery(OrgKeys.list())`，选中调 `switchOrg` 换 org 上下文并 `invalidateQueries` 所有 `X-Workspace-ID` 相关键）。

### 4.4 概览（`/workspace/overview`）

```
┌──────────────────────────────────────────────────────────────┐
│ 概览                           Org: Acme Inc  [Free Plan]     │
├──────────────────────────────────────────────────────────────┤
│ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│ │ 存储空间   │ │ 解析 Token │ │ LLM 调用  │ │ 席位      │          │
│ │ 1.2/5 GB │ │ 300/2.5M │ │ 1.8k/∞   │ │ 5/10     │          │
│ │ [====]   │ │ [===]    │ │ [==]     │ │ [=====]  │          │
│ └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                                              │
│ 最近活动                                      查看全部 →       │
│ ┌──────────────────────────────────────────┐                │
│ │ ● 解析完成  report.pdf  2分钟前  (dataset)│                │
│ │ ● 对话请求  FAQ Bot      10分钟前         │                │
│ │ ● 新建数据集  Q3数据     昨天              │                │
│ └──────────────────────────────────────────┘                │
└──────────────────────────────────────────────────────────────┘
```

**组件树** & **数据流**：

```
<OverviewPage>
 ├─ useQuery(UsageKeys.current(org.id)) → <MeterCardGrid>
 │     └─ <MeterCard meter → {label,icon,tone}>   # 近限→警示色
 │           ├─ <Progress value=used/limit color=token/>
 │           └─ <UsageLimitCallout onCta={()=>nav('/billing')}/>
 ├─ useQuery(UsageKeys.events(org.id, thisMonth, '')) → <RecentActivity>
 └─ <Spotlight/> 背景
```

### 4.5 成员与邀请（`/workspace/members`）

```
┌──────────────────────────────────────────────────────────────────┐
│ 成员 & 邀请                              [搜索]  [邀请成员 +]        │
├──────────────────────────────────────────────────────────────────┤
│ 成员 (5/10 席位)                                                  │
│ ┌────────────┬──────────┬────────┬──────────┬──────┬─────────┐  │
│ │ (头像) 姓名 │ 邮箱       │ 角色 ▾  │ 状态      │ 加入时间│ 操作     │  │
│ ├────────────┼──────────┼────────┼──────────┼──────┼─────────┤  │
│ │ Alice      │ a@x.com  │ owner  │ 已接受    │ Mar  │ [详情]   │  │
│ │ Bob        │ b@x.com  │ admin ▾│ 已接受    │ Apr  │ [移除]   │  │
│ │ Carol      │ c@x.com  │ member▾│ 已邀请    │ --   │ [重发][删]│  │
│ └────────────┴──────────┴────────┴──────────┴──────┴─────────┘  │
└──────────────────────────────────────────────────────────────────┘

邀请成员弹窗（Dialog）：
  邀请链接 [https://app/register?invite=abc...]  [复制]
  或 邮箱模糊邀请: [email]  角色: [member ▾]
  有效期: [7天 ▾]   剩余席位可用: 5
  ┌──────────────────────┐
  │        [生成邀请]      │
  └──────────────────────┘
```

**组件树 / 数据流**：

```
<MembersPage>
 ├─ useQuery(OrgKeys.members(org.id))        → <MembersTable>
 │     (复用 admin/users 表格骨架；角色列用 Select 就地改 → updateMemberRole mutation → invalidate)
 ├─ useQuery(UsageKeys.current(org.id))       → <SeatUsageBar/>  满员禁用邀请
 └─ useCreateInvitationForm() → <InviteMemberDialog>
       - useMutation(post /orgs/:id/invitations)
       - onSuccess → invalidate(OrgKeys.invitations)
```

### 4.6 订阅 & 计费（`/workspace/billing`）

```
┌──────────────────────────────────────────────────────────────────┐
│ 订阅 & 计费                        Acme Inc   [专业版 Monthly]     │
├──────────────────────────────────────────────────────────────────┤
│ 当前套餐卡                        [续费/升级]  [更换套餐]           │
│ ┌────────────────────────────────────────┐                       │
│ │ 专业版     $49/月   续费: 2026-06-01   │                       │
│ │ ● 存储 50GB  ● Token 25M  ● 席位 20   │                       │
│ │ 状态: Active ✓  (past_due 时红色横幅)  │                       │
│ └────────────────────────────────────────┘                       │
│                                                                  │
│ 套餐对比（免费 / 专业 / 企业）                                     │
│ ┌────────────────┬────────────────┬───────────────┐             │
│ │   Free $0      │  Pro $49/月▸   │  Enterprise    │             │
│ │  5GB/2.5M/10席 │ 50GB/25M/20席  │ 自定义/SSO     │             │
│ │   [当前套餐]     │   [选择专业]    │  [联系销售]     │             │
│ └────────────────┴────────────────┴───────────────┘             │
│  账单历史                                                       │
│ ┌────────────────────────────┬────────┬─────────┐               │
│ │ 2026-04-01  Pro Monthly    │ $49    │ [发票PDF]│               │
│ └────────────────────────────┴────────┴─────────┘               │
└──────────────────────────────────────────────────────────────────┘
```

**组件树 / 数据流**：

```
<BillingPage>
 ├─ useQuery(BillingKeys.subscription(org.id)) → <CurrentPlanCard>
 │     - status===past_due → <PastDueBanner onRenew={()=>checkout}/>
 ├─ useQuery(BillingKeys.plans())             → <PlanGrid>
 │     <PlanCard plan onSelect={()=>checkoutMutation.mutate(plan.code)}>
 └─ useQuery(BillingKeys.invoices(org.id))    → <InvoicesTable>
```

### 4.7 用量（`/workspace/usage`）

```
┌──────────────────────────────────────────────────────────────────┐
│ 用量明细                                    [周期: 2026-05 ▾]      │
├──────────────────────────────────────────────────────────────────┤
│ ┌──── 存储空间 ────┬──── 解析 Token ─┐  折线图（按 Meter 切换）     │
│ │ [趋势折线图]      │ [趋势折线图]     │                             │
│ └─────────────────┴───────────────┘                              │
│  明细表格：日期 / Meter / 消耗 / 来源(数据集/对话/解析) / 计费      │
│ ┌────────┬────────┬────┬─────────┬──────┐                       │
│ │ 日期    │ Meter   │消耗│ 来源    │ 计费  │                       │
│ │ 05-01  │ storage│1GB │ dataset │ ✓    │                       │
│ └────────┴────────┴────┴─────────┴──────┘                       │
└──────────────────────────────────────────────────────────────────┘
```

**组件树**：`UsagePage` → `<MeterTabs>`（切换 meter → `UsageKeys.events(org, period, meter)`）→ `<UsageChart>`（图表库）+ `<UsageEventsTable>`。

### 4.8 设置（`/workspace/settings`）

```
[Tabs]  个人资料 | 账号安全 | 组织 | 会话
个人资料: 头像 / 昵称 / 语言 / 时区   [保存]
账号安全: 邮箱(验证态) / [修改密码]
组织:     名称 / Slug / Logo        [保存]
          ─ 危险区 ─  转移 Owner / 退出组织 / 删除组织
会话:     活动设备列表 [退出此设备]
```

---

## 5. 复用业务包的组合方式

**数据集 / 智能体 / 对话页**：直接沿用 RAGFlow 已有的 `pages/dataset/`、`pages/agent/`、`pages/chat/` 目录，仅做两处 SaaS 化接线：

1. 它们的作用对象从"当前用户"改为"**当前组织**"——查询 hook 增加 `org_id` 入参，`X-Workspace-ID` 头由 `next-request` 统一附加，页面代码不改。
2. 让它们在动作前调用 `useRequireQuota(meter)` 守卫，资源不足时弹出迁移到 `/billing` 的引导。

这样新增的 SaaS 页面自成体系，既有业务能力零改动接管。

---

## 6. 关键数据流时序

### 6.1 登录 → 进入工作区

```
Submit(LoginForm)
  → useLogin mutation
  → auth-service.login → POST /auth/login
  → onSuccess: 存 token (authorizationUtil)
  → navigate('/onboarding' if no org else '/workspace')
WorkspaceLayout mount
  → useAuth() 校验 token
  → OrgGuard: writespace-store.org 为空 → 读 OrgKeys.list()
      ├─ 有组织 → 默认选中最近使用 → 设 store.org
      └─ 无组织 → 跳 /onboarding
  → 各子页 useQuery 携 X-Workspace-ID 拉数据
```

### 6.2 切换组织

```
OrgSwitcher 选中 Acme
  → org-service.switchOrg → POST /orgs/:id/switch (重签发 org 上下文 token)
  → 更新 workspace-store.org
  → queryClient.removeQueries() (清空旧 org 缓存) 或重挂 subtree
  → 各查询自动带新 X-Workspace-ID 重拉
```

### 6.3 邀请新成员

```
InviteMemberDialog 提交
  → mutation POST /orgs/:id/invitations
  → onSuccess invalidate(OrgKeys.invitations)
  → 列表显示"已邀请"状态；对方接受后 invalidate(OrgKeys.members) 刷新
```

---

## 7. 后台管理新增页面（`web/src/pages/admin/`）

沿用 RAGFlow `admin/navigation-layout.tsx` 左侧导航。新增菜单项插入既有导航数组：

```ts
navItems = [
  { path: Routes.AdminOverview,        name: t('admin.overview') },
  { path: Routes.AdminServices,        ... },        // 沿用
  { path: Routes.AdminUserManagement,  ... },        // 沿用
  { path: Routes.AdminOrganizations,   name: t('admin.organizations') }, // 新
  { path: Routes.AdminPlans,           name: t('admin.plans') },         // 新
  { path: Routes.AdminUsage,           name: t('admin.usage') },         // 新
  { path: Routes.AdminSandboxSettings, ... },        // 沿用
  ...IS_ENTERPRISE ? [Whitelist/Roles/Monitoring] : [],
]
```

### 7.1 组织管理（`admin/organizations.tsx` + `organization-detail.tsx`）

复用 `users.tsx` 表格骨架（`createColumnHelper` + 筛选 + 分页）：

```
组织列表：名称 / Slug / 套餐 / 成员数 / 引导状态(active/suspended) / owner / 创建时间 / 操作
操作: [详情] [改套餐] [暂停/恢复]
组织详情：头部资料卡 + Tabs(成员 / 用量 / 订阅状态)
  订阅状态卡: 套餐 / 周期 / renews_at / status → 动作 [延后到期] [立即暂停] [升级]
```

### 7.2 套餐管理（`admin/plans.tsx`）

```
套餐列表：名称 / 编码 / 月价 / 存储/Tok/席位/LLM 限额 / 上架开关 / 操作(编辑/删除)
新增套餐弹窗：BillingPlan 表单（数字限额 + active 开关）
```

### 7.3 用量大盘（`admin/usage.tsx`）

复用 `roles.tsx` 的 `Tabs` + 筛选：全局用量表（按 组织/Meter/周期 过滤）＋顶部 `stats/overview` 数字卡（MRR / 注册组织数 / 活跃用户 / 总 Token）。

### 7.4 概览大盘（`admin/overview.tsx`）

```
┌────────────────────────────────────────────────────────┐
│ 平台概览                                                │
│ [TODAY] MRR      [注册组织]  [活跃用户]  [本月 Token]     │
│   $12,400           1,208        3,405       88.2M      │
│ ┌────────── 营收趋势折线 ──────────┐ ┌ 套餐分布饼图 ─┐   │
│ │                                │ │ Free 62%      │   │
│ │                                │ │ Pro  28%      │   │
│ │                                │ │ Ent  10%      │   │
│ └────────────────────────────────┘ └───────────────┘   │
└────────────────────────────────────────────────────────┘
```

**后台数据流**：新增 `admin-service.ts` 的 `listOrganizations` / `listPlans` / `overviewStats` / `listGlobalUsage`，全部走既有 `/api/v1/admin` 通道（复用 `request` 拦截器 + 401 跳 `/admin`）。

---

## 8. 组件与复用清单（速查）

| 新增组件 | 位置 | 复用 | 依赖 hook/service |
|---|---|---|---|
| `AuthCard/LoginForm/RegisterForm` | `pages/login/components/` | `FlipCard3D`、`Spotlight` | `useLogin`/`useRegister` |
| `OrgSwitcher` | `pages/workspace/__org-switcher.tsx` | `ui/dropdown-menu`、`RAGFlowAvatar` | `OrgKeys.list` |
| `MeterCardGrid` / `MeterCard` | `pages/workspace/overview/` | `ui/progress`、`ui/card` | `UsageKeys.current` |
| `UsageLimitCallout` | `components/usage/` | `ui/alert`、`ui/button` | none（props 回调） |
| `MembersTable` | `pages/workspace/members/` | tanstack-table 骨架 | `OrgKeys.members` |
| `InviteMemberDialog` | `pages/workspace/members/` | `ui/dialog`、RHF+zod | `post invitations` |
| `CurrentPlanCard`/`PlanGrid`/`PlanCard` | `pages/workspace/billing/` | `ui/card`、`ui/badge` | `BillingKeys.*` |
| `InvoicesTable` | `pages/workspace/billing/` | tanstack-table | `BillingKeys.invoices` |
| `UsageChart`/`UsageEventsTable` | `pages/workspace/usage/` | 图表库 + table | `UsageKeys.events` |
| `SettingsPage` + 子表单 | `pages/workspace/settings/` | RHF+zod、`ui/tabs` | `me.*`、`org.*` |
| `OrganizationTable` | `pages/admin/organizations.tsx` | users.tsx 骨架 | `admin-service` |

---

## 9. 落地建议（遵循 web/AGENTS.md —— 强制约定复查）

1. **永远用 Query Key Factory**（第 2.2 节），新增键一律入 Factory，不写裸数组；`invalidateQueries` 引用同一 Factory。
2. **hook 不直接 import request 层**——只调 `service` 层（`org-service`/`billing-service`/`usage-service`）。
3. **颜色 token**：所有新增样式用 `bg-bg-base`/`text-text-primary`/`border-border-button` 等，不写死 hex；除非明确授权，不改 `src/components/ui/`。
4. **就近挂查询**：`MembersTable` 自己拉 `OrgKeys.members`，不要把查询上提到 `MembersPage` 父组件。
5. **内联 handler 提取**：`onClick={handleCreate}` 而非 `onClick={() => ...}`。
6. **i18n**：新文案只加 `en.ts`（句首大写、其余小写）/ `zh.ts`。
7. **时间**：一律 `dayjs`；**ts:`dayjs().format('YYYY-MM-DD')`**。

按此设计实施，用户侧（登录/注册/引导/概览/成员/计费/用量/设置）与后台（组织/套餐/用量/概览）可并行开发，共用第 2 节基础设施与第 8 节复用清单。