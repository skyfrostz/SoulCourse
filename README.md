# SoulCourse

- 前端：Vue + Vite，开发端口 `5712`
- 管理后台：并入 Vue 前端，开发时访问 `http://localhost:5712/admin/`
- 后端：Go + Gin + SQLite，API 端口 `1309`
- 日志：Go 后端使用中文彩色日志，格式为 `[时间]...[级别]...[模块]...[操作]...`
- 存储：`backend/data/soulcourse.db`
- 上传目录：`backend/data/uploads`

## 目录说明

```text
frontend/                 Vue 前端
frontend/dist/            前端构建产物，可由 Go 直接托管
backend/main.go           后端主入口
backend/internal/         配置、HTTP、服务、SQLite 仓储
backend/internal/logx/    中文彩色日志实现
backend/internal/http/webdist/
                         内嵌前端资源目录
backend/cmd/release/      一键构建跨平台二进制
```

## 启动方式

### 1. 启动后端

```bash
cd backend
go mod tidy
go run .
```

后端会自动尝试加载以下位置的 `.env`：

- 当前工作目录 `.env`
- 上级目录 `.env`
- 可执行文件同级目录 `.env`
- 可执行文件上级目录 `.env`

后端默认监听：

- `http://localhost:1309`
- 健康检查：`http://localhost:1309/healthz`
- API 前缀：`http://localhost:1309/api/v1`

如果要把整站挂到子路径，例如 `/subject314`，在根目录 `.env` 设置：

```env
APP_BASE_PATH=/subject314
```

设置后，后端入口会变成：

- `http://localhost:1309/subject314`
- 健康检查：`http://localhost:1309/subject314/healthz`
- API 前缀：`http://localhost:1309/subject314/api/v1`
- 上传访问：`http://localhost:1309/subject314/uploads/...`

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

如果根目录 `.env` 设置了 `APP_BASE_PATH=/subject314`，前端开发路由和代理也会自动改到这个基路径：

- 用户端：`http://localhost:5712/subject314/`
- 管理端：`http://localhost:5712/subject314/admin/`
- API：`/subject314/api/v1`
- 上传：`/subject314/uploads/...`

### 3. 可选：由 Go 直接托管前端

```bash
cd frontend
pnpm install
pnpm build

cd ../backend
go run .
```

构建完成后，后端如果能找到 `frontend/dist`，会直接托管前端静态资源并处理 SPA 路由回退：

- 用户端：`http://localhost:1309/`
- 管理端：`http://localhost:1309/admin`
- 其他前端路由：`http://localhost:1309/topics`、`/knowledge/...`

如果设置了 `APP_BASE_PATH=/subject314`，则对应变为：

- 用户端：`http://localhost:1309/subject314/`
- 管理端：`http://localhost:1309/subject314/admin`
- 其他前端路由：`http://localhost:1309/subject314/topics`

如果后端二进制里已经嵌入了前端产物，则不再依赖外部 `frontend/dist`。

## 日志说明

后端日志已经统一为中文彩色格式，示例：

```text
[时间]2026-07-06 00:30:12 [级别]信息 [模块]系统 [操作]后端服务启动 [地址]:1309 [环境]local
[时间]2026-07-06 00:30:18 [级别]信息 [模块]HTTP [操作]请求完成 [方法]GET [路径]/admin [状态]200 [耗时]3ms [IP]127.0.0.1
[时间]2026-07-06 00:30:23 [级别]警告 [模块]HTTP [操作]请求完成 [方法]POST [路径]/api/v1/admin/login [状态]401 [耗时]9ms [IP]127.0.0.1
```

日志特点：

- 全中文字段，便于直接阅读
- 不同级别带颜色区分
- 每一次 HTTP 请求都会记录
- 启动、关闭、数据库连接、静态资源托管、异常恢复都会记录

## 一键构建二进制

### 方式一：直接构建当前平台

```bash
cd backend
go build -o soulcourse .
```

如果已经将前端构建产物复制到 `backend/internal/http/webdist/dist`，该二进制会直接内嵌前端页面。

### 方式二：一键构建 Windows、macOS、Linux 二进制

```bash
cd backend
go run ./cmd/release
```

这个命令会自动完成：

1. 在 `frontend` 执行 `pnpm build`
2. 将前端构建产物同步到 Go 内嵌目录
3. 编译以下平台的后端二进制

- Windows `amd64`
- Windows `arm64`
- macOS `amd64`
- macOS `arm64`
- Linux `amd64`
- Linux `arm64`

输出目录：

```text
release/
  soulcourse-windows-amd64.exe
  soulcourse-windows-arm64.exe
  soulcourse-darwin-amd64
  soulcourse-darwin-arm64
  soulcourse-linux-amd64
  soulcourse-linux-arm64
  .env.example
  README.md
```

这些二进制已经内嵌前端页面，拿到对应系统上直接运行即可。

## 二进制运行

以 Linux/macOS 为例：

```bash
chmod +x soulcourse-linux-amd64
./soulcourse-linux-amd64
```

以 Windows 为例：

```powershell
.\soulcourse-windows-amd64.exe
```

建议将 `.env.example` 复制为 `.env` 后，按需修改以下变量：

- `APP_BASE_PATH`
- `HTTP_PORT`
- `ADMIN_TOKEN`
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`
- `JWT_SECRET`
- `SQLITE_PATH`
- `MEDIA_UPLOAD_DIR`

## 环境变量

根目录复制一份：

```bash
cp .env.example .env
```

核心变量：

```env
APP_BASE_PATH=
HTTP_PORT=1309
SQLITE_PATH=data/soulcourse.db
MEDIA_UPLOAD_DIR=data/uploads
CORS_ALLOWED_ORIGINS=http://localhost:5712,http://127.0.0.1:5712
FRONTEND_DIST_DIR=
VITE_API_BASE_URL=
ADMIN_TOKEN=replace-me-for-admin-panel
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=admin_dev_password
JWT_SECRET=replace-me-before-production
```

说明：

- `APP_BASE_PATH` 留空表示挂在站点根路径。
- 如果要挂到 `/subject314`，写成 `APP_BASE_PATH=/subject314`。
- 前端默认会根据 `APP_BASE_PATH` 生成路由和 API 地址，一般不需要单独设置 `VITE_API_BASE_URL`。
- `VITE_API_BASE_URL` 只在你想把 API 单独指到其他地址时再填写。

其中 `FRONTEND_DIST_DIR` 留空时，后端会自动尝试：

- `frontend/dist`
- `../frontend/dist`

二进制运行时，如果已经使用 `go run ./cmd/release` 构建了内嵌前端版本，可以不再单独提供 `FRONTEND_DIST_DIR`。

## 说明

第一次启动后端时会自动创建 SQLite 数据库，并写入一批基础帖子、评论、洞察、话题和后台内容记录。
