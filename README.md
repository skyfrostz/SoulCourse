# 选科知谈

面向中国高一学生和家长的选科主题论坛。本地项目已经包含前端、后端、数据库、缓存、迁移数据和 Docker Compose 开发环境。

## 本地路径

```text
/Users/skyfrost/Documents/Codex/2026-06-30/new-chat/subject-choice-forum
```

## 快速启动

```bash
cd /Users/skyfrost/Documents/Codex/2026-06-30/new-chat/subject-choice-forum
cp .env.example .env
docker compose up --build
```

启动后访问：

- 前端：http://localhost:5173
- 后端健康检查：http://localhost:8081/healthz
- API 示例：http://localhost:8081/api/v1/posts
- PostgreSQL：localhost:5432
- Redis：localhost:6380

## 不用 Docker 单独跑前端

当前 Codex 工作区自带 Node 和 pnpm；如果你的终端也安装了 Node/pnpm，可以直接：

```bash
cd frontend
pnpm install
pnpm dev
```

宿主机当前没有检测到 `go` 命令，所以后端默认通过 Docker 跑。若后期想在宿主机直接开发后端，可以安装 Go 后执行：

```bash
cd backend
go mod tidy
go run ./cmd/api
```

## 主要功能

- 选科组合筛选：物理/历史方向，化学、生物、政治、地理再选科目。
- 论坛信息流：经验帖、家长提问、数据建议。
- 帖子详情：评论列表和发表评论。
- 数据建议：组合趋势图、热门话题、精选建议。
- API fallback：前端会优先请求本地 Go API；API 不可用时显示本地种子数据，方便单独改 UI。

## 常用命令

```bash
make dev       # 启动前端、后端、PostgreSQL、Redis
make infra     # 只启动 PostgreSQL 和 Redis
make backend   # 只启动后端服务
make frontend  # 只启动前端服务
make test      # 后端 go test + 前端 pnpm build
make down      # 停止容器
make clean     # 停止并删除本地数据卷
```

也可以使用脚本：

```bash
./scripts/dev.sh
./scripts/infra.sh
./scripts/test.sh
```

## 技术选型

- 前端：Vue 3 + Vite + TypeScript + Pinia + TanStack Query + ECharts
- 后端：Go + Gin + pgx + PostgreSQL + Redis
- 工程：Docker Compose + 数据库迁移 SQL + 分层目录

## 参考官方资料

- Vue Quick Start: https://vuejs.org/guide/quick-start.html
- Vite Guide: https://vite.dev/guide/
- Gin Documentation: https://gin-gonic.com/docs/
- Pinia Documentation: https://pinia.vuejs.org/
- Go Downloads: https://go.dev/dl/
