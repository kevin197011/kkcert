# KKCert

根域名 TLS 证书自动申请、续签与 Git 同步运维平台。

## 功能

- 批量管理根域名，通过 GoDaddy DNS-01 申请 Let's Encrypt 免费证书（90 天）
- 每日定时检测证书剩余天数，到期前自动续签
- 证书自动 commit & push 到 Git 仓库
- Web 管理后台（本地登录 + OIDC SSO）
- 基于角色的权限控制（admin / operator / viewer）
- ACME 账户持久化，避免重复注册

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.22+ |
| 前端 | React + Vite + TypeScript |
| 存储 | bbolt |

## 快速开始

### 构建与运行（Docker Compose）

```bash
rake run          # 构建镜像并重启容器
```

访问 http://localhost:8080 ，默认账号 `admin` / `changeme`（可通过 `KKCERT_BOOTSTRAP_PASSWORD` 修改）

### 开发模式

```bash
# 终端 1：Docker 启动后端
rake run

# 终端 2：前端热更新（代理 /api 到 8080）
cd frontend && npm run dev
```

## 配置

首次启动后访问 **系统设置** 页面配置：

1. **ACME** — Let's Encrypt 注册邮箱（建议先开启 Staging 测试）
2. **GoDaddy** — API Key / Secret（DNS 托管须在 GoDaddy）
3. **Git** — 仓库地址、鉴权方式（SSH 或 Token）
4. **续签策略** — 提前续签天数（默认 30）、检测 Cron（默认 `0 3 * * *`）

环境变量：

| 变量 | 默认 | 说明 |
|------|------|------|
| `KKCERT_DATA_DIR` | `./data` | 数据目录 |
| `KKCERT_LISTEN` | `:8080` | 监听地址 |
| `KKCERT_BOOTSTRAP_PASSWORD` | `changeme` | 首次 admin 密码 |
| `KKCERT_BASE_URL` | 自动推断 | OIDC 回调基础 URL |

## Git 仓库结构

```
certs/
  example.com/
    fullchain.pem
    privkey.pem
    metadata.json
```

## Docker

与 `rake run` 相同：

```bash
docker compose up -d --build --force-recreate
```

## 文档

详见 [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md)
