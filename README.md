# SoulCourse

- 前端：Vue + Vite，开发端口 `5712`
- 管理后台：并入前端静态资源，访问 `http://localhost:5712/admin/`
- 后端：Go + Gin + SQLite，API 端口 `1309`
- 存储：`backend/data/soulcourse.db`
- 上传目录：`backend/data/uploads`

## 目录说明

```text
frontend/                 Vue 前端
frontend/public/admin/    合并后的管理后台静态资源
backend/main.go           后端主入口
backend/internal/         配置、HTTP、服务、SQLite 仓储
```

## 启动方式

### 1. 启动后端

```bash
cd backend
go mod tidy
go run .
```

后端默认监听：

- `http://localhost:1309`
- 健康检查：`http://localhost:1309/healthz`
- API 前缀：`http://localhost:1309/api/v1`

### 2. 启动前端

```bash
cd frontend
pnpm install
pnpm dev
```

前端默认监听：

- 用户端：`http://localhost:5712`
- 管理端：`http://localhost:5712/admin/`

Vite 已经代理：

- `/api` -> `http://localhost:1309`
- `/uploads` -> `http://localhost:1309`

## 环境变量

根目录复制一份：

```bash
cp .env.example .env
```

核心变量：

```env
HTTP_PORT=1309
SQLITE_PATH=data/soulcourse.db
MEDIA_UPLOAD_DIR=data/uploads
CORS_ALLOWED_ORIGINS=http://localhost:5712,http://127.0.0.1:5712
ADMIN_TOKEN=replace-me-for-admin-panel
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=admin_dev_password
JWT_SECRET=replace-me-before-production
```

## 说明

第一次启动后端时会自动创建 SQLite 数据库，并写入一批基础帖子、评论、洞察、话题和后台内容记录。
