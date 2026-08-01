# SoulCourse

- 前端：Vue + Vite，开发端口 `5712`
- 管理后台：并入 Vue 前端，开发时访问 `http://localhost:5712/admin/`
- 后端：Go + Gin；生产 PostgreSQL、本地开发可用 SQLite，API 端口 `1309`
- 日志：Go 后端使用中文彩色日志，格式为 `[时间]...[级别]...[模块]...[操作]...`
- 存储：`backend/data/soulcourse.db`
- 上传目录：`backend/data/uploads`

公测前验收以 [docs/public-beta-readiness.md](docs/public-beta-readiness.md) 为准；未通过 P0/P1 闸门不得开放公测。
当前 API 契约骨架见 [docs/openapi/openapi.yaml](docs/openapi/openapi.yaml)。
PostgreSQL 16 公测 schema 由 goose migration 管理，见 `backend/migrations/postgres/`；生产运行时使用 PostgreSQL repository，SQLite 仅保留本地开发和迁移源用途。
前端契约类型由 OpenAPI 生成：

```bash
go -C backend run ./cmd/openapi-types
```

检查生成产物是否与契约同步：

```bash
cd frontend
pnpm openapi:check
```

## 目录说明

```text
frontend/                 Vue 前端
frontend/dist/            前端构建产物，可由 Go 直接托管
backend/main.go           后端主入口
backend/internal/         配置、HTTP、服务、SQLite/PostgreSQL 仓储
backend/internal/logx/    中文彩色日志实现
backend/migrations/       PostgreSQL 16 goose 目标 schema
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
go build -trimpath -ldflags="-s -w" -o soulcourse .
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

## 2 核 2G 服务器部署

服务器只需要构建目标平台，不必执行跨平台打包：

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm build

cd ../backend
rm -rf internal/http/webdist/dist
mkdir -p internal/http/webdist/dist
cp -R ../frontend/dist/. internal/http/webdist/dist/
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o soulcourse .
```

将 `.env.production.example` 复制为 `.env` 并填写生产密钥后运行：

```bash
GOMEMLIMIT=1536MiB ./soulcourse
```

`GOMEMLIMIT` 给系统和反向代理预留内存；生产应用默认使用托管 PostgreSQL 16（连接池上限 20）和 S3 兼容对象存储。本次预算受限公测可临时使用 SQLite + S3：设置 `DATABASE_DRIVER=sqlite`、`ALLOW_SQLITE_PRODUCTION=true` 和绝对路径 `SQLITE_PATH`。该开关默认关闭，PostgreSQL 迁移与仓储代码保留，后续切换无需重写业务。

建议将 `.env.example` 复制为 `.env` 后，按需修改以下变量：

- `APP_BASE_PATH`
- `HTTP_PORT`
- `TRUSTED_PROXIES`（生产必填）
- `ADMIN_EMAIL`
- `ADMIN_PASSWORD`（仅本地开发）
- `ADMIN_PASSWORD_HASH`（生产必填）
- `JWT_SECRET`
- `METRICS_TOKEN`（生产必填）
- `DATABASE_DRIVER`、`DATABASE_URL` 与数据库连接池/超时；临时 SQLite 公测还需 `ALLOW_SQLITE_PRODUCTION=true` 和 `SQLITE_PATH`
- `STORAGE_DRIVER`、`S3_ENDPOINT`、`S3_BUCKET`、`S3_REGION`、`S3_CDN_BASE_URL`

正式生产默认使用托管 PostgreSQL 16 和 S3 兼容对象存储；本轮不引入 Redis。预算受限时可使用显式 SQLite 公测模式，仍必须使用 S3。部署 PostgreSQL 模式前必须在目标 PostgreSQL 16 实例执行并确认 goose migration：

```bash
export DATABASE_URL='postgres://user:password@host:5432/soulcourse?sslmode=require'
APP_ENV=production GUANGDONG_DATA_YEAR=2026 \
  GOOSE_BIN=/opt/soulcourse/bin/goose deploy/migrate-production.sh
```

数据库工厂会精确校验 goose schema 版本与核心表；空库、落后/超前版本、已回滚版本和缺表都会在监听端口前拒绝启动。生产启动还会执行一次 S3 `HeadBucket` 与 SMTP TLS/AUTH/NOOP 检查，不创建对象或发送邮件。`HeadBucket` 和 SMTP NOOP 只能证明目标凭证/网络/协议可用，不能证明供应商侧的 bucket 生命周期、版本控制、CDN 缓存或投递到收件箱；这些必须在目标环境用下面的命令逐项记录结果。

对象存储上线前（以 AWS CLI 或兼容其 API 的 CLI 为例）执行并保存输出；不要把下面的占位符提交到仓库：

```bash
export AWS_ENDPOINT_URL='https://<s3-endpoint>'
export AWS_DEFAULT_REGION='<region>'
export S3_BUCKET='<bucket>'
aws --endpoint-url "$AWS_ENDPOINT_URL" s3api get-bucket-versioning --bucket "$S3_BUCKET"
aws --endpoint-url "$AWS_ENDPOINT_URL" s3api get-bucket-lifecycle-configuration --bucket "$S3_BUCKET"
aws --endpoint-url "$AWS_ENDPOINT_URL" s3api head-bucket --bucket "$S3_BUCKET"
```

验收要求：版本控制状态为 `Enabled`；生命周期至少覆盖未完成 multipart upload 的清理（不超过 1 天）和上传临时前缀的 30 天回收，并由供应商规则实际确认。公开图片前缀还必须通过 CDN GET 验证，私有原始政策文件必须验证未经签名 URL 不可读；不同 S3 兼容供应商的生命周期字段可能不同，不能仅凭 AWS CLI 成功推断规则已生效。

运行账号需要业务表 DML、`goose_db_version` 只读权限，以及目标 bucket 的 `HeadBucket`/ListBucket、对象前缀 Get/Put 权限；migration 使用独立高权限账号。SMTP 启动检查会产生一次认证审计事件但不会发送邮件，systemd 将启动失败限制为 5 分钟内最多 3 次。

SQLite -> PostgreSQL 演练命令（只读源 SQLite，事务写入目标 PostgreSQL，并输出每张表的行数与 SHA-256 manifest）：

```bash
go -C backend run ./cmd/sqlite-to-postgres \
  -sqlite data/soulcourse.db \
  -postgres "$DATABASE_URL" \
  -manifest ../tmp/sqlite-to-postgres-manifest.json
```

只生成迁移前 manifest、不写入 PostgreSQL：

```bash
go -C backend run ./cmd/sqlite-to-postgres \
  -sqlite data/soulcourse.db \
  -dry-run \
  -manifest ../tmp/sqlite-source-manifest.json
```

## 环境变量

根目录复制一份：

```bash
cp .env.example .env
```

核心变量：

```env
APP_BASE_PATH=
HTTP_PORT=1309
TRUSTED_PROXIES=
SQLITE_PATH=data/soulcourse.db
MEDIA_UPLOAD_DIR=data/uploads
HTTP_MAX_BODY_BYTES=1048576
CORS_ALLOWED_ORIGINS=http://localhost:5712,http://127.0.0.1:5712
FRONTEND_DIST_DIR=
VITE_API_BASE_URL=
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=admin_dev_password
ADMIN_PASSWORD_HASH=
METRICS_TOKEN=
JWT_SECRET=replace-me-before-production
EMAIL_VERIFICATION_COOLDOWN_SECONDS=60
EMAIL_VERIFICATION_EMAIL_HOURLY_LIMIT=5
EMAIL_VERIFICATION_IP_HOURLY_LIMIT=20
EMAIL_VERIFICATION_MAX_VALIDATION_ATTEMPTS=5
```

说明：

- `APP_BASE_PATH` 留空表示挂在站点根路径。
- 如果要挂到 `/subject314`，写成 `APP_BASE_PATH=/subject314`。
- 前端默认会根据 `APP_BASE_PATH` 生成路由和 API 地址，一般不需要单独设置 `VITE_API_BASE_URL`。
- `VITE_API_BASE_URL` 只在你想把 API 单独指到其他地址时再填写。
- 生产环境使用独立管理员 Cookie 会话登录后台，必须配置 `ADMIN_EMAIL` 与 `ADMIN_PASSWORD_HASH`，不要配置明文 `ADMIN_PASSWORD`。
- 生产环境必须配置至少 32 字符的 `JWT_SECRET` 和 `METRICS_TOKEN`；`METRICS_TOKEN` 用于保护 `/metrics`，可用 `openssl rand -hex 32` 生成。
- 生产环境必须配置 `TRUSTED_PROXIES`，只填写你实际信任的 Nginx/负载均衡 IP 或 CIDR。同机 Nginx 通常为 `127.0.0.1,::1`；反向代理需传 `X-Forwarded-For`、`X-Forwarded-Proto` 和 `X-Request-ID`，应用会据此正确计算限流 IP 并发送 HSTS。

其中 `FRONTEND_DIST_DIR` 留空时，后端会自动尝试：

- `frontend/dist`
- `../frontend/dist`

二进制运行时，如果已经使用 `go run ./cmd/release` 构建了内嵌前端版本，可以不再单独提供 `FRONTEND_DIST_DIR`。

## 生产服务安装

生产服务必须使用专用账号和权限为 `0600` 的环境文件，不再从 root 目录读取明文管理员密码：

```bash
sudo useradd --system --home /var/lib/soulcourse --shell /usr/sbin/nologin soulcourse
sudo install -d -o soulcourse -g soulcourse -m 0700 /var/lib/soulcourse /var/lib/soulcourse/uploads
sudo install -d -o root -g soulcourse -m 0750 /etc/soulcourse
sudo install -o root -g soulcourse -m 0600 .env.production.example /etc/soulcourse/soulcourse.env
sudo install -o root -g root -m 0644 deploy/soulcourse.service /etc/systemd/system/soulcourse.service
sudo systemctl daemon-reload
sudo systemctl enable --now soulcourse
```

编辑 `/etc/soulcourse/soulcourse.env`，替换所有占位值后再启动。`deploy/run-production.sh` 会先执行 `deploy/preflight.sh`，弱密钥、明文管理员密码、SMTP 缺失、PostgreSQL/S3 配置错误、缺少前端产物或 root 运行都会阻止服务启动。migration 与广东数据闸门必须在重启应用前独立执行。

发布后必须同时验证存活和依赖就绪：

```bash
curl -fsS http://127.0.0.1:1309/healthz
curl -fsS http://127.0.0.1:1309/readyz
```

本地/测试默认使用 SQLite；生产配置默认要求 PostgreSQL，只有显式 `ALLOW_SQLITE_PRODUCTION=true` 才允许临时 SQLite 公测模式。PostgreSQL repository 已通过真实 PostgreSQL 16 的注册、会话、发帖、评论、互动、举报、私信、通知和列表集成测试，两次 SQLite 快照迁移演练也已通过；托管实例最终切换、完整 HTTP/管理后台回归、PITR 和性能验收仍是上线闸门。

## 说明

第一次启动后端时会自动创建 SQLite 数据库，并写入一批基础帖子、评论、洞察、话题和后台内容记录。
