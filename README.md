<p align="center">
  <picture>
    <img alt="选科π" src="frontend/public/brand/logo-full.png" width="140">
  </picture>
</p>

<h1 align="center">选科π · SoulCourse</h1>

<p align="center">
  <strong>The Subject Selection Knowledge Forum for China's New Gaokao</strong>
</p>

<p align="center">
  <a href="https://vuejs.org"><img src="https://img.shields.io/badge/vue-3.5-%234FC08D?logo=vuedotjs" alt="Vue 3.5"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-1.26-%2300ADD8?logo=go" alt="Go 1.26"></a>
  <a href="https://www.postgresql.org"><img src="https://img.shields.io/badge/postgresql-17-%234169E1?logo=postgresql" alt="PostgreSQL 17"></a>
  <a href="https://redis.io"><img src="https://img.shields.io/badge/redis-8-%23DC382D?logo=redis" alt="Redis 8"></a>
  <a href="https://www.docker.com"><img src="https://img.shields.io/badge/docker-compose-%232496ED?logo=docker" alt="Docker"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License"></a>
</p>

<p align="center">
  <a href="#-introduction-en">English</a> ·
  <a href="#-介绍-zh">中文</a> ·
  <a href="#-はじめに-ja">日本語</a> ·
  <a href="#-소개-ko">한국어</a>
</p>

---

## 🇬🇧 Introduction (EN)

**SoulCourse** (选科π) is a full-stack forum application designed for Chinese high school students and their parents navigating the **New Gaokao** (新高考) subject selection reform. Under China's reformed college entrance exam system, students must choose elective subject combinations — typically physics- or history-track with two additional electives from chemistry, biology, politics, and geography — and these choices directly determine which university majors they qualify for.

The forum provides a community space for sharing experiences, asking questions, exploring data-driven insights, accessing province-specific policy knowledge, and receiving AI-powered personalized subject combination advice.

### Why This Project Exists

Every year, over 10 million Chinese students face the gaokao. Under the new system, subject selection is a high-stakes decision made in the first year of high school — yet most students and parents lack structured, trustworthy guidance. **SoulCourse** bridges this gap by combining peer experience sharing, official data aggregation, and AI-assisted decision support in a single platform.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Browser                            │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                Vue 3 SPA (Vite + TypeScript)                ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐  ││
│  │  │  Pinia   │  │TanStack  │  │  Axios   │  │  Naive UI  │  ││
│  │  │  Store   │  │  Query   │  │  Client  │  │ Components │  ││
│  │  └──────────┘  └──────────┘  └─────┬────┘  └────────────┘  ││
│  └────────────────────────────────────┼────────────────────────┘│
└───────────────────────────────────────┼─────────────────────────┘
                                        │ HTTP / JSON
┌───────────────────────────────────────┼─────────────────────────┐
│                          Docker Host  │                          │
│  ┌────────────────────────────────────┴────────────────────────┐│
│  │              Nginx (SPA static serving, :80)                 ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                   Go API Server (:8080)                      ││
│  │  ┌──────────┐  ┌──────────────┐  ┌───────────────────────┐  ││
│  │  │   Gin    │  │  Service     │  │  DeepSeek AI          │  ││
│  │  │  Router  │──│  Layer       │──│  (Choice Advice)      │  ││
│  │  │  + JWT   │  │  (business   │  └───────────────────────┘  ││
│  │  │  Auth    │  │   logic)     │                               ││
│  │  └──────────┘  └──────┬───────┘                               ││
│  │                       │                                       ││
│  │              ┌────────┴────────┐                              ││
│  │              │   Repository    │                              ││
│  │              │   (pgx / SQL)   │                              ││
│  │              └────────┬────────┘                              ││
│  └───────────────────────┼───────────────────────────────────────┘│
│                          │                                        │
│  ┌───────────────────────┼───────────────────────────────────────┐│
│  │         ┌─────────────┴──────────────┐                        ││
│  │         │     PostgreSQL 17 Alpine   │    Redis 8 Alpine      ││
│  │         │     (primary data store)   │    (session / cache)   ││
│  │         └────────────────────────────┘                        ││
│  └───────────────────────────────────────────────────────────────┘│
└───────────────────────────────────────────────────────────────────┘
```

**Clean layered backend:**

```
cmd/api/main.go          → entry point, dependency injection
  ├── internal/config/    → environment-based configuration
  ├── internal/storage/   → PostgreSQL & Redis connection pools
  ├── internal/domain/    → domain types & business constants
  ├── internal/repository/ → data access (raw SQL via pgx)
  ├── internal/service/   → business logic, auth, AI integration
  └── internal/http/      → HTTP layer
        ├── handler/      → request handlers (thin controllers)
        └── middleware/   → JWT auth, logging, security headers
```

**Component-driven frontend:**

```
src/
  ├── types/         → TypeScript type definitions
  ├── lib/           → API client, labels, curated reference content
  ├── stores/        → Pinia stores (auth, filters, preferences)
  ├── composables/   → TanStack Query hooks + API state
  ├── components/    → reusable UI components (TopNav, FilterRail, PostCard, …)
  ├── pages/         → 18 route-level page components
  └── router/        → Vue Router configuration
```

### Key Features

| Feature | Description |
|---------|-------------|
| **Subject Combination Filtering** | Filter posts by track (physics / history) and up to 2 elective subjects |
| **Forum Feed** | Browse posts sorted by Recommended (custom scoring), Latest, or Hot |
| **Post Categories** | Experience sharing, parent questions, and data-driven suggestions |
| **Comments & Interactions** | Nested comments, likes, favorites, and author follows |
| **Subject Insights** | Pre-computed data on combination popularity, heat, match rates, and expert advice |
| **Email Verification** | Registration requires a 6-digit email verification code, with SMTP-ready delivery |
| **Enterprise Admin Panel** | Separate admin console for content publishing, policy library, categories, requirements, trends, users, SMTP checks, and audit logs |
| **Discussion Topics** | Curated topics (e.g., "How to choose physics track", "Chemistry importance") |
| **User Authentication** | JWT-based registration & login with roles: student, parent, teacher, counselor |
| **AI Choice Advice** | DeepSeek-powered personalized subject selection advice with backend rule-based degradation when AI is not configured |
| **Knowledge Base** | Province-specific gaokao reform info, official policy documents, major requirements |
| **Real Province Data** | Actual statistics from Zhejiang, Shanghai, Gansu education exam authorities |
| **API-Connected UX** | Login, registration, publishing, comments, likes, favorites, follows, and admin edits require backend persistence |
| **Responsive Design** | Works on desktop, tablet, and mobile browsers |

### Enterprise Admin Panel & Email Verification

The user-facing Vite client runs on `5173`. The admin panel is intentionally separate and runs on `5175` in development:

```bash
cd admin
python3 -m http.server 5175 --bind 127.0.0.1
```

Admin operators sign in with an email account and password. Backend admin APIs remain protected internally by `ADMIN_TOKEN`; the login endpoint exchanges valid admin credentials for the API session token used by the console. The admin console manages a unified content ledger for posts, categories, province policy packages, official policy sources, major requirements, subject trends, advice notes, users, and system settings. The console no longer provides a local-only mode: edits, workflow decisions, deletes, and SMTP checks must reach the backend API.

Published content API:

- `GET /api/v1/content?module=policies`

Content management APIs:

- `POST /api/v1/admin/login`
- `GET /api/v1/admin/content`
- `POST /api/v1/admin/content`
- `PUT /api/v1/admin/content/:id`
- `POST /api/v1/admin/content/:id/workflow`
- `DELETE /api/v1/admin/content/:id`
- `GET /api/v1/admin/content-summary`
- `GET /api/v1/admin/audit-logs`

The workflow endpoint records explicit review actions for content publishing, advice-note approval, policy rechecks, and user certification decisions. Risk actions such as rejection, return-to-edit, account freeze, unpublish, and recheck should include an operator note so the audit trail remains closed-loop.

SMTP and verification test APIs:

- `GET /api/v1/admin/email-config`
- `POST /api/v1/admin/email-test`

Registration email verification uses:

- `POST /api/v1/auth/email-verification-code`
- `POST /api/v1/auth/register` with `verificationCode`

Aliyun SMTP can be enabled with environment variables:

```bash
ADMIN_TOKEN=<openssl rand -hex 32>
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=<development password>
ADMIN_PASSWORD_HASH=<bcrypt hash for production>
SMTP_HOST=smtpdm.aliyun.com
SMTP_PORT=465
SMTP_USERNAME=<aliyun smtp username>
SMTP_PASSWORD=<aliyun smtp password>
SMTP_FROM_EMAIL=<verified sender email>
SMTP_FROM_NAME=选科π
SMTP_USE_TLS=true
SMTP_START_TLS=false
EMAIL_VERIFICATION_TTL_MINUTES=10
EMAIL_VERIFICATION_SUBJECT=选科π邮箱验证码
```

### Recommendation Algorithm

The default "Recommended" sort uses a weighted scoring function to promote quality content while preventing viral posts from dominating:

```
score = min(likes, 300) × 0.8
      + min(comments, 80) × 4
      + min(favorites, 120) × 3
      + teacher/counselor author bonus
      + recency decay factor
```

### Technology Stack

**Frontend:**
- [Vue 3](https://vuejs.org) with `<script setup>` composition API
- [Vite 8](https://vite.dev) — build tool & dev server
- [TypeScript 6](https://www.typescriptlang.org) — type safety
- [Pinia 3](https://pinia.vuejs.org) — state management
- [TanStack Query 5](https://tanstack.com/query) — server state & caching
- [Naive UI](https://www.naiveui.com) — component library
- [ECharts 6](https://echarts.apache.org) — data visualizations
- [Lucide Vue](https://lucide.dev) — icon set

**Backend:**
- [Go 1.26](https://go.dev) — language runtime
- [Gin](https://gin-gonic.com) — HTTP framework
- [pgx v5](https://github.com/jackc/pgx) — PostgreSQL driver & connection pool
- [go-redis v9](https://redis.uptrace.dev) — Redis client
- [golang-jwt](https://github.com/golang-jwt/jwt) — JWT authentication
- [zap](https://github.com/uber-go/zap) — structured logging
- [DeepSeek API](https://platform.deepseek.com) — AI-powered advice

**Infrastructure:**
- [Docker Compose](https://docs.docker.com/compose/) — dev & production orchestration
- [PostgreSQL 17 Alpine](https://www.postgresql.org) — primary data store
- [Redis 8 Alpine](https://redis.io) — caching & session store
- [Nginx 1.29 Alpine](https://nginx.org) — production frontend serving
- SQL migration files — schema versioning via `docker-entrypoint-initdb.d`

### Quick Start

**Prerequisites:** Docker ≥ 24.x and Docker Compose ≥ 2.x.

```bash
git clone https://github.com/skyfrostz/SoulCourse.git
cd subject-choice-forum

# Set up environment
cp .env.example .env

# Start everything (PostgreSQL, Redis, Go API, Vue dev server)
docker compose up --build
```

**Access points:**

| Service | URL |
|---------|-----|
| Frontend (Vite dev) | http://localhost:5173 |
| Backend API | http://localhost:8081/api/v1/posts |
| Admin panel | http://localhost:5175 |
| Health check | http://localhost:8081/healthz |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6380 |

**Development without Docker (frontend only):**

```bash
cd frontend
pnpm install
pnpm dev
```

**Development without Docker (backend only, requires Go):**

```bash
cd backend
go mod tidy
go run ./cmd/api
```

### Makefile Commands

```bash
make dev       # Start all services
make infra     # Start PostgreSQL and Redis only
make backend   # Start backend service only
make frontend  # Start frontend service only
make test      # Run go test + pnpm build
make down      # Stop containers
make clean     # Stop and remove data volumes
```

### API Endpoints

All endpoints are under `/api/v1`. Auth: ○ = optional, ● = required.

| Method | Endpoint | Auth | Description |
|--------|----------|:----:|-------------|
| GET | `/healthz` | | Liveness check |
| GET | `/readyz` | | Readiness (DB + Redis) |
| POST | `/auth/register` | | Register user |
| POST | `/auth/login` | | Login |
| GET | `/me` | ● | Get current user profile |
| GET | `/taxonomy` | ○ | Get tracks, subjects, categories |
| GET | `/insights` | ○ | List subject insights |
| GET | `/insights/:id` | ○ | Get insight detail |
| GET | `/topics` | ○ | List discussion topics |
| GET | `/topics/:slug` | ○ | Get topic with posts |
| GET | `/posts` | ○ | List posts (with filters) |
| POST | `/posts` | ● | Create a post |
| GET | `/posts/:id` | ○ | Get post with comments |
| POST | `/posts/:id/comments` | ● | Create a comment |
| POST | `/posts/:id/like` | ● | Toggle like |
| POST | `/posts/:id/favorite` | ● | Toggle favorite |
| POST | `/authors/:name/follow` | ● | Toggle follow author |
| POST | `/ai/choice-advice` | ● | AI-powered advice |

### Production Deployment

See [DEPLOY.md](./DEPLOY.md) for the full production deployment guide, including:
- Server prerequisites and git workflow
- Production environment variable setup
- Multi-stage Docker builds
- Nginx / Caddy reverse proxy configuration with HTTPS
- Database backup procedures
- Security hardening recommendations

Quick production launch:

```bash
cp .env.production.example .env
# Edit .env with your real secrets
docker compose -f docker-compose.prod.yml up -d --build
```

Production Nginx serves the frontend and admin panel as static assets and proxies same-origin `/api/` requests to the Go backend container. Default production ports are:

| Service | Port |
|---------|------|
| User frontend | `${FRONTEND_HOST_PORT:-80}` |
| Admin panel | `${ADMIN_HOST_PORT:-8082}` |
| Backend API | `${HTTP_HOST_PORT:-8080}` |

Before exposing the server publicly, replace `JWT_SECRET`, `ADMIN_TOKEN`, `ADMIN_EMAIL`, `ADMIN_PASSWORD_HASH`, PostgreSQL credentials, CORS origins, SMTP credentials, and `AI_API_KEY` in `.env`. Do not use `ADMIN_PASSWORD` in production unless it is a short-lived bootstrap password.

### Pre-Launch Verification

Latest local pre-launch checks:

- `node --check admin/app.js`
- `go test ./...`
- `pnpm build`
- `docker compose config`
- `docker compose -f docker-compose.prod.yml --env-file .env.production.example config`
- API smoke test: health, posts, topics, insights, admin login/content/SMTP config, email verification, registration, login, `/me`, post creation, comment, like, and favorite.

### Project Status & Roadmap

**Current (v0.x):**
- [x] Full forum CRUD with filtering and sorting
- [x] JWT authentication with role-based access
- [x] AI-powered subject choice advice (DeepSeek)
- [x] Email verification registration and SMTP-ready delivery
- [x] Enterprise admin console with content, policy, advice, user, permission, and audit workflows
- [x] Subject insights and topic discussions
- [x] Province-specific knowledge base
- [x] Docker Compose dev & production environments

**Planned:**
- [ ] Image upload with OSS/S3 integration
- [ ] Full-text search (PostgreSQL tsvector)
- [ ] Notification system (WebSocket)
- [ ] Social OAuth login (WeChat, QQ)
- [ ] Mobile PWA support
- [ ] Automated test suite expansion
- [ ] CI/CD pipeline (GitHub Actions)

### Contributing

Contributions are welcome. Please open an issue first to discuss what you'd like to change.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### License

This project is licensed under the Apache License 2.0. See [LICENSE](./LICENSE) for the full text.

---

## 🇨🇳 介绍 (ZH)

**选科π**是一个面向中国新高考选科的全栈论坛应用，为中国高一学生和家长提供选科交流、数据洞察和 AI 辅助决策服务。

### 项目背景

每年超过一千万中国学生面临高考。在新高考改革下，选科是在高一就要做出的高利害决策——物理/历史二选一，再从化学、生物、政治、地理中选两科——这个组合直接决定大学专业报考资格。然而大多数学生和家长缺乏系统的、可信赖的指导。**选科π**通过将经验分享、官方数据聚合和 AI 辅助决策整合到一个平台，填补了这一信息鸿沟。

### 系统架构

项目采用前后端分离的清洁分层架构：

- **前端**：Vue 3 + Vite + TypeScript + Pinia + TanStack Query + Naive UI + ECharts
- **后端**：Go + Gin + pgx + golang-jwt + zap
- **数据层**：PostgreSQL 17（主存储）+ Redis 8（缓存/会话）
- **AI 服务**：DeepSeek API（选科建议；未配置 AI 时后端提供规则化降级结果）
- **部署**：Docker Compose 编排，Nginx 托管前端静态资源

详细架构图和技术栈说明请参考上方英文部分。

### 主要功能

- **选科组合筛选**：按物理/历史方向 + 两门再选科目过滤帖子
- **论坛信息流**：推荐（加权评分算法）、最新、最热三种排序
- **帖子分类**：经验分享帖、家长提问帖、数据建议帖
- **互动系统**：评论、点赞、收藏、关注作者
- **科目洞察**：组合热度、趋势图、匹配率、专家建议
- **话题讨论**：精选讨论主题，聚合相关帖子
- **用户认证**：JWT 注册/登录，支持学生、家长、教师、咨询师角色
- **邮箱验证**：新用户注册必须提交邮箱验证码，支持阿里云 SMTP 发信
- **企业后台**：独立管理端，支持内容、政策库、建议库、用户认证、权限、SMTP 和审计管理
- **AI 选科建议**：基于 DeepSeek 的个性化选科建议，API 不可用时自动降级
- **知识库**：各省份高考改革信息、官方政策文件、专业选科要求
- **真实数据**：浙江、上海、甘肃等省份教育考试院官方统计数据
- **后端强一致**：登录、注册、发帖、评论、点赞、收藏、关注和后台编辑必须写入后端

### 快速开始

```bash
git clone https://github.com/skyfrostz/SoulCourse.git
cd subject-choice-forum
cp .env.example .env
docker compose up --build
```

启动后访问：
- 前端：http://localhost:5173
- 后端 API：http://localhost:8081/api/v1/posts
- 管理后台：http://localhost:5175
- 健康检查：http://localhost:8081/healthz

不使用 Docker 单独跑前端：

```bash
cd frontend && pnpm install && pnpm dev
```

### 开发命令

```bash
make dev       # 启动全部服务（前端、后端、PostgreSQL、Redis）
make infra     # 仅启动 PostgreSQL 和 Redis
make backend   # 仅启动后端
make frontend  # 仅启动前端
make test      # 运行后端测试 + 前端构建检查
make down      # 停止所有容器
make clean     # 停止并清除数据卷
```

### 推荐算法

默认的"推荐"排序使用加权评分函数：

```
分数 = min(点赞, 300) × 0.8
     + min(评论, 80) × 4
     + min(收藏, 120) × 3
     + 教师/咨询师作者加成
     + 时间衰减因子
```

该算法鼓励高质量内容，同时防止热门帖子过度占据信息流。

### 生产部署

详细部署指南请参考 [DEPLOY.md](./DEPLOY.md)，包括服务器配置、环境变量设置、反向代理配置和运维命令。

快速生产部署：

```bash
cp .env.production.example .env
# 编辑 .env 填入真实密钥
docker compose -f docker-compose.prod.yml up -d --build
```

生产环境中，用户端和管理端 Nginx 会把同源 `/api/` 请求反向代理到后端容器。默认端口：

- 用户端：`${FRONTEND_HOST_PORT:-80}`
- 管理端：`${ADMIN_HOST_PORT:-8082}`
- 后端 API：`${HTTP_HOST_PORT:-8080}`

公开部署前请务必替换 `.env` 中的 `JWT_SECRET`、`ADMIN_TOKEN`、`ADMIN_EMAIL`、`ADMIN_PASSWORD_HASH`、数据库密码、CORS 域名、SMTP 凭据和 `AI_API_KEY`。生产环境优先使用 `ADMIN_PASSWORD_HASH`，不要长期保留明文 `ADMIN_PASSWORD`。

### 路线图

**已完成：**
- [x] 完整的论坛 CRUD 及筛选排序
- [x] JWT 认证及角色管理
- [x] AI 选科建议（DeepSeek）
- [x] 邮箱验证码注册与 SMTP 配置预留
- [x] 企业级管理后台与闭环审核流程
- [x] 科目洞察与话题讨论
- [x] 省份知识库
- [x] Docker Compose 开发与生产环境

**计划中：**
- [ ] 图片上传（OSS/S3）
- [ ] 全文搜索（PostgreSQL tsvector）
- [ ] 通知系统（WebSocket）
- [ ] 社交登录（微信、QQ）
- [ ] PWA 移动端支持
- [ ] 自动化测试扩展
- [ ] CI/CD（GitHub Actions）

### 贡献指南

欢迎贡献代码。请先提出 issue 讨论你想要修改的内容。

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m '添加某某功能'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 开源协议

本项目采用 Apache License 2.0 协议开源。详见 [LICENSE](./LICENSE)。

---

## 🇯🇵 はじめに (JA)

**選科π**（SoulCourse）は、中国の新高考（新ガオカオ）における科目選択を支援するフルスタックフォーラムアプリケーションです。高校1年生とその保護者が、科目選択に関する経験共有、データ分析、AI アドバイスを得られるコミュニティスペースを提供します。

### 技術スタック

| レイヤー | 技術 |
|----------|------|
| フロントエンド | Vue 3, Vite, TypeScript, Pinia, TanStack Query, Naive UI, ECharts |
| バックエンド | Go 1.26, Gin, pgx, golang-jwt, zap |
| データベース | PostgreSQL 17 Alpine |
| キャッシュ | Redis 8 Alpine |
| AI | DeepSeek API（OpenAI 互換） |
| インフラ | Docker Compose, Nginx |

### クイックスタート

```bash
git clone https://github.com/skyfrostz/SoulCourse.git
cd subject-choice-forum
cp .env.example .env
docker compose up --build
```

- フロントエンド: http://localhost:5173
- バックエンド API: http://localhost:8081/api/v1/posts
- ヘルスチェック: http://localhost:8081/healthz

### ライセンス

Apache License 2.0 — 詳細は [LICENSE](./LICENSE) を参照してください。

---

## 🇰🇷 소개 (KO)

**SoulCourse** (选科π)은 중국의 신가오카오(新高考) 과목 선택을 지원하는 풀스택 포럼 애플리케이션입니다. 고등학교 1학년 학생과 학부모가 과목 선택 경험을 공유하고, 데이터 기반 인사이트를 탐색하며, AI 기반 맞춤형 조언을 받을 수 있는 커뮤니티 공간을 제공합니다.

### 기술 스택

| 계층 | 기술 |
|------|------|
| 프론트엔드 | Vue 3, Vite, TypeScript, Pinia, TanStack Query, Naive UI, ECharts |
| 백엔드 | Go 1.26, Gin, pgx, golang-jwt, zap |
| 데이터베이스 | PostgreSQL 17 Alpine |
| 캐시 | Redis 8 Alpine |
| AI | DeepSeek API (OpenAI 호환) |
| 인프라 | Docker Compose, Nginx |

### 빠른 시작

```bash
git clone https://github.com/skyfrostz/SoulCourse.git
cd subject-choice-forum
cp .env.example .env
docker compose up --build
```

- 프론트엔드: http://localhost:5173
- 백엔드 API: http://localhost:8081/api/v1/posts
- 헬스 체크: http://localhost:8081/healthz

### 라이선스

Apache License 2.0 — 자세한 내용은 [LICENSE](./LICENSE)를 참조하세요.

---

<p align="center">
  <sub>Built with ❤️ for students navigating the New Gaokao | Apache 2.0 Licensed</sub>
</p>
