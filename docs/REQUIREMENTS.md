# KKCert 证书运维平台 — 需求规格说明书

| 项目 | 说明 |
|------|------|
| 版本 | v1.4.5 |
| 日期 | 2026-07-07 |
| 状态 | 待审核 |

## 1. 背景与目标

运维团队需统一管理多个根域名的 TLS 证书：通过 GoDaddy DNS 完成 ACME 验证，自动申请 Let's Encrypt 免费证书（有效期 90 天），到期前自动续签，并将证书同步至 Git 仓库供下游系统拉取。

v1.4 增加完整 OpenAPI 文档与 API Token 管理机制，便于 AI Agent 与自动化脚本接入。

## 2. 范围

### 2.1 包含

| 编号 | 功能 | 说明 |
|------|------|------|
| F-01 | 根域名批量管理 | 增删查根域名，支持 `*.example.com` 通配符 |
| F-02 | 证书申请 | DNS-01 验证，对接 GoDaddy API |
| F-03 | 到期检测 | 每日定时扫描所有证书剩余天数 |
| F-04 | 自动续签 | 剩余天数 ≤ 阈值时触发续签 |
| F-05 | Git 同步 | 续签后自动 push；证书列表可手动重新同步 |
| F-06 | Web 管理后台 | 域名、证书、设置的可视化管理 |
| F-07 | 手动续签 | 后台一键触发指定域名续签 |
| F-08 | 登录与权限 | 本地账号 + OIDC SSO，基于角色的 API 访问控制 |
| F-09 | 用户管理 | 管理员增删改用户、分配角色 |
| F-10 | ACME 账户持久化 | 复用 LE 账户密钥，避免重复注册触发限流 |
| F-11 | 主题切换 | 浅色 / 深色模式，跟随系统或手动切换，偏好本地持久化 |
| F-12 | 数据清理 | 续签后物理删除旧证书；定期清理无效历史数据 |
| F-13 | OpenAPI & API Token | Swagger 在线文档；可吊销的 API Token，供 Agent 接入 |

### 2.2 不包含（v1）

- 多 DNS 提供商支持（仅 GoDaddy）
- SAML / LDAP 登录（仅 OIDC）
- 细粒度资源级权限（按域名授权）
- 证书部署到 CDN/负载均衡

## 3. 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25+ |
| 前端 | React 18 + Vite + TypeScript |
| 数据库 | bbolt（嵌入式 KV） |
| 部署 | Docker Compose + 多阶段 Dockerfile（推荐） |
| 证书 | Let's Encrypt（ACME v2，90 天有效期） |
| DNS | GoDaddy API（DNS-01） |
| 认证 | 本地密码（bcrypt）+ OIDC（Authorization Code）+ API Token |
| API 文档 | OpenAPI 3.0（`backend/internal/api/openapi.yaml`，运行时 `GET /api/openapi.yaml`）+ Swagger UI |

## 4. 功能需求

### 4.1 域名管理（F-01）

- 支持批量添加根域名（每行一个或逗号分隔）
- 字段：`domain`（必填）、`wildcard`（是否申请 `*.domain`）、`enabled`（是否参与检测）
- 删除域名时归档域名记录，并物理删除关联证书
- **权限**：`admin` / `operator` 可写；`viewer` 只读

### 4.2 证书申请与续签（F-02 / F-04 / F-07）

**申请流程：**

```
配置 GoDaddy 凭证 → 加载/创建 ACME 账户 → 添加域名 → 触发申请
  → ACME DNS-01（GoDaddy 自动添加 TXT 记录）
  → 获取 cert + key → 存储 bbolt → 同步 Git
```

- 证书覆盖：`example.com` + `*.example.com`（wildcard 开启时）
- 续签成功后 **物理删除** 该域名下的旧证书记录，仅保留最新一份
- 手动续签：API `POST /api/domains/{id}/renew`（异步，立即返回 `{"status":"started"}`）

#### 手动续签状态展示（F-07）

域名管理页「申请/续签」按钮触发后，页面顶部展示**可关闭的状态条**（`notice`），实时反映本次任务进度；详情仍可在「操作日志」页查看。

**交互流程：**

```
点击「申请/续签」
  → 按钮变为「申请中...」并禁用（同页删除按钮一并禁用）
  → 状态条：正在提交申请...
  → API 返回 started
  → 状态条：证书申请中，DNS 传播可能需要 1～5 分钟...
  → 每 3s 轮询操作日志，匹配本次任务
  → 成功 / 失败 / 超时（12min）更新状态条
```

**状态条样式：**

| `kind` | 场景 | 示例文案 |
|--------|------|----------|
| `info` | 提交中、进行中 | `正在提交申请...` / `证书申请中，DNS 传播可能需要 1～5 分钟...` |
| `ok` | 申请成功 | `申请成功：expires 2026-10-05` |
| `err` | 失败或超时 | ACME 错误原文 / `申请超时，请到操作日志查看详情` |

- 状态条左侧加粗显示域名，右侧 `×` 可手动关闭
- 轮询**仅匹配本次点击之后**产生的日志（`created_at >= 点击时刻`），避免误展示历史失败记录

**后端日志约定（`action=renew`）：**

| 顺序 | level | message | 含义 |
|------|-------|---------|------|
| 1 | `info` | `started` | 任务已受理，ACME 流程开始 |
| 2a | `info` | `expires YYYY-MM-DD` | 申请成功 |
| 2b | `error` | ACME 错误详情 | 申请失败 |

- 前端轮询时忽略 `message=started`，继续等待终态日志
- 自动续签失败同样写入 `action=renew` 错误日志，但**不触发**域名页状态条（仅操作日志可见）

**轮询参数：**

| 项 | 值 |
|----|-----|
| 间隔 | 3s |
| 超时 | 12min（覆盖 DNS 传播最长 10min + 缓冲） |
| 匹配条件 | `action=renew` 且 `domain` 一致且 `created_at` 在本次点击之后 |

#### 多域名并发申请（F-02.1）

多个根域名**同时**点「申请/续签」时，各域名在后台以**独立异步任务**运行，整体原则：**不同域名互不抢 DNS 记录，同一域名互斥，共享资源需注意限流与 Git 冲突**。

**按域名隔离（互不影响）：**

| 项 | 说明 |
|----|------|
| 任务粒度 | 每域名一次 `POST /api/domains/{id}/renew` → 独立 goroutine |
| DNS 记录 | 各根域名写入各自的 `_acme-challenge.<domain>`，不同 zone 不交叉 |
| 数据存储 | 证书按 `domain_id` 分桶保存；续签成功仅删除**该域名**旧证书 |
| 同域互斥 | 同一域名并发重复点击时，后者等待前者结束（进程内 mutex） |
| 操作日志 | `action=renew` + `domain` 字段区分，可并行产生多条日志 |

**共享资源（可能相互影响）：**

| 资源 | 行为 | 建议 |
|------|------|------|
| Git 工作区 | 续签成功后的自动 push 共用 `./data/git-workspace`；多域名**并行**完成时可能 push 冲突 | 证书已入库；失败看 `git_sync` 日志，用证书列表 **一键同步 Git** 补推 |
| GoDaddy API | 手动并行无间隔；定时任务串行且间隔 10s | 批量手动申请时宜错开，或依赖定时任务 |
| Let's Encrypt | 共用持久化 ACME 账户 | 避免短时间大量并行，注意 LE 频率限制 |
| 前端状态条 | 同时多点时，状态条只反映**当前页面最后一次点击**的域名 | 其余域名进度查 **操作日志** |

**定时自动续签：** 每日检测按域名**串行**执行，相邻域名间隔 **10s**（GoDaddy 限流），不会并行抢资源。

**单证书内通配符：** 同一域名的 `example.com` + `*.example.com` 共用 `_acme-challenge.example.com`，属于**单次申请内部**顺序处理，与「多根域名并发」无关。

#### GoDaddy DNS-01 行为

| 项 | 说明 |
|----|------|
| 实现 | lego `providers/dns/godaddy` |
| 验证记录 | `_acme-challenge` TXT |
| DNS TTL | **≥ 600 秒**（GoDaddy API 强制下限）；系统默认 **600** |
| 传播检测 | Present 前清空 `_acme-challenge` TXT；首次检测前等待 45s；轮询至多 10min；以权威 NS 为准（`domaincontrol.com`），辅以 `8.8.8.8` / `1.1.1.1` |

- 申请/续签时自动向 GoDaddy 写入 TXT 记录，完成后清理挑战记录
- **申请前**自动清除该域名下 `_acme-challenge` 的全部 TXT（避免失败重试残留导致 `Incorrect TXT record`）
- 申请失败时再次尝试清除 `_acme-challenge` TXT
- 凭证通过系统设置 `godaddy_api_key` / `godaddy_api_secret` 配置
- 若 TTL 未设置或 < 600，GoDaddy 返回 `invalid TTL, TTL (0) must be greater than 600`，申请失败

### 4.2.1 数据清理（F-12）

#### 续签即时清理

| 时机 | 行为 |
|------|------|
| 证书申请/续签成功 | 删除同域名下所有旧证书 KV，仅保留新证书 |
| 删除域名（归档） | 物理删除该域名下全部证书记录 |

#### 定期清理

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `cleanup_cron` | `0 4 * * *` | 每日清理任务 Cron |

**清理范围：**

| 数据 | 条件 |
|------|------|
| 证书 | 所属域名已归档，或 `active=false` 的残留记录 |
| 会话 | 已过期的 Session |
| API Token | 已过期的 Token 记录 |
| 操作日志 | 写入时保留最近 50 条（已有逻辑） |

- 清理结果写入操作日志（`action=cleanup`）
- 管理员可手动触发：`POST /api/cleanup/run`

### 4.3 ACME 账户持久化（F-10）

| 项 | 说明 |
|----|------|
| 存储位置 | bbolt `acme_accounts` bucket |
| 分区键 | `production` / `staging`（与 `acme_staging` 设置对应） |
| 存储内容 | 账户私钥 PEM、`registration.Resource` JSON |
| 行为 | 首次申请时注册并持久化；后续续签复用同一账户 |
| 重置 | 管理员可在设置中触发「重置 ACME 账户」（需确认） |

### 4.4 到期检测（F-03）

- 默认每日 **03:00**（服务器本地时区）执行全量扫描
- 扫描间隔可在设置中配置（cron 表达式）
- 输出：域名、过期时间、剩余天数、状态（正常 / 即将过期 / 已过期）
- 定时任务以系统身份运行，不依赖用户会话

### 4.5 自动续签（F-04）

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `renew_before_days` | 30 | 剩余天数 ≤ 此值时自动续签 |
| `auto_renew_enabled` | true | 全局开关 |

- 续签失败记录错误日志，前端展示最近错误
- 同一域名 24h 内不重复自动续签（防抖动）

### 4.6 Git 同步（F-05）

**仓库目录结构：**

```
<certs-dir>/
  example.com/
    fullchain.pem
    privkey.pem
    metadata.json    # 过期时间、续签时间
```

- 支持 HTTPS（Token）或 SSH（私钥路径）鉴权
- 每次证书变更自动 commit，message：`kkcert: renew example.com`
- push 失败保留本地 commit，下次重试

#### 手动重新同步（F-05.1）

适用于证书已在 KKCert 中签发，但 Git 仓库未更新或 push 曾失败的场景（含多域名批量补推）。

| 项 | 说明 |
|----|------|
| 入口 | **证书列表** 页头 **一键同步 Git** |
| 权限 | `admin` / `operator` |
| API | `POST /api/certificates/sync-git` |
| 行为 | 拉取全部**有效**证书，写入 Git 工作区，**单次 commit + push** |
| 范围 | 所有 `active=true` 的证书记录（每个域名目录各一份 `fullchain.pem` / `privkey.pem` / `metadata.json`） |
| commit message | `kkcert: sync all (N domains)` |
| 响应 | `{"status":"ok","count":N}`，`N` 为同步域名数 |
| 日志 | 成功 `git_sync` / `pushed N certificates`；失败 `git_sync` / `error` |
| 前置条件 | 系统设置中已配置 `git_repo_url` 及鉴权；至少有一张有效证书 |

- 同步为**同步请求**，完成后界面提示同步域名数量或错误原文
- 不触发 ACME 续签，仅推送已有证书记录
- 单域名续签成功时仍自动执行单域名 `kkcert: renew {domain}` push（与批量同步独立）

### 4.7 登录与权限控制（F-08）

#### 认证方式

| 方式 | 说明 |
|------|------|
| 本地账号 | 用户名 + 密码，bcrypt 存储 |
| OIDC SSO | Authorization Code Flow，首次登录自动建用户 |
| Session Token | 登录后 24h 有效，等同用户角色 |
| API Token | 管理员创建，可指定角色与有效期（见 4.9） |

#### 角色与权限

| 角色 | 权限 |
|------|------|
| `admin` | 全部操作：用户管理、Token 管理、系统设置、域名、续签 |
| `operator` | 域名管理、续签、触发检测；只读设置 |
| `viewer` | 只读：概览、域名、证书、日志 |

#### 会话

- 登录成功返回 Session Token（Bearer）
- 默认有效期 24h，过期需重新登录
- `POST /api/auth/logout` 销毁会话

#### OIDC 配置项

| 配置项 | 说明 |
|--------|------|
| `oidc_enabled` | 是否启用 OIDC 登录 |
| `oidc_issuer` | IdP Issuer URL |
| `oidc_client_id` | Client ID |
| `oidc_client_secret` | Client Secret |
| `oidc_redirect_url` | 回调地址，如 `https://kkcert.example.com/api/auth/oidc/callback` |
| `oidc_default_role` | 新 OIDC 用户默认角色，默认 `viewer` |

#### 引导账号

- 首次启动且无用户时，创建 `admin` 账号
- 初始密码来自环境变量 `KKCERT_BOOTSTRAP_PASSWORD`，未设置则 `changeme` 并打日志告警

### 4.8 用户管理（F-09）

- **权限**：仅 `admin`
- 支持创建、编辑（角色/密码）、禁用、删除本地用户
- OIDC 用户首次登录自动创建，管理员可调整角色
- 字段：`username`、`email`、`role`、`enabled`、`auth_type`（`local` / `oidc`）
- 不可删除最后一个 `admin` 用户

### 4.9 OpenAPI 与 API Token（F-13）

#### OpenAPI / Swagger

| 资源 | 路径 | 权限 |
|------|------|------|
| OpenAPI 规范 | `GET /api/openapi.yaml` | 公开 |
| Swagger UI | `GET /api/docs` | 公开 |

- 规范版本与 `info.version` 同步（当前 1.4.0）
- 所有受保护接口标注 `security: BearerAuth`
- Swagger UI 支持「Authorize」填入 Bearer Token 后直接调试

#### API Token 生命周期

| 阶段 | 行为 |
|------|------|
| 创建 | `POST /api/tokens`，仅 `admin`；返回完整 `token`（**仅一次**） |
| 存储 | 服务端仅存 SHA-256 哈希 + 前 8 位前缀（用于列表展示） |
| 格式 | `kkcert_` + 32 字节随机 hex |
| 认证 | `Authorization: Bearer kkcert_xxx...`，权限等同创建时指定的 `role` |
| 使用记录 | 每次认证成功更新 `last_used_at` |
| 吊销 | `DELETE /api/tokens/{id}`，立即失效 |
| 过期 | 创建时可设 `expires_days`（0 = 永久）；过期 Token 由定期清理任务删除 |

#### Token 管理字段

| 字段 | 说明 |
|------|------|
| `name` | 用途标识，如 `cursor-agent` |
| `role` | `admin` / `operator` / `viewer` |
| `expires_days` | 有效天数，0 表示永久 |
| `prefix` | 列表展示用，如 `kkcert_a1b2c3d4` |
| `last_used_at` | 最后使用时间 |

#### Web 管理

- 管理员侧边栏「API Token」页面：创建、列表、吊销
- 页面提供 Swagger 文档外链

#### AI Agent 接入指南

**推荐流程：**

```
1. 管理员登录 Web → API Token → 创建 Token（建议 operator 角色）
2. 复制返回的 kkcert_xxx...（仅显示一次）
3. Agent 配置环境变量 KKCERT_TOKEN=kkcert_xxx...
4. 所有请求携带 Header: Authorization: Bearer $KKCERT_TOKEN
5. 参考 GET /api/docs 或 GET /api/openapi.yaml 调用接口
```

**常用 Agent 接口：**

| 场景 | 方法 | 路径 |
|------|------|------|
| 健康检查 | GET | `/api/health` |
| 查看证书状态 | GET | `/api/certificates` |
| 添加域名 | POST | `/api/domains` |
| 手动续签 | POST | `/api/domains/{id}/renew` |
| 查看操作日志 | GET | `/api/logs` |
| 触发到期检测 | POST | `/api/check/run` |

**示例（curl）：**

```bash
# 创建 Token（需 admin Session）
curl -X POST http://localhost:8080/api/tokens \
  -H "Authorization: Bearer <session-token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-agent","role":"operator","expires_days":90}'

# 使用 API Token 查询证书
curl http://localhost:8080/api/certificates \
  -H "Authorization: Bearer kkcert_..."
```

### 4.10 主题与界面（F-11）

#### 主题模式

| 模式 | 说明 |
|------|------|
| 浅色（light） | 靛蓝渐变侧栏 + 浅色主内容区，适合日间使用 |
| 深色（dark） | 深蓝黑侧栏 + 深色主内容区，适合夜间使用 |

#### 行为

- 登录页右上角提供主题切换按钮
- 已登录后：主内容区右上角用户下拉菜单，内含主题切换与退出登录
- 用户选择存储于 `localStorage`（`kkcert_theme`）
- 首次访问：读取 `prefers-color-scheme`，无偏好时默认浅色
- 切换时同步 `document.documentElement[data-theme]`，避免闪烁

#### 视觉规范

- 字体：Plus Jakarta Sans（Google Fonts）
- 使用 CSS 变量统一管理颜色，禁止硬编码色值
- 主色：靛蓝渐变（`--accent` → `--accent-2`），点缀色青绿（`--accent-3`）
- 布局：深色渐变侧栏 + 浅色/深色主内容区，径向光晕背景
- 侧栏：分组导航（监控 / 管理）+ SVG 图标
- 顶栏：主内容区右上角用户头像下拉（用户名、角色、主题、退出）
- 页脚：所有页面（含登录页）底部统一展示「系统运维驱动」（`PageFooter` 组件）
- 统计卡片：图标 + 数值 + 状态色光晕
- 登录页：双栏布局（品牌介绍 + 表单），移动端折叠为单栏
- 动态背景：登录页与后台主内容区使用**运维拓扑风格**轻量动画（正交网格、链路脉冲、节点状态闪烁）；侧栏光晕仅透明度呼吸，无位移
- 登录页 Logo 图标**静态**展示，不做浮动动画
- 无障碍：`prefers-reduced-motion` 时关闭全部背景动画
- 卡片圆角 16px，表格行 hover，表单 focus 环，按钮渐变；卡片半透明毛玻璃
- 响应式：≤768px 侧栏折叠为顶部横向导航

#### 页面布局原则

各页面**禁止**将「创建表单 + 数据列表 + 多块配置」纵向堆叠在同一屏；主内容区以列表/概览为主，次要操作通过侧滑面板（`Sheet`）或标签页（`Tabs`）承载。

| 页面 | 主内容 | 次要操作 |
|------|--------|----------|
| 概览 | 统计卡片 + **需关注**证书（warning/expired） | 「立即检测」；完整列表跳转证书列表 |
| 域名管理 | 域名表格 | 「添加域名」→ 侧滑面板 |
| 证书列表 | 证书表格 | 「一键同步 Git」在页头 |
| 操作日志 | 日志表格 | — |
| 系统设置 | **标签页**切换单模块配置（ACME / SSO / GoDaddy / Git / 续签） | 每页底部统一「保存设置」 |
| 用户管理 | 用户表格 | 「新建用户」→ 侧滑面板 |
| API Token | Token 表格 | 「创建 Token」→ 侧滑面板 |

- 侧滑面板：`frontend/src/components/Sheet.tsx`，Esc / 遮罩关闭
- 设置标签：`frontend/src/components/Tabs.tsx`

#### 域名续签状态条

- 位置：域名管理页列表上方（`PageHeader` 下方）
- 与 DNS 传播等待对齐：进行中提示「1～5 分钟」，轮询超时 12 分钟
- 样式复用 `notice` / `notice-info` / `notice-ok` / `notice-err`

#### 品牌标识与图标

| 类型 | 组件 / 文件 | 用途 |
|------|-------------|------|
| Logo | `frontend/src/Logo.tsx` | 盾形 + **K** 字，品牌唯一标识 |
| Favicon | `frontend/public/favicon.svg` | 与 Logo 同图形，渐变描边 |
| 导航图标 | `frontend/src/icons.tsx` | 功能入口，与 Logo 同套描边风格 |

**一致性要求：**

- 侧栏品牌区、登录页主视觉、favicon **必须**使用同一 `Logo` 图形（盾 + K），禁止混用字母「K」或其它图标
- 导航与统计区使用功能图标（仪表盘、地球、证书文档等），**不得**复用 Logo 图形
- 全部 SVG 统一：`viewBox="0 0 24 24"`、`strokeWidth=2`、圆角端点（`strokeLinecap/join: round`）
- Logo 容器：渐变背景圆角方块（`brand-icon` / `login-hero-icon`），图标颜色 `#fff`

### 4.11 系统设置

| 配置项 | 必填 | 说明 |
|--------|------|------|
| `acme_email` | 是 | Let's Encrypt 注册邮箱 |
| `acme_staging` | 否 | 是否使用测试环境（默认 false） |
| `godaddy_api_key` | 是 | GoDaddy API Key |
| `godaddy_api_secret` | 是 | GoDaddy API Secret |
| `git_repo_url` | 是 | 远程仓库地址 |
| `git_branch` | 否 | 默认 `main` |
| `git_auth_type` | 是 | `ssh` / `token` |
| `git_ssh_key_path` | 条件 | SSH 私钥路径 |
| `git_token` | 条件 | HTTPS Token |
| `git_certs_dir` | 否 | 仓库内证书目录，默认 `certs` |
| `renew_before_days` | 否 | 默认 30 |
| `check_cron` | 否 | 默认 `0 3 * * *` |
| `cleanup_cron` | 否 | 默认 `0 4 * * *` |
| OIDC 相关 | 条件 | 见 4.7 |

- **权限**：读取 `admin` / `operator` / `viewer`；写入仅 `admin`

## 5. 非功能需求

| 类别 | 要求 |
|------|------|
| 可用性 | 单实例部署，进程重启后任务与数据不丢失 |
| 安全 | 密码 bcrypt；Session 随机 Token；API Token 仅存哈希；OIDC state 防 CSRF；私钥不落日志 |
| 性能 | 100 域名规模下日检 < 5 分钟 |
| 可观测 | 结构化日志；前端展示最近 50 条操作记录 |
| 可集成 | OpenAPI 3.0 完整描述；Swagger UI 可在线调试 |

## 6. 接口概要

完整定义见 [`backend/internal/api/openapi.yaml`](../backend/internal/api/openapi.yaml)（源码）与在线 `GET /api/openapi.yaml`、`GET /api/docs`。

### 6.1 公开接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/health` | 健康检查 |
| GET | `/api/openapi.yaml` | OpenAPI 规范 |
| GET | `/api/docs` | Swagger UI |
| POST | `/api/auth/login` | 本地登录 |
| GET | `/api/auth/oidc/login` | 跳转 OIDC |
| GET | `/api/auth/oidc/callback` | OIDC 回调 |

### 6.2 认证

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/auth/me` | 已登录 | 当前用户 |
| POST | `/api/auth/logout` | 已登录 | 登出 |

### 6.3 API Token（F-13）

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET | `/api/tokens` | admin | 列出 Token（不含密钥） |
| POST | `/api/tokens` | admin | 创建 Token，响应含一次性 `token` |
| PUT | `/api/tokens/{id}` | admin | 更新名称/角色 |
| DELETE | `/api/tokens/{id}` | admin | 吊销 Token |

### 6.4 业务接口

| 方法 | 路径 | 权限 | 说明 |
|------|------|------|------|
| GET/POST/PUT/DELETE | `/api/users` | admin | 用户管理 |
| GET/POST | `/api/domains` | 读:all / 写:admin,operator | 域名 |
| DELETE | `/api/domains/{id}` | admin,operator | 删除域名 |
| POST | `/api/domains/{id}/renew` | admin,operator | 手动续签（异步，返回 `started`） |
| GET | `/api/certificates` | 已登录 | 证书列表 |
| POST | `/api/certificates/sync-git` | admin,operator | 一键将全部有效证书 push 到 Git |
| GET/PUT | `/api/settings` | 读:all / 写:admin | 设置 |
| POST | `/api/settings/acme/reset` | admin | 重置 ACME 账户 |
| GET | `/api/logs` | 已登录 | 操作日志 |
| POST | `/api/check/run` | admin,operator | 手动触发检测 |
| POST | `/api/cleanup/run` | admin | 手动触发数据清理 |

### 6.5 认证 Header

所有受保护接口：

```
Authorization: Bearer <session-token | kkcert_... | legacy-api-token>
```

## 7. 部署

### 7.1 部署方式

| 方式 | 说明 |
|------|------|
| **Docker Compose（推荐）** | `Rakefile` + `docker-compose.yml`，一键构建运行 |
| 单二进制 | `go build` 产物内嵌前端静态资源，适合无 Docker 环境 |

- 镜像多阶段构建：Node 构建前端 → Go 编译后端 → Alpine 运行镜像
- 运行镜像包含：`ca-certificates`、`git`、`openssh-client`（Git 同步需要）
- 数据持久化：宿主机 `./data/` 挂载至容器 `/data`（bbolt、Git 工作区、SSH 密钥）

### 7.2 Docker Compose 运行与调测

**配置文件：**

| 文件 | 说明 |
|------|------|
| `docker-compose.yml` | 服务定义、端口、卷、环境变量 |
| `Dockerfile` | 多阶段镜像构建 |
| `Rakefile` | 常用命令封装 |

**常用命令（`rake`）：**

| 命令 | 等价 | 说明 |
|------|------|------|
| `rake run` / `rake up` | `docker compose up -d --build` | 构建并后台启动 |
| `rake build` | `docker compose build` | 仅构建镜像 |
| `rake down` / `rake stop` | `docker compose down` | 停止并移除容器 |
| `rake logs` | `docker compose logs -f` | 跟踪容器日志 |
| `rake restart` | `docker compose up -d --build --force-recreate` | 强制重建容器 |
| `rake ps` | `docker compose ps` | 查看容器状态 |
| `rake dev` | `docker compose up --build` | 前台运行（调试用） |

**调测流程：**

```bash
# 1. 启动
rake run

# 2. 健康检查
curl http://localhost:8080/api/health
# 期望: {"status":"ok"}

# 3. 查看日志（续签/ACME 错误排查）
rake logs

# 4. 停止
rake down
```

**访问地址：**

| 入口 | URL |
|------|-----|
| Web 管理后台 | http://localhost:8080 |
| Swagger 文档 | http://localhost:8080/api/docs |
| OpenAPI 规范 | http://localhost:8080/api/openapi.yaml |

默认账号：`admin` / `changeme`（或 `KKCERT_BOOTSTRAP_PASSWORD` 指定值）。

### 7.3 本地开发（前后端分离）

适用于前端 UI 热更新调试，后端仍跑在容器内：

```bash
# 终端 1：Docker 启动后端
rake up

# 终端 2：Vite 前端（代理 /api → :8080）
rake dev-frontend
```

前端开发服务器默认将 `/api` 代理到 `http://localhost:8080`。

### 7.4 环境变量

容器内通过 `docker-compose.yml` 注入，可在项目根目录 `.env` 中覆盖：

| 变量 | 默认 | 说明 |
|------|------|------|
| `KKCERT_DATA_DIR` | `/data`（容器内） | 数据目录；Compose 挂载 `./data` |
| `KKCERT_LISTEN` | `:8080` | 监听地址 |
| `KKCERT_BOOTSTRAP_PASSWORD` | `changeme` | 首次 admin 密码 |
| `KKCERT_BASE_URL` | 自动推断 | OIDC 回调基础 URL |

示例 `.env`：

```bash
KKCERT_BOOTSTRAP_PASSWORD=your-secure-password
```

Agent 侧建议配置：

| 变量 | 说明 |
|------|------|
| `KKCERT_BASE_URL` | 服务地址，如 `http://localhost:8080` |
| `KKCERT_TOKEN` | API Token（`kkcert_` 前缀） |

## 8. 验收标准

- [ ] 证书列表页头「一键同步 Git」可将全部有效证书单次 push，操作日志有 `pushed N certificates`
- [ ] 不同根域名可同时续签，各自 `_acme-challenge` 互不干扰；同一域名重复点击被互斥
- [ ] 多域名并行续签若 Git push 失败，一键同步 Git 可补推全部证书
- [ ] 点击「申请/续签」后状态条依次显示提交中 → 进行中，不误展示历史错误
- [ ] 续签成功状态条显示过期日期；失败显示 ACME 错误；超过 12min 显示超时提示
- [ ] 操作日志可见 `renew / started` 及终态记录
- [ ] `rake run` 后 `curl /api/health` 返回 ok，Web 可登录
- [ ] `rake logs` 可查看续签与 ACME 日志
- [ ] 未登录访问受保护 API 返回 401
- [ ] 本地登录与 OIDC 登录均可获取会话
- [ ] `viewer` 无法修改域名或设置
- [ ] 管理员可管理用户与角色
- [ ] 服务重启后 ACME 账户复用，不重复注册
- [ ] 添加域名后可成功申请证书（staging 可验证）
- [ ] 剩余天数 ≤ 阈值时自动续签
- [ ] 证书文件成功 push 到 Git 仓库
- [ ] 续签成功后数据库仅保留该域名最新一份证书
- [ ] 定期清理任务可清除归档域名证书、过期会话与过期 API Token
- [ ] `GET /api/docs` 可打开 Swagger UI 并调试接口
- [ ] 管理员可创建 API Token，Agent 可用 Bearer Token 调用 API
- [ ] 吊销 Token 后立即无法认证

## 9. 风险与假设

| 项 | 说明 |
|----|------|
| GoDaddy API 限流 | 定时续签串行、间隔 10s；手动多域名并行可能触发限流，宜错开 |
| 多域名并行 Git | 自动 push 共用工作区，可能冲突；用「一键同步 Git」补推 |
| GoDaddy DNS TTL | TXT 记录 TTL 不得小于 600 秒；后端使用 lego 默认配置（600） |
| Let's Encrypt 频率限制 | 续签防抖 24h；ACME 账户持久化降低注册频率 |
| OIDC | IdP 需支持 standard OIDC Discovery |
| API Token 泄露 | Token 仅创建时显示一次；建议设有效期；最小权限原则 |
| 假设 | 根域名 DNS 托管在 GoDaddy |
