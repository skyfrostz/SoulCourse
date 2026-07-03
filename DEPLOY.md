# subject-choice-forum 部署指南

## 项目架构

```
┌──────────────────────────────────────────────┐
│                  Server                       │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐  │
│  │  Nginx   │  │  Backend │  │  Postgres  │  │
│  │ (Vue SPA)│  │ (Go API) │  │  17 Alpine │  │
│  │  :80     │◄─┤ :8080    │──┤            │  │
│  └──────────┘  └──────────┘  └───────────┘  │
│                       │          ┌────────┐  │
│                       └──────────┤ Redis  │  │
│                                  │ :6379  │  │
│                                  └────────┘  │
└──────────────────────────────────────────────┘
```

## 前置条件

服务器需要安装：

- **Docker** >= 24.x
- **Docker Compose** >= 2.x（推荐使用 `docker compose` 插件）
- **Git**（用于拉取代码）

## 第一步：将代码推送到服务器

将项目推送到 Git 仓库，然后在服务器上克隆：

```bash
# 在本地
git remote add origin <你的仓库地址>
git push -u origin main

# 在服务器上
git clone <你的仓库地址> /opt/subject-choice-forum
cd /opt/subject-choice-forum
```

或者直接使用 rsync/scp 把代码传到服务器。

## 第二步：配置生产环境变量

```bash
cd /opt/subject-choice-forum

# 以模板为起点创建 .env
cp .env.production.example .env
```

编辑 `.env`，将所有 `<...>` 占位符替换为实际值：

```bash
# ⚠️ 这些值必须修改！
JWT_SECRET=<运行 openssl rand -hex 32 生成>
POSTGRES_PASSWORD=<强密码，不要用 forum_dev_password>
CORS_ALLOWED_ORIGINS=https://你的域名.com
AI_API_KEY=<你的 DeepSeek API 密钥>
```

## 第三步：启动服务

```bash
# 构建并启动所有服务（后台运行）
docker compose -f docker-compose.prod.yml up -d --build

# 查看启动日志
docker compose -f docker-compose.prod.yml logs -f

# 查看所有容器状态
docker compose -f docker-compose.prod.yml ps
```

## 第四步：验证部署

```bash
# 检查后端健康状态
curl http://localhost:8080/health

# 检查前端
curl http://localhost/health
```

## 第五步：配置反向代理（可选但推荐）

如果要绑定域名并启用 HTTPS，建议在宿主机上再套一层 Nginx 或用 Caddy。

### 方案 A：宿主机 Nginx

```nginx
# /etc/nginx/sites-available/subject-choice-forum

server {
    listen 443 ssl http2;
    server_name 你的域名.com;

    ssl_certificate     /etc/letsencrypt/live/你的域名.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/你的域名.com/privkey.pem;

    # API 请求转发到后端
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 其他请求转发到前端
    location / {
        proxy_pass http://127.0.0.1:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

server {
    listen 80;
    server_name 你的域名.com;
    return 301 https://$host$request_uri;
}
```

### 方案 B：Caddy（更简单，自动 HTTPS）

```caddyfile
你的域名.com {
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle {
        reverse_proxy 127.0.0.1:80
    }
}
```

配置完 Nginx/Caddy 后，记得在 `.env` 中更新 `CORS_ALLOWED_ORIGINS` 为你的域名。

## 常用运维命令

```bash
# 查看所有容器状态
docker compose -f docker-compose.prod.yml ps

# 查看实时日志
docker compose -f docker-compose.prod.yml logs -f

# 查看特定服务的日志
docker compose -f docker-compose.prod.yml logs -f backend

# 重启某个服务
docker compose -f docker-compose.prod.yml restart backend

# 更新代码后重新构建和部署
git pull
docker compose -f docker-compose.prod.yml up -d --build

# 停止所有服务
docker compose -f docker-compose.prod.yml down

# 停止服务并清空数据卷（⚠️ 会删除数据库数据）
docker compose -f docker-compose.prod.yml down -v

# 备份数据库
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U forum forum > backup_$(date +%Y%m%d).sql
```

## 安全建议

1. **永远不要**将 `.env` 文件提交到 Git 仓库（已在 `.gitignore` 中）
2. 使用强随机 `JWT_SECRET`：`openssl rand -hex 32`
3. PostgreSQL 和 Redis 端口不要暴露到公网（生产 compose 中已注释）
4. 定期更新基础镜像：`docker compose -f docker-compose.prod.yml pull`
5. 配置防火墙只开放 80/443 端口
6. 考虑使用 Docker secrets 存放敏感信息（生产环境建议）

## 本地开发 vs 生产部署的差异

| 项目 | 本地开发 | 生产部署 |
|------|---------|---------|
| 后端构建目标 | `dev`（air 热重载） | `runtime`（编译好的二进制） |
| 前端运行方式 | `pnpm dev`（Vite 开发服务器） | Nginx 托管静态文件 |
| PostgreSQL 端口 | 暴露到宿主机 (5432) | 不对外暴露 |
| Redis 端口 | 暴露到宿主机 (6380) | 不对外暴露 |
| 源码挂载 | 是（修改实时生效） | 否（镜像自包含） |
| 容器重启策略 | 无 | `unless-stopped` |
